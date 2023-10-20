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
//nolint:goerr113
package webhook_test

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maksim-paskal/aks-node-termination-handler/pkg/template"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/webhook"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if err := testWebhookRequest(r); err != nil {
		log.WithError(err).Error()
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		_, _ = w.Write([]byte("OK"))
	}
}))

func getWebhookURL() string {
	return ts.URL + "/metrics/job/aks-node-termination-handler"
}

func testWebhookRequest(r *http.Request) error {
	if r.RequestURI != "/metrics/job/aks-node-termination-handler" {
		return errors.New("Request URI is not correct")
	}

	defer r.Body.Close()

	body, _ := io.ReadAll(r.Body)

	if bodyString := string(body); bodyString != "node_termination_event{node=\"test\"} 1\n" {
		return fmt.Errorf("Response body [%s] is not correct", bodyString)
	}

	return nil
}

func TestWebHook(t *testing.T) { //nolint:funlen,tparallel
	t.Parallel()

	type Test struct {
		Name  string
		Args  map[string]string
		Error bool
	}

	tests := []Test{
		{
			Name: "ValidHookAndTemplate",
			Args: map[string]string{
				"webhook.url":      getWebhookURL(),
				"webhook.template": `node_termination_event{node="{{ .Node }}"} 1`,
			},
		},
		{
			Name: "EmptyURL",
			Args: map[string]string{
				"webhook.url":      "",
				"webhook.template": `node_termination_event{node="{{ .Node }}"} 1`,
			},
		},
		{
			Name: "InvalidTemplate",
			Args: map[string]string{
				"webhook.url":      getWebhookURL(),
				"webhook.template": `{{`,
			},
			Error: true,
		},
		{
			Name: "InvalidContext",
			Args: map[string]string{
				"webhook.url":      "example.com",
				"webhook.template": `{{ .Node }}`,
			},
			Error: true,
		},
		{
			Name: "InvalidStatus",
			Args: map[string]string{
				"webhook.url":      ts.URL,
				"webhook.template": `{{ .Node }}`,
			},
			Error: true,
		},
		{
			Name: "InvalidMethod",
			Args: map[string]string{
				"webhook.url":      getWebhookURL(),
				"webhook.template": `{{ .Node }}`,
				"webhook.method":   `???`,
			},
			Error: true,
		},
	}

	for _, test := range tests { //nolint:paralleltest
		tc := test
		t.Run(test.Name, func(t *testing.T) {
			for key, value := range tc.Args {
				_ = flag.Set(key, value)
			}

			err := webhook.SendWebHook(context.Background(), template.MessageType{
				Node: "test",
			})
			if tc.Error {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
