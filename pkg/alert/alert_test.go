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
package alert_test

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maksim-paskal/aks-node-termination-handler/pkg/alert"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/config"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/template"
	log "github.com/sirupsen/logrus"
)

var ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if err := testWebhookRequest(r); err != nil {
		log.WithError(err).Error()
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		_, _ = w.Write([]byte("OK"))
	}
}))

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

func TestWebHook(t *testing.T) {
	t.Parallel()

	_ = flag.Set("webhook.url", ts.URL+"/metrics/job/aks-node-termination-handler")
	_ = flag.Set("webhook.template", `node_termination_event{node="{{ .Node }}"} 1`)

	if err := config.Load(); err != nil {
		t.Fatal(err)
	}

	if err := alert.SendWebHook(context.Background(), template.MessageType{
		Node: "test",
	}); err != nil {
		t.Fatal(err)
	}
}
