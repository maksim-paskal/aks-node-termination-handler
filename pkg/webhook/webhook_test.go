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

	"github.com/hashicorp/go-retryablehttp"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/metrics"
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

	retryClient := retryablehttp.NewClient()
	retryClient.HTTPClient.Transport = metrics.NewInstrumenter("TestWebHook").
		WithProxy("").
		WithInsecureSkipVerify(true).
		InstrumentedRoundTripper()
	retryClient.RetryMax = 0

	retryClientProxy := retryablehttp.NewClient()
	retryClientProxy.HTTPClient.Transport = metrics.NewInstrumenter("TestWebHookWithProxy").
		WithProxy("http://someproxy").
		WithInsecureSkipVerify(true).
		InstrumentedRoundTripper()
	retryClientProxy.RetryMax = 0

	type Test struct {
		Name         string
		Args         map[string]string
		Error        bool
		ErrorMessage string
		NodeName     string
		HTTPClient   *retryablehttp.Client
	}

	tests := []Test{
		{
			Name: "ValidHookAndTemplate",
			Args: map[string]string{
				"webhook.url":      getWebhookURL(),
				"webhook.template": `node_termination_event{node="{{ .NodeName }}"} 1`,
			},
		},
		{
			Name: "EmptyURL",
			Args: map[string]string{
				"webhook.url":      "",
				"webhook.template": `node_termination_event{node="{{ .NodeName }}"} 1`,
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
				"webhook.template": `{{ .NodeName }}`,
			},
			Error: true,
		},
		{
			Name: "InvalidStatus",
			Args: map[string]string{
				"webhook.url":      ts.URL,
				"webhook.template": `{{ .NodeName }}`,
			},
			Error: true,
		},
		{
			Name: "InvalidMethod",
			Args: map[string]string{
				"webhook.url":      getWebhookURL(),
				"webhook.template": `{{ .NodeName }}`,
				"webhook.method":   `???`,
			},
			Error: true,
		},
		{
			Name: "WebhookTemplateFile",
			Args: map[string]string{
				"webhook.url":           getWebhookURL(),
				"webhook.template-file": "testdata/WebhookTemplateFile.txt",
			},
		},
		{
			Error: true,
			Name:  "WebhookTemplateFileInvalid",
			Args: map[string]string{
				"webhook.url":           getWebhookURL(),
				"webhook.template-file": "faketestdata/WebhookTemplateFile.txt",
			},
		},
		{
			Error: true,
			Name:  "InvalidNodeName",
			Args: map[string]string{
				"webhook.url": getWebhookURL(),
			},
			NodeName: "!!invalid!!GetNodeLabels",
		},
		{
			Error:        true,
			ErrorMessage: "error making roundtrip: proxyconnect tcp: dial tcp",
			Name:         "HTTPClientProxy",
			Args: map[string]string{
				"webhook.url": getWebhookURL(),
			},
			HTTPClient: retryClientProxy,
		},
	}

	// clear flags
	cleanAllFlags := func() {
		for _, test := range tests {
			for key := range test.Args {
				_ = flag.Set(key, "")
			}
		}
	}

	for _, tc := range tests { //nolint:paralleltest
		t.Run(tc.Name, func(t *testing.T) {
			cleanAllFlags()

			for key, value := range tc.Args {
				_ = flag.Set(key, value)
			}

			messageType := &template.MessageType{
				NodeName: "test",
			}

			if len(tc.NodeName) > 0 {
				messageType.NodeName = tc.NodeName
			}

			if httpClient := tc.HTTPClient; httpClient != nil {
				webhook.SetHTTPClient(httpClient)
			} else {
				webhook.SetHTTPClient(retryClient)
			}

			err := webhook.SendWebHook(context.TODO(), messageType)
			if tc.Error {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.ErrorMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
