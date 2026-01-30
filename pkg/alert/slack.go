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

package alert

import (
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/vince-riv/aks-node-termination-handler/pkg/config"
	"github.com/vince-riv/aks-node-termination-handler/pkg/template"
)

const slackAuthTestCacheDuration = 30 * time.Minute

var (
	slackClient       *slack.Client //nolint:gochecknoglobals
	slackLastAuthTest atomic.Int64  //nolint:gochecknoglobals
)

func initSlack() error {
	token := config.GetSlackToken()
	channel := *config.Get().SlackChannel

	hasToken := len(token) > 0
	hasChannel := len(channel) > 0

	// Both empty - Slack alerting disabled
	if !hasToken && !hasChannel {
		log.Info("Slack alerting disabled")

		return nil
	}

	// Only one configured - log error and disable
	if hasToken && !hasChannel {
		log.Error("SLACK_TOKEN set but slack.channel not configured, Slack alerting disabled")

		return nil
	}

	if !hasToken && hasChannel {
		log.Error("slack.channel set but SLACK_TOKEN not configured, Slack alerting disabled")

		return nil
	}

	// Both configured - initialize client
	slackClient = slack.New(token)

	// Verify the token is valid
	authTest, err := slackClient.AuthTest()
	if err != nil {
		return errors.Wrap(err, "error in Slack AuthTest")
	}

	slackLastAuthTest.Store(time.Now().Unix())

	log.Printf("Slack authorized as %s in team %s", authTest.User, authTest.Team)

	return nil
}

func pingSlack() error {
	if slackClient == nil {
		return nil
	}

	// Skip auth test if last successful test was within cache duration
	lastTest := time.Unix(slackLastAuthTest.Load(), 0)
	if time.Since(lastTest) < slackAuthTestCacheDuration {
		return nil
	}

	_, err := slackClient.AuthTest()
	if err != nil {
		return errors.Wrap(err, "error in Slack AuthTest")
	}

	slackLastAuthTest.Store(time.Now().Unix())

	return nil
}

// buildSlackMessage builds the message content for Slack.
// Currently returns simple text, but can be extended to return Block Kit blocks.
func buildSlackMessage(obj *template.MessageType) (string, []slack.Block, error) { //nolint:unparam
	messageText, err := template.Message(obj)
	if err != nil {
		return "", nil, errors.Wrap(err, "error in template.Message")
	}

	// For now, return simple text with no blocks.
	// To add Block Kit support later, build blocks here and return them.
	return messageText, nil, nil
}

// SendSlack sends an alert message to the configured Slack channel.
func SendSlack(obj *template.MessageType) error {
	if slackClient == nil {
		return nil
	}

	channel := *config.Get().SlackChannel

	messageText, blocks, err := buildSlackMessage(obj)
	if err != nil {
		return errors.Wrap(err, "error building Slack message")
	}

	// Build message options
	options := []slack.MsgOption{
		slack.MsgOptionText(messageText, false),
	}

	// Add blocks if present (for future Block Kit support)
	if len(blocks) > 0 {
		options = append(options, slack.MsgOptionBlocks(blocks...))
	}

	_, timestamp, err := slackClient.PostMessage(channel, options...)
	if err != nil {
		return errors.Wrap(err, "error in Slack PostMessage")
	}

	log.Infof("Slack message sent, timestamp=%s", timestamp)

	return nil
}
