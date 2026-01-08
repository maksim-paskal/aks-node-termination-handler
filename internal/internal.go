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
package internal

import (
	"context"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/vince-riv/aks-node-termination-handler/pkg/alert"
	"github.com/vince-riv/aks-node-termination-handler/pkg/api"
	"github.com/vince-riv/aks-node-termination-handler/pkg/cache"
	"github.com/vince-riv/aks-node-termination-handler/pkg/client"
	"github.com/vince-riv/aks-node-termination-handler/pkg/config"
	"github.com/vince-riv/aks-node-termination-handler/pkg/events"
	"github.com/vince-riv/aks-node-termination-handler/pkg/metrics"
	"github.com/vince-riv/aks-node-termination-handler/pkg/template"
	"github.com/vince-riv/aks-node-termination-handler/pkg/types"
	"github.com/vince-riv/aks-node-termination-handler/pkg/web"
	"github.com/vince-riv/aks-node-termination-handler/pkg/webhook"
)

func Run(ctx context.Context) error {
	err := config.Load()
	if err != nil {
		return errors.Wrap(err, "error in config load")
	}

	err = config.Check()
	if err != nil {
		return errors.Wrap(err, "error in config check")
	}

	log.Debugf("using config: %s", config.Get().String())

	retryClient := retryablehttp.NewClient()
	retryClient.HTTPClient.Transport = metrics.NewInstrumenter("webhook").
		WithProxy(*config.Get().WebhookProxy).
		WithInsecureSkipVerify(*config.Get().WebhookInsecure).
		InstrumentedRoundTripper()
	retryClient.RetryMax = *config.Get().WebhookRetries
	webhook.SetHTTPClient(retryClient)

	err = alert.Init()
	if err != nil {
		return errors.Wrap(err, "error in init alerts")
	}

	err = client.Init()
	if err != nil {
		return errors.Wrap(err, "error in init api")
	}

	go cache.SheduleCleaning(ctx)
	go web.Start(ctx)

	if err := startReadingEvents(ctx); err != nil {
		return errors.Wrap(err, "error in startReadingEvents")
	}

	return nil
}

func startReadingEvents(ctx context.Context) error {
	azureResource, err := api.GetAzureResourceName(ctx, *config.Get().NodeName)
	if err != nil {
		return errors.Wrap(err, "error in getting azure resource name")
	}

	eventReader := events.NewReader()
	eventReader.AzureResource = azureResource
	eventReader.Period = *config.Get().Period
	eventReader.Endpoint = *config.Get().Endpoint
	eventReader.RequestTimeout = *config.Get().RequestTimeout
	eventReader.NodeName = *config.Get().NodeName
	eventReader.BeforeReading = func(ctx context.Context) error {
		// add event to node
		if err := api.AddNodeEvent(ctx, "Info", "ReadEvents", config.EventMessageBeforeListen); err != nil {
			return errors.Wrap(err, "error in add node event")
		}

		return nil
	}

	eventReader.EventReceived = func(ctx context.Context, event types.ScheduledEventsEvent) (bool, error) {
		// add event to node
		if err := api.AddNodeEvent(ctx, "Warning", string(event.EventType), config.EventMessageReceived); err != nil {
			return false, errors.Wrap(err, "error in add node event")
		}

		// check if event is excludedm by default Freeze event is excluded
		if config.Get().IsExcludedEvent(event.EventType) {
			log.Infof("Excluded event %s by user config", event.EventType)

			return false, nil
		}

		// send event in separate goroutine
		go func() {
			if err := sendEvent(ctx, event); err != nil {
				log.WithError(err).Error("error in sendEvent")
			}
		}()

		// drain node
		if err := api.DrainNode(ctx, *config.Get().NodeName, string(event.EventType), event.EventId); err != nil {
			return false, errors.Wrap(err, "error in DrainNode")
		}

		return true, nil
	}

	// check for run in synchronous mode or not
	// synchronous mode is used for e2e tests
	if *config.Get().ExitAfterNodeDrain {
		eventReader.ReadEvents(ctx)
	} else {
		go eventReader.ReadEvents(ctx)
	}

	return nil
}

func sendEvent(ctx context.Context, event types.ScheduledEventsEvent) error {
	message, err := template.NewMessageType(ctx, *config.Get().NodeName, event)
	if err != nil {
		return errors.Wrap(err, "error in template.NewMessageType")
	}

	log.Infof("Message: %+v", message)

	message.Template = *config.Get().AlertMessage

	if err := alert.SendTelegram(message); err != nil {
		log.WithError(err).Error("error in alert.SendTelegram")
	}

	if err := webhook.SendWebHook(ctx, message); err != nil {
		log.WithError(err).Error("error in webhook.SendWebHook")
	}

	return nil
}
