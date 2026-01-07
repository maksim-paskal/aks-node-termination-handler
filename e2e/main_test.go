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
package main_test

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vince-riv/aks-node-termination-handler/internal"
	"github.com/vince-riv/aks-node-termination-handler/pkg/client"
	"github.com/vince-riv/aks-node-termination-handler/pkg/config"
	"github.com/vince-riv/aks-node-termination-handler/pkg/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	azureResourceName = "test-e2e-resource"
	eventID           = "test-event-id"
	eventType         = types.EventTypePreempt
	taintKey          = "aks-node-termination-handler/preempt"
	taintEffect       = corev1.TaintEffectNoSchedule
)

func TestDrain(t *testing.T) { //nolint:funlen,cyclop
	t.Parallel()

	log.SetLevel(log.DebugLevel)
	log.SetReportCaller(true)

	handler := http.NewServeMux()
	handler.HandleFunc("/document", func(w http.ResponseWriter, _ *http.Request) {
		message, _ := json.Marshal(types.ScheduledEventsType{
			DocumentIncarnation: 1,
			Events: []types.ScheduledEventsEvent{
				{
					EventId:      eventID,
					EventType:    eventType,
					ResourceType: "resourceType",
					Resources:    []string{azureResourceName},
				},
			},
		})

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(message)
	})

	testServer := httptest.NewServer(handler)

	_ = flag.Set("config", "./testdata/config_test.yaml")
	_ = flag.Set("endpoint", testServer.URL+"/document")
	_ = flag.Set("resource.name", azureResourceName)

	flag.Parse()

	ctx := context.TODO()

	if err := internal.Run(ctx); err != nil {
		t.Fatal(err)
	}

	node, err := client.GetKubernetesClient().CoreV1().Nodes().Get(ctx, *config.Get().NodeName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if !node.Spec.Unschedulable {
		t.Fatal("node must be unschedulable")
	}

	if len(node.Spec.Taints) == 0 {
		t.Fatal("node must have taints")
	}

	taintFound := false

	for _, taint := range node.Spec.Taints {
		if taint.Key == taintKey && taint.Value == eventID && taint.Effect == taintEffect {
			taintFound = true

			break
		}
	}

	if !taintFound {
		t.Fatal("taint not found")
	}

	if err := checkNodeEvent(ctx); err != nil {
		t.Fatal(err)
	}
}

func checkNodeEvent(ctx context.Context) error { //nolint:cyclop
	events, err := client.GetKubernetesClient().CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "error in list events")
	}

	nodeName := *config.Get().NodeName
	eventMessageReceived := 0
	eventMessageBeforeListen := 0

	for _, event := range events.Items {
		if event.Source.Component != "aks-node-termination-handler" {
			continue
		}

		if event.InvolvedObject.Name != nodeName {
			continue
		}

		if event.Reason == eventType && event.Message == config.EventMessageReceived {
			eventMessageReceived++
		}

		if event.Reason == "ReadEvents" && event.Message == config.EventMessageBeforeListen {
			eventMessageBeforeListen++
		}
	}

	if eventMessageReceived == 0 {
		return errors.New("eventMessageReceived not found in events")
	}

	if eventMessageBeforeListen == 0 {
		return errors.New("eventMessageBeforeListen not found in events")
	}

	return nil
}
