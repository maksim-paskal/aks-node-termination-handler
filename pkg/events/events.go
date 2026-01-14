/*
Copyright paskal.maksim@gmail.com (Original Author 2021-2025)
Copyright github@vince-riv.io (Modifications 2026-present)
Licensed under the Apache License, Version 2.0 (the "License")
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package events

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/vince-riv/aks-node-termination-handler/pkg/cache"
	"github.com/vince-riv/aks-node-termination-handler/pkg/config"
	"github.com/vince-riv/aks-node-termination-handler/pkg/metrics"
	"github.com/vince-riv/aks-node-termination-handler/pkg/types"
	"github.com/vince-riv/aks-node-termination-handler/pkg/utils"
)

const (
	requestTimeout = 10 * time.Second
	readInterval   = 5 * time.Second
	eventCacheTTL  = 10 * time.Minute
)

var httpClient = &http.Client{
	Transport: metrics.NewInstrumenter("events").InstrumentedRoundTripper(),
}

type Reader struct {
	// method of making request
	Method string
	// endpoint to read events
	Endpoint string
	// timeout of making request
	RequestTimeout time.Duration
	// intervals of reading events
	Period time.Duration
	// name of the node
	NodeName string
	// name of the resource to watch
	AzureResource string
	// BeforeReading is a function that will be called before reading events
	BeforeReading func(ctx context.Context) error `json:"-"`
	// EventReceived is a function that will be called when event received
	// return true if you want to stop reading events
	EventReceived func(ctx context.Context, event types.ScheduledEventsEvent) (bool, error) `json:"-"`
}

func NewReader() *Reader {
	return &Reader{
		Method:         http.MethodGet,
		Endpoint:       "http://169.254.169.254/metadata/scheduledevents?api-version=2020-07-01",
		RequestTimeout: requestTimeout,
		Period:         readInterval,
	}
}

func (r *Reader) ReadEvents(ctx context.Context) {
	log.Infof("Start reading events %s", r.String())

	if r.BeforeReading != nil {
		if err := r.BeforeReading(ctx); err != nil {
			log.WithError(err).Error("Error in BeforeReading")
		}
	}

	for ctx.Err() == nil {
		stopReadingEvents, err := r.ReadEndpoint(ctx)
		if err != nil {
			metrics.ErrorReadingEndpoint.WithLabelValues(r.getMetricsLabels()...).Inc()

			log.WithError(err).Error()
		}

		if stopReadingEvents {
			log.Info("Stop reading events")

			return
		}

		utils.SleepWithContext(ctx, r.Period)
	}
}

func (r *Reader) getScheduledEvents(ctx context.Context) (*types.ScheduledEventsType, error) {
	ctx, cancel := context.WithTimeout(ctx, r.RequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, r.Method, r.Endpoint, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error in http.NewRequestWithContext")
	}

	req.Header.Add("Metadata", "true")

	log.WithFields(log.Fields{
		"method":  req.Method,
		"url":     req.URL,
		"headers": req.Header,
	}).Debug("Doing request for getScheduledEvents()")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "error in client.Do(req)")
	}

	defer resp.Body.Close()

	log.Debugf("response status: %s", resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "error in io.ReadAll")
	}

	log.Debugf("response body: %s", string(body))

	if len(body) == 0 {
		log.Warn("Events response is empty")

		return &types.ScheduledEventsType{}, nil
	}

	message := types.ScheduledEventsType{}

	if err := json.Unmarshal(body, &message); err != nil {
		return nil, errors.Wrap(err, "error in json.Unmarshal")
	}

	return &message, nil
}

func (r *Reader) ReadEndpoint(ctx context.Context) (bool, error) {
	message, err := r.getScheduledEvents(ctx)
	if err != nil {
		return false, errors.Wrap(err, "error in getScheduledEvents")
	}

	for _, event := range message.Events {
		for _, resource := range event.Resources {
			if resource == r.AzureResource {
				log.Infof("%+v", message)

				if cache.HasKey(event.EventId) {
					log.Infof("Event %s already processed", event.EventId)

					continue
				}

				// check if NotBefore is too far in the future
				skip, err := r.shouldSkipNotBefore(event)
				if err != nil {
					log.WithError(err).Warn("failed to parse NotBefore, processing event anyway")
				} else if skip {
					continue
				}

				// add to cache, ignore similar events for 10 minutes
				cache.Add(event.EventId, eventCacheTTL)

				metrics.ScheduledEventsTotal.WithLabelValues(append(r.getMetricsLabels(), string(event.EventType))...).Inc()

				if r.EventReceived != nil {
					return r.EventReceived(ctx, event)
				}
			}
		}
	}

	return false, nil
}

func (r *Reader) getMetricsLabels() []string {
	return []string{
		r.NodeName,
		r.AzureResource,
	}
}

func (r *Reader) String() string {
	b, _ := json.Marshal(r) //nolint:errchkjson

	return string(b)
}

// Ping checks connectivity to the instance metadata API endpoint.
func Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, *config.Get().Endpoint, nil)
	if err != nil {
		return errors.Wrap(err, "error creating request")
	}

	req.Header.Add("Metadata", "true")

	log.WithFields(log.Fields{
		"method":  req.Method,
		"url":     req.URL,
		"headers": req.Header,
	}).Debug("Doing request for Ping()")

	resp, err := httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "error connecting to instance metadata API")
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return errors.Errorf("instance metadata API returned status %d", resp.StatusCode)
	}

	return nil
}

// shouldSkipNotBefore checks if the event's NotBefore time is too far in the future.
// Returns true if the event should be skipped (NotBefore exceeds threshold).
func (r *Reader) shouldSkipNotBefore(event types.ScheduledEventsEvent) (bool, error) {
	threshold := *config.Get().NotBeforeThreshold
	if threshold <= 0 {
		return false, nil
	}

	notBeforeTime, err := event.NotBeforeTime()
	if err != nil {
		return false, errors.Wrap(err, "error parsing NotBefore time")
	}

	// empty NotBefore means event has already started
	if notBeforeTime.IsZero() {
		return false, nil
	}

	timeUntilEvent := time.Until(notBeforeTime)
	if timeUntilEvent > threshold {
		log.Debugf("Event %s NotBefore (%s) is %s in the future, exceeds threshold %s, skipping",
			event.EventId, event.NotBefore, timeUntilEvent.Round(time.Second), threshold)

		return true, nil
	}

	return false, nil
}
