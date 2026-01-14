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
	"os/signal"
	"syscall"
	"time"

	logrushooksentry "github.com/maksim-paskal/logrus-hook-sentry"
	log "github.com/sirupsen/logrus"
	"github.com/vince-riv/aks-node-termination-handler/internal"
	"github.com/vince-riv/aks-node-termination-handler/pkg/config"
)

var version = flag.Bool("version", false, "version")

func main() {
	flag.Parse()

	if *version {
		fmt.Println(config.GetVersion()) //nolint:forbidigo
		os.Exit(0)
	}

	logLevel, err := log.ParseLevel(*config.Get().LogLevel)
	if err != nil {
		log.WithError(err).Fatal()
	}

	log.SetLevel(logLevel)
	log.SetReportCaller(true)

	if !*config.Get().LogPretty {
		log.SetFormatter(&log.JSONFormatter{})
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Infof("Starting %s...", config.GetVersion())
	if *config.Get().DryRun {
		log.Info("DRY RUN MODE ENABLED: No changes will be made to the cluster")
	}

	hook, err := logrushooksentry.NewHook(ctx, logrushooksentry.Options{
		SentryDSN: *config.Get().SentryDSN,
		Release:   config.GetVersion(),
	})
	if err != nil {
		log.WithError(err).Error()
	}

	log.AddHook(hook)

	signalChanInterrupt := make(chan os.Signal, 1)
	signal.Notify(signalChanInterrupt, syscall.SIGINT, syscall.SIGTERM)

	log.RegisterExitHandler(func() {
		cancel()
	})

	go func() {
		select {
		case <-signalChanInterrupt:
			log.Error("Got interruption signal...")
			cancel()
		case <-ctx.Done():
		}
		<-signalChanInterrupt
		os.Exit(1)
	}()

	if !*config.Get().SkipIMDSCheck {
		err = internal.WaitForIMDS(ctx)
		if err != nil {
			log.WithError(err).Fatal()
		}
	} else {
		log.Info("Skipping instance metadata service check (skipIMDSCheck=true)")
	}

	err = internal.Run(ctx)
	if err != nil {
		log.WithError(err).Fatal()
	}

	<-ctx.Done()

	log.Infof("Waiting %s before shutdown...", config.Get().GracePeriod())
	time.Sleep(config.Get().GracePeriod())
}
