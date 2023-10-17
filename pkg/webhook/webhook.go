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

	"github.com/maksim-paskal/aks-node-termination-handler/pkg/config"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/metrics"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/template"
	"github.com/pkg/errors"
)

var client = &http.Client{
	Transport: metrics.NewInstrumenter("webhook", true).InstrumentedRoundTripper(),
}

var errHTTPNotOK = errors.New("http result not OK")

func SendWebHook(ctx context.Context, obj template.MessageType) error {
	ctx, cancel := context.WithTimeout(ctx, *config.Get().WebHookTimeout)
	defer cancel()

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

	req, err := http.NewRequestWithContext(ctx, *config.Get().WebHookMethod, *config.Get().WebHookURL, requestBody)
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
