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
package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/maksim-paskal/aks-node-termination-handler/pkg/alert"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/config"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/template"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/types"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var client = &http.Client{}

func ReadEvents(ctx context.Context, azureResource string) {
	log.Infof("Watching for resource in events %s", azureResource)

	for {
		if ctx.Err() != nil {
			log.Info("Context canceled")

			return
		}

		stopReadingEvents, err := readEndpoint(ctx, azureResource)
		if err != nil {
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

func readEndpoint(ctx context.Context, azureResource string) (bool, error) {
	log.Debugf("read %s", *config.Get().Endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, *config.Get().Endpoint, nil)
	if err != nil {
		return false, errors.Wrap(err, "error in http.NewRequestWithContext")
	}

	req.Header.Add("Metadata", "true")

	resp, err := client.Do(req)
	if err != nil {
		return false, errors.Wrap(err, "error in client.Do(req)")
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, errors.Wrap(err, "error in io.ReadAll")
	}

	message := types.ScheduledEventsType{}

	err = json.Unmarshal(body, &message)
	if err != nil {
		return false, errors.Wrap(err, "error in json.Unmarshal")
	}

	for _, event := range message.Events {
		for _, r := range event.Resources {
			if r == azureResource {
				log.Info(string(body))

				err := alert.SendALL(ctx, template.MessageType{
					Event:    event,
					Node:     azureResource,
					Template: *config.Get().AlertMessage,
				})
				if err != nil {
					log.WithError(err).Error("error in alerts.Send")
				}

				err = DrainNode(ctx, *config.Get().NodeName, event.EventType, event.EventId)
				if err != nil {
					return false, errors.Wrap(err, "error in DrainNode")
				}

				return true, nil
			}
		}
	}

	return false, nil
}
