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
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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

	webhookClient := &http.Client{
		Transport: metrics.NewInstrumenter("TestWebHook").
			WithProxy("").
			WithInsecureSkipVerify(true).
			InstrumentedRoundTripper(),
	}

	webhookClientProxy := &http.Client{
		Transport: metrics.NewInstrumenter("TestWebHookWithProxy").
			WithProxy("http://someproxy").
			WithInsecureSkipVerify(true).
			InstrumentedRoundTripper(),
	}

	type Test struct {
		Name         string
		Args         map[string]string
		Error        bool
		ErrorMessage string
		NodeName     string
		HTTPClient   *http.Client
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
			HTTPClient: webhookClientProxy,
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
				webhook.SetHTTPClient(webhookClient)
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
func TestDoRequestWithRetry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		retries        int
		mockResponses  []*http.Response
		mockErrors     []error
		expectedError  error
		expectedStatus int
	}{
		{
			name:    "SuccessOnFirstTry",
			retries: 3,
			mockResponses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("OK")),
				},
			},
			expectedError:  nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:    "SuccessOnRetry",
			retries: 3,
			mockResponses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString("Internal Server Error")),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("OK")),
				},
			},
			expectedError:  nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:    "FailAfterRetries",
			retries: 3,
			mockResponses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString("Internal Server Error")),
				},
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString("Internal Server Error")),
				},
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString("Internal Server Error")),
				},
			},
			expectedError:  webhook.ErrHTTPNotOK,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:    "NetworkError",
			retries: 3,
			mockErrors: []error{
				errors.New("network error"),
				errors.New("network error"),
				errors.New("network error"),
			},
			expectedError: errors.New("network error"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &http.Client{
				Transport: &mockTransport{
					responses: tc.mockResponses,
					errors:    tc.mockErrors,
				},
			}
			webhook.SetHTTPClient(mockClient)

			req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
			require.NoError(t, err)

			resp, err := webhook.DoRequestWithRetry(req, tc.retries)
			if tc.expectedError != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedStatus, resp.StatusCode)
			}
		})
	}
}

type mockTransport struct {
	responses []*http.Response
	errors    []error
	index     int
}

func (m *mockTransport) RoundTrip(*http.Request) (*http.Response, error) {
	if m.index < len(m.errors) && m.errors[m.index] != nil {
		err := m.errors[m.index]
		m.index++
		return nil, err
	}
	if m.index < len(m.responses) {
		resp := m.responses[m.index]
		m.index++
		return resp, nil
	}
	return nil, errors.New("no more mock responses")
}
