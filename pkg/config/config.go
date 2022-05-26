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
package config

import (
	"flag"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
)

const (
	azureEndpoint                 = "http://169.254.169.254/metadata/scheduledevents?api-version=2020-07-01"
	defaultAlertMessage           = "Draining node={{ .Node }}, type={{ .Event.EventType }}"
	defaultPeriod                 = 5 * time.Second
	defaultPodGracePeriodSeconds  = -1
	defaultNodeGracePeriodSeconds = 120
)

var (
	errNoNode             = errors.New("no node name is defined, run with -node=test")
	errChatIDMustBeInt    = errors.New("TelegramChatID must be integer")
	errInvalidTaintEffect = errors.New("TaintEffect must be either NoSchedule, NoExecute or PreferNoSchedule")
)

type Type struct {
	ConfigFile             *string
	LogPretty              *bool
	LogLevel               *string
	DevelopmentMode        *bool
	KubeConfigFile         *string
	Endpoint               *string
	NodeName               *string
	Period                 *time.Duration
	TelegramToken          *string
	TelegramChatID         *string
	AlertMessage           *string
	WebHookInsecure        *bool
	WebHookContentType     *string
	WebHookURL             *string
	WebHookTemplate        *string
	SentryDSN              *string
	WebHTTPAddress         *string
	TaintNode              *bool
	TaintEffect            *string
	PodGracePeriodSeconds  *int
	NodeGracePeriodSeconds *int
}

var config = Type{
	ConfigFile:             flag.String("config", os.Getenv("CONFIG"), "config file"),
	LogLevel:               flag.String("log.level", "INFO", "log level"),
	LogPretty:              flag.Bool("log.pretty", false, "log in text"),
	KubeConfigFile:         flag.String("kubeconfig", "", "kubeconfig file"),
	Endpoint:               flag.String("endpoint", azureEndpoint, "scheduled-events endpoint"),
	NodeName:               flag.String("node", os.Getenv("MY_NODE_NAME"), "node to drain"),
	Period:                 flag.Duration("period", defaultPeriod, "period to scrape endpoint"),
	TelegramToken:          flag.String("telegram.token", os.Getenv("TELEGRAM_TOKEN"), "telegram token"),
	TelegramChatID:         flag.String("telegram.chatID", os.Getenv("TELEGRAM_CHATID"), "telegram chatID"),
	AlertMessage:           flag.String("alert.message", defaultAlertMessage, "default message"),
	WebHookContentType:     flag.String("webhook.contentType", "application/json", "request content-type header"),
	WebHookInsecure:        flag.Bool("webhook.insecure", false, "use insecure tls config"),
	WebHookURL:             flag.String("webhook.url", os.Getenv("WEBHOOK_URL"), "send alerts to webhook"),
	WebHookTemplate:        flag.String("webhook.template", "test", "request body"),
	SentryDSN:              flag.String("sentry.dsn", "", "sentry DSN"),
	WebHTTPAddress:         flag.String("web.address", ":17923", ""),
	TaintNode:              flag.Bool("taint.node", false, "Taint the node before cordon and draining"),
	TaintEffect:            flag.String("taint.effect", "NoSchedule", "Taint effect to set on the node"),
	PodGracePeriodSeconds:  flag.Int("podGracePeriodSeconds", defaultPodGracePeriodSeconds, "grace period is seconds for pods termination"), //nolint:lll
	NodeGracePeriodSeconds: flag.Int("nodeGracePeriodSeconds", defaultNodeGracePeriodSeconds, "maximum time in seconds to drain the node"),  //nolint:lll
}

func Check() error {
	if len(*config.NodeName) == 0 {
		return errNoNode
	}

	if len(*config.TelegramChatID) > 0 {
		if _, err := strconv.Atoi(*config.TelegramChatID); err != nil {
			return errChatIDMustBeInt
		}
	}

	taintEffect := *config.TaintEffect
	if taintEffect != string(corev1.TaintEffectNoSchedule) &&
		taintEffect != string(corev1.TaintEffectNoExecute) &&
		taintEffect != string(corev1.TaintEffectPreferNoSchedule) {
		return errInvalidTaintEffect
	}

	return nil
}

func Get() *Type {
	return &config
}

func Set(specifiedConfig Type) {
	config = specifiedConfig
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
		return errors.Wrap(err, "error in yaml.Unmarshal")
	}

	return nil
}

func String() string {
	out, _ := yaml.Marshal(config)

	return string(out)
}

var gitVersion = "dev"

func GetVersion() string {
	return gitVersion
}
