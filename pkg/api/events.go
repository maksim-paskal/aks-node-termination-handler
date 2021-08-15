package api

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/maksim-paskal/aks-node-termination-handler/pkg/alerts"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/config"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	ctx               context.Context
	client            = &http.Client{}
	stopReadingEvents = false
)

func ReadEvents(appCtx context.Context, azureResource string) {
	ctx = appCtx

	log.Infof("Watching for resource in events %s", azureResource)

	for {
		if stopReadingEvents {
			log.Info("Stop reading events")
			<-ctx.Done()
		} else {
			err := readEndpoint(azureResource)
			if err != nil {
				log.Error(err)
			}
		}

		time.Sleep(*config.Get().Period)
	}
}

func readEndpoint(azureResource string) error { //nolint:cyclop
	log.Debugf("read %s", *config.Get().Endpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", *config.Get().Endpoint, nil)
	if err != nil {
		return errors.Wrap(err, "error in http.NewRequestWithContext")
	}

	req.Header.Add("Metadata", "true")

	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "error in client.Do(req)")
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "error in ioutil.ReadAll")
	}

	message := types.ScheduledEventsType{}

	err = json.Unmarshal(body, &message)
	if err != nil {
		return errors.Wrap(err, "error in json.Unmarshal")
	}

	if len(message.Events) > 0 { //nolint:nestif
		for _, event := range message.Events {
			for _, r := range event.Resources {
				if r == azureResource {
					log.Info(string(body))

					err := alerts.Send(alerts.TemplateMessageType{
						Event:    event,
						Node:     azureResource,
						Template: *config.Get().Alert,
					})
					if err != nil {
						return errors.Wrap(err, "error in alerts.Send")
					}

					err = DrainNode(ctx, *config.Get().NodeName)
					if err != nil {
						return errors.Wrap(err, "error in DrainNode")
					}

					stopReadingEvents = true
				}
			}
		}
	}

	return nil
}
