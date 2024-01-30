/*
Copyright paskal.maksim@gmail.com
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

	"github.com/maksim-paskal/aks-node-termination-handler/pkg/alert"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/api"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/cache"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/config"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/metrics"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/template"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/types"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const eventCacheTTL = 10 * time.Minute

var httpClient = &http.Client{
	Transport: metrics.NewInstrumenter("events", false).InstrumentedRoundTripper(),
}

func ReadEvents(ctx context.Context, azureResource string) {
	log.Infof("Watching for resource in events %s", azureResource)

	nodeEvent := types.EventMessage{
		Type:    "Info",
		Reason:  "ReadEvents",
		Message: "Start to listen events from Azure API",
	}
	if err := api.AddNodeEvent(ctx, &nodeEvent); err != nil {
		log.WithError(err).Error()
	}

	for ctx.Err() == nil {
		stopReadingEvents, err := readEndpoint(ctx, azureResource)
		if err != nil {
			metrics.ErrorReadingEndpoint.WithLabelValues(getsharedMetricsLabels(azureResource)...).Inc()

			log.WithError(err).Error()
		}

		if stopReadingEvents {
			log.Info("Stop reading events")

			return
		}

		log.Debugf("Sleep %s", *config.Get().Period)
		utils.SleepWithContext(ctx, *config.Get().Period)
	}
}

func readEndpoint(ctx context.Context, azureResource string) (bool, error) { //nolint:cyclop,funlen
	reqCtx, cancel := context.WithTimeout(ctx, *config.Get().RequestTimeout)
	defer cancel()

	log.Debugf("read %s", *config.Get().Endpoint)

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, *config.Get().Endpoint, nil)
	if err != nil {
		return false, errors.Wrap(err, "error in http.NewRequestWithContext")
	}

	req.Header.Add("Metadata", "true")

	resp, err := httpClient.Do(req)
	if err != nil {
		return false, errors.Wrap(err, "error in client.Do(req)")
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, errors.Wrap(err, "error in io.ReadAll")
	}

	if len(body) == 0 {
		log.Info("Scheduled events response is empty")
		return false, nil
	}

	message := types.ScheduledEventsType{}

	err = json.Unmarshal(body, &message)
	if err != nil {
		return false, errors.Wrap(err, "error in json.Unmarshal")
	}

	for _, event := range message.Events {
		for _, r := range event.Resources {
			if r == azureResource { //nolint:nestif
				log.Info(string(body))

				if cache.HasKey(event.EventId) {
					log.Infof("Event %s already processed", event.EventId)

					continue
				}

				// add to cache, ignore similar events for 10 minutes
				cache.Add(event.EventId, eventCacheTTL)

				metrics.ScheduledEventsTotal.WithLabelValues(append(getsharedMetricsLabels(azureResource), string(event.EventType))...).Inc() //nolint:lll

				nodeEvent := types.EventMessage{
					Type:    "Warning",
					Reason:  string(event.EventType),
					Message: "Azure API sended schedule event for this node",
				}
				if err := api.AddNodeEvent(ctx, &nodeEvent); err != nil {
					log.WithError(err).Error()
				}

				if config.Get().IsExcludedEvent(event.EventType) {
					log.Infof("Excluded event %s by user config", event.EventType)

					continue
				}

				err := alert.SendALL(ctx, template.MessageType{
					Event:    event,
					Node:     azureResource,
					Template: *config.Get().AlertMessage,
				})
				if err != nil {
					log.WithError(err).Error("error in alerts.Send")
				}

				err = api.DrainNode(ctx, *config.Get().NodeName, string(event.EventType), event.EventId)
				if err != nil {
					return false, errors.Wrap(err, "error in DrainNode")
				}

				return true, nil
			}
		}
	}

	return false, nil
}

func getsharedMetricsLabels(resourceName string) []string {
	return []string{
		*config.Get().NodeName,
		resourceName,
	}
}
