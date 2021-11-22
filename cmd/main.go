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
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/maksim-paskal/aks-node-termination-handler/pkg/alerts"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/api"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/config"
	logrushooksentry "github.com/maksim-paskal/logrus-hook-sentry"
	log "github.com/sirupsen/logrus"
)

var (
	ctx     = context.Background()
	version = flag.Bool("version", false, "version")
)

func main() { //nolint:funlen,cyclop
	flag.Parse()

	if *version {
		fmt.Println(config.GetVersion()) //nolint:forbidigo
		os.Exit(0)
	}

	log.Infof("Starting %s...", config.GetVersion())

	logLevel, err := log.ParseLevel(*config.Get().LogLevel)
	if err != nil {
		log.WithError(err).Fatal()
	}

	log.SetLevel(logLevel)

	if logLevel >= log.DebugLevel {
		log.SetReportCaller(true)
	}

	if !*config.Get().LogPretty {
		log.SetFormatter(&log.JSONFormatter{})
	}

	hook, err := logrushooksentry.NewHook(logrushooksentry.Options{
		SentryDSN: *config.Get().SentryDSN,
		Release:   config.GetVersion(),
	})
	if err != nil {
		log.WithError(err).Error()
	}

	log.AddHook(hook)

	err = config.Check()
	if err != nil {
		log.WithError(err).Fatal()
	}

	err = config.Load()
	if err != nil {
		log.WithError(err).Fatal()
	}

	log.Debugf("using config:\n%s", config.String())

	err = alerts.InitAlerts()
	if err != nil {
		log.WithError(err).Fatal()
	}

	err = api.MakeAuth()
	if err != nil {
		log.WithError(err).Fatal()
	}

	azureResource, err := api.GetAzureResourceName(ctx, *config.Get().NodeName)
	if err != nil {
		log.WithError(err).Fatal()
	}

	go api.ReadEvents(ctx, azureResource)

	<-ctx.Done()
}
