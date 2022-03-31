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

	"github.com/maksim-paskal/aks-node-termination-handler/pkg/alert"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/api"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/config"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/web"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func Run(ctx context.Context) error {
	err := config.Check()
	if err != nil {
		return errors.Wrap(err, "error in config check")
	}

	err = config.Load()
	if err != nil {
		return errors.Wrap(err, "error in config load")
	}

	log.Debugf("using config:\n%s", config.String())

	err = alert.Init()
	if err != nil {
		return errors.Wrap(err, "error in init alerts")
	}

	err = api.Init()
	if err != nil {
		return errors.Wrap(err, "error in init api")
	}

	azureResource, err := api.GetAzureResourceName(ctx, *config.Get().NodeName)
	if err != nil {
		return errors.Wrap(err, "error in getting azure resource name")
	}

	go api.ReadEvents(ctx, azureResource)
	go web.Start()

	return nil
}
