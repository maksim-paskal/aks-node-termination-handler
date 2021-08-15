package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/maksim-paskal/aks-node-termination-handler/pkg/alerts"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/api"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/config"
	log "github.com/sirupsen/logrus"
)

var (
	ctx     = context.Background()
	version = flag.Bool("version", false, "version")
)

func main() {
	flag.Parse()

	if *version {
		fmt.Println(config.GetVersion()) //nolint:forbidigo
		os.Exit(0)
	}

	log.Infof("Starting %s...", config.GetVersion())

	logLevel, err := log.ParseLevel(*config.Get().LogLevel)
	if err != nil {
		log.Fatal(err)
	}

	log.SetLevel(logLevel)

	if logLevel >= log.DebugLevel {
		log.SetReportCaller(true)
	}

	if !*config.Get().LogPretty {
		log.SetFormatter(&log.JSONFormatter{})
	}

	err = config.Check()
	if err != nil {
		log.Fatal(err)
	}

	err = config.Load()
	if err != nil {
		log.Fatal(err)
	}

	log.Debugf("using config:\n%s", config.String())

	err = alerts.InitAlerts()
	if err != nil {
		log.Fatal(err)
	}

	err = api.MakeAuth()
	if err != nil {
		log.Fatal(err)
	}

	azureResource, err := api.GetAzureResourceName(ctx, *config.Get().NodeName)
	if err != nil {
		log.Fatal(err)
	}

	go api.ReadEvents(ctx, azureResource)

	<-ctx.Done()
}
