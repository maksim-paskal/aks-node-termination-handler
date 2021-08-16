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
package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/config"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	bot    *tgbotapi.BotAPI
	client = &http.Client{}
)

type WebHookData struct {
	Text string `json:"text"`
}

func InitAlerts() error {
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

func Send(obj TemplateMessageType) error {
	messageText, err := TemplateMessage(obj)
	if err != nil {
		return err
	}

	if len(*config.Get().TelegramToken) != 0 {
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
	}

	if len(*config.Get().WebHookURL) != 0 {
		result, err := sendPostJSON(*config.Get().WebHookURL, WebHookData{
			Text: messageText,
		})
		if err != nil {
			return errors.Wrap(err, "error in sendPostJson")
		}

		log.Infof("Webhook result %s", result)
	}

	return nil
}

func sendPostJSON(url string, data interface{}) (bodyString string, err error) {
	jsonString, err := json.Marshal(data)
	if err != nil {
		return "", errors.Wrap(err, "error in json.Marshal")
	}

	log.Debug(string(jsonString))

	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewBuffer(jsonString))
	if err != nil {
		return "", errors.Wrap(err, "error in http.NewRequestWithContext")
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "error in client.Do")
	}

	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	bodyString = string(body)

	if resp.StatusCode != http.StatusOK {
		return bodyString, errors.Wrap(errHTTPNotOK, fmt.Sprintf("StatusCode=%d", resp.StatusCode))
	}

	return bodyString, nil
}
