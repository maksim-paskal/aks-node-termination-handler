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
package webhook

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/maksim-paskal/aks-node-termination-handler/pkg/config"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/template"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var client = &http.Client{}

var errHTTPNotOK = errors.New("http result not OK")

func SetHTTPClient(c *http.Client) {
	client = c
}

func SendWebHook(ctx context.Context, obj *template.MessageType) error {
	ctx, cancel := context.WithTimeout(ctx, *config.Get().WebHookTimeout)
	defer cancel()

	if len(*config.Get().WebHookURL) == 0 {
		return nil
	}

	message, err := template.NewMessageType(ctx, obj.NodeName, obj.Event)
	if err != nil {
		return errors.Wrap(err, "error in template.NewMessageType")
	}

	message.Template = *config.Get().WebHookTemplate

	if len(*config.Get().WebHookTemplateFile) > 0 {
		templateFile, err := os.ReadFile(*config.Get().WebHookTemplateFile)
		if err != nil {
			return errors.Wrap(err, "error in os.ReadFile")
		}

		message.Template = string(templateFile)
	}

	webhookBody, err := template.Message(message)
	if err != nil {
		return errors.Wrap(err, "error in template.Message")
	}

	requestBody := bytes.NewBufferString(webhookBody + "\n")

	req, err := http.NewRequestWithContext(ctx, *config.Get().WebHookMethod, *config.Get().WebHookURL, requestBody)
	if err != nil {
		return errors.Wrap(err, "error in http.NewRequestWithContext")
	}

	req.Header.Set("Content-Type", *config.Get().WebHookContentType)

	log.WithFields(log.Fields{
		"method":  req.Method,
		"url":     req.URL,
		"headers": req.Header,
	}).Infof("Doing request with body: %s", requestBody.String())

	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "error in client.Do")
	}
	defer resp.Body.Close()

	log.Infof("response status: %s", resp.Status)

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(errHTTPNotOK, fmt.Sprintf("StatusCode=%d", resp.StatusCode))
	}

	return nil
}
