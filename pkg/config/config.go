/*
Copyright paskal.maksim@gmail.com (Original Author 2021-2025)
Copyright github@vince-riv.io (Modifications 2026-present)
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
	"encoding/json"
	"flag"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/vince-riv/aks-node-termination-handler/pkg/types"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
)

const (
	azureEndpoint                 = "http://169.254.169.254/metadata/scheduledevents?api-version=2020-07-01"
	defaultAlertMessage           = "Draining node={{ .NodeName }}, type={{ .Event.EventType }}"
	defaultPeriod                 = 5 * time.Second
	defaultPodGracePeriodSeconds  = -1
	defaultNodeGracePeriodSeconds = 120
	defaultGracePeriodSecond      = 10
	defaultRequestTimeout         = 5 * time.Second
	defaultWebHookTimeout         = 30 * time.Second
	defaultDryRun                 = false
)

const (
	EventMessageReceived     = "Azure API sended schedule event for this node"
	EventMessageBeforeListen = "Start to listen events from Azure API"
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
	DryRun                 *bool
	KubeConfigFile         *string
	Endpoint               *string
	NodeName               *string
	Period                 *time.Duration
	RequestTimeout         *time.Duration
	TelegramToken          *string
	TelegramChatID         *string
	AlertMessage           *string
	WebHookContentType     *string
	WebHookURL             *string
	WebHookTemplate        *string
	WebHookTemplateFile    *string
	WebHookMethod          *string
	WebHookTimeout         *time.Duration
	WebhookInsecure        *bool
	WebhookProxy           *string
	WebhookRetries         *int
	SentryDSN              *string
	WebHTTPAddress         *string
	TaintNode              *bool
	TaintEffect            *string
	PodGracePeriodSeconds  *int
	NodeGracePeriodSeconds *int
	GracePeriodSeconds     *int
	DrainOnFreezeEvent     *bool
	ResourceName           *string
	ExitAfterNodeDrain     *bool
	DisableEviction        *bool
	NotBeforeThreshold     *time.Duration
}

var config = Type{
	ConfigFile:             flag.String("config", os.Getenv("CONFIG"), "config file"),
	LogLevel:               flag.String("log.level", "INFO", "log level"),
	LogPretty:              flag.Bool("log.pretty", false, "log in text"),
	KubeConfigFile:         flag.String("kubeconfig", "", "kubeconfig file"),
	Endpoint:               flag.String("endpoint", azureEndpoint, "scheduled-events endpoint"),
	NodeName:               flag.String("node", os.Getenv("MY_NODE_NAME"), "node to drain"),
	Period:                 flag.Duration("period", defaultPeriod, "period to scrape endpoint"),
	RequestTimeout:         flag.Duration("request.timeout", defaultRequestTimeout, "request timeout"),
	TelegramToken:          flag.String("telegram.token", os.Getenv("TELEGRAM_TOKEN"), "telegram token"),
	TelegramChatID:         flag.String("telegram.chatID", os.Getenv("TELEGRAM_CHATID"), "telegram chatID"),
	AlertMessage:           flag.String("alert.message", defaultAlertMessage, "default message"),
	WebHookMethod:          flag.String("webhook.method", "POST", "request method"),
	WebHookContentType:     flag.String("webhook.contentType", "application/json", "request content-type header"),
	WebHookURL:             flag.String("webhook.url", os.Getenv("WEBHOOK_URL"), "send alerts to webhook"),
	WebHookTimeout:         flag.Duration("webhook.timeout", defaultWebHookTimeout, "request timeout"),
	WebHookTemplate:        flag.String("webhook.template", os.Getenv("WEBHOOK_TEMPLATE"), "request body"),
	WebHookTemplateFile:    flag.String("webhook.template-file", os.Getenv("WEBHOOK_TEMPLATE_FILE"), "path to request body template file"),
	WebhookInsecure:        flag.Bool("webhook.insecureSkip", true, "skip tls verification for webhook"),
	WebhookProxy:           flag.String("webhook.http-proxy", os.Getenv("WEBHOOK_HTTP_PROXY"), "use http proxy for webhook"),
	WebhookRetries:         flag.Int("webhook.retries", 3, "number of retries for webhook"), //nolint:mnd
	SentryDSN:              flag.String("sentry.dsn", "", "sentry DSN"),
	WebHTTPAddress:         flag.String("web.address", ":17923", ""),
	TaintNode:              flag.Bool("taint.node", false, "Taint the node before cordon and draining"),
	TaintEffect:            flag.String("taint.effect", "NoSchedule", "Taint effect to set on the node"),
	PodGracePeriodSeconds:  flag.Int("podGracePeriodSeconds", defaultPodGracePeriodSeconds, "grace period is seconds for pods termination"),
	NodeGracePeriodSeconds: flag.Int("nodeGracePeriodSeconds", defaultNodeGracePeriodSeconds, "maximum time in seconds to drain the node"),
	GracePeriodSeconds:     flag.Int("gracePeriodSeconds", defaultGracePeriodSecond, "grace period is seconds for application termination"),
	DrainOnFreezeEvent:     flag.Bool("drainOnFreezeEvent", false, "drain node on freeze event"),
	ResourceName:           flag.String("resource.name", "", "Azure resource name to drain"),
	ExitAfterNodeDrain:     flag.Bool("exitAfterNodeDrain", false, "process will exit after node drain"),
	DryRun:                 flag.Bool("dryRun", defaultDryRun, "if true, nodes will not be tainted, cordoned, or drained"),
	DisableEviction:        flag.Bool("disableEviction", false, "if true, force drain to use delete, even if eviction is supported. This will bypass checking PodDisruptionBudgets"),
	NotBeforeThreshold:     flag.Duration("notBeforeThreshold", 0, "ignore events where NotBefore is further in the future than this threshold (0 to disable)"),
}

func (t *Type) GracePeriod() time.Duration {
	return time.Duration(*t.GracePeriodSeconds) * time.Second
}

func (t *Type) NodeGracePeriod() time.Duration {
	return time.Duration(*t.NodeGracePeriodSeconds) * time.Second
}

// check is event is excluded from draining node.
func (t *Type) IsExcludedEvent(e types.ScheduledEventsEventType) bool {
	if e == types.EventTypeFreeze && !*t.DrainOnFreezeEvent {
		return true
	}

	return false
}

func (t *Type) String() string {
	b, err := json.Marshal(t)
	if err != nil {
		return err.Error()
	}

	return string(b)
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

	configByte, err := os.ReadFile(*config.ConfigFile)
	if err != nil {
		return errors.Wrap(err, "error in os.ReadFile")
	}

	err = yaml.Unmarshal(configByte, &config)
	if err != nil {
		return errors.Wrap(err, "error in yaml.Unmarshal")
	}

	return nil
}

var gitVersion = "dev"

func GetVersion() string {
	return gitVersion
}
