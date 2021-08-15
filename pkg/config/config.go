package config

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	azureEndpoint = "http://169.254.169.254/metadata/scheduledevents?api-version=2020-07-01"
	defaultPeriod = 5 * time.Second
)

type Type struct {
	ConfigFile      *string
	LogPretty       *bool
	LogLevel        *string
	DevelopmentMode *bool
	KubeConfigFile  *string
	Endpoint        *string
	NodeName        *string
	Period          *time.Duration
	TelegramToken   *string
	TelegramChatID  *int
	Alert           *string
}

var config = Type{
	ConfigFile:     flag.String("config", getEnvDefault("CONFIG", "config.yaml"), "config file"),
	LogLevel:       flag.String("log.level", "INFO", "log level"),
	LogPretty:      flag.Bool("log.prety", false, "log in text"),
	KubeConfigFile: flag.String("kubeconfig", "", "kubeconfig file"),
	Endpoint:       flag.String("endpoint", azureEndpoint, "scheduled-events endpoint"),
	NodeName:       flag.String("node", os.Getenv("MY_NODE_NAME"), "node to drain"),
	Period:         flag.Duration("period", defaultPeriod, "period to scrape endpoint"),
	TelegramToken:  flag.String("telegram.token", "", "telegram token"),
	TelegramChatID: flag.Int("telegram.chatID", -1, "telegram chatID"),
}

func Check() error {
	if len(*config.NodeName) == 0 {
		return errNoNode
	}

	return nil
}

func Get() *Type {
	return &config
}

func Load() error {
	if len(*config.ConfigFile) == 0 {
		return nil
	}

	configByte, err := ioutil.ReadFile(*config.ConfigFile)
	if err != nil {
		return errors.Wrap(err, "error in ioutil.ReadFile")
	}

	err = yaml.Unmarshal(configByte, &config)
	if err != nil {
		return errors.Wrap(err, "error in yaml.Unmarshal(")
	}

	return nil
}

func String() string {
	out, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Sprintf("ERROR: %t", err)
	}

	return string(out)
}

var gitVersion = "dev"

func GetVersion() string {
	return gitVersion
}

func getEnvDefault(name string, defaultValue string) string {
	r := os.Getenv(name)
	defaultValueLen := len(defaultValue)

	if defaultValueLen == 0 {
		return r
	}

	if len(r) == 0 {
		return defaultValue
	}

	return r
}
