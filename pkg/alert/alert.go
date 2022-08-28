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
package alert

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/config"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/template"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var errHTTPNotOK = errors.New("http result not OK")

const httpRequestTimeout = 5 * time.Second

var (
	bot    *tgbotapi.BotAPI
	client = &http.Client{
		Timeout: httpRequestTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: *config.Get().WebHookInsecure, //nolint:gosec
			},
		},
	}
)

func Init() error {
	if len(*config.Get().TelegramToken) == 0 {
		log.Warning("not sending Telegram message, no token")

		return nil
	}

	var err error

	bot, err = tgbotapi.NewBotAPI(*config.Get().TelegramToken)
	if err != nil {
		return errors.Wrap(err, "error in NewBotAPI")
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	return nil
}

// healthcheck.
func Ping() error {
	if len(*config.Get().TelegramToken) != 0 {
		if _, err := bot.GetMe(); err != nil {
			return errors.Wrap(err, "error in bot.GetMe")
		}
	}

	return nil
}

func SendALL(ctx context.Context, obj template.MessageType) error {
	if err := SendTelegram(obj); err != nil {
		return errors.Wrap(err, "error in sending to telegram")
	}

	if err := SendWebHook(ctx, obj); err != nil {
		return errors.Wrap(err, "error in sending to webhook")
	}

	return nil
}

func SendTelegram(obj template.MessageType) error {
	if len(*config.Get().TelegramToken) == 0 {
		return nil
	}

	messageText, err := template.Message(obj)
	if err != nil {
		return errors.Wrap(err, "error in template.Message")
	}

	chatID, err := strconv.Atoi(*config.Get().TelegramChatID)
	if err != nil {
		return errors.Wrap(err, "error converting chatID")
	}

	msg := tgbotapi.NewMessage(int64(chatID), messageText)

	result, err := bot.Send(msg)
	if err != nil {
		return errors.Wrap(err, "error in bot.Send")
	}

	log.Infof("Telegram MessageID=%d", result.MessageID)

	return nil
}

func SendWebHook(ctx context.Context, obj template.MessageType) error {
	if len(*config.Get().WebHookURL) == 0 {
		return nil
	}

	webhookBody, err := template.Message(template.MessageType{
		Node:     obj.Node,
		Event:    obj.Event,
		Template: *config.Get().WebHookTemplate,
	})
	if err != nil {
		return errors.Wrap(err, "error in template.Message")
	}

	requestBody := bytes.NewBufferString(fmt.Sprintf("%s\n", webhookBody))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, *config.Get().WebHookURL, requestBody)
	if err != nil {
		return errors.Wrap(err, "error in http.NewRequestWithContext")
	}

	req.Header.Set("Content-Type", *config.Get().WebHookContentType)

	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "error in client.Do")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(errHTTPNotOK, fmt.Sprintf("StatusCode=%d", resp.StatusCode))
	}

	return nil
}
