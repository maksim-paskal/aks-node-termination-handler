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
package alerts_test

import (
	"testing"

	"github.com/maksim-paskal/aks-node-termination-handler/pkg/alerts"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/types"
)

func TestTemplateMessage(t *testing.T) {
	t.Parallel()

	obj := alerts.TemplateMessageType{
		Event: types.ScheduledEventsEvent{
			EventId:   "someID",
			EventType: "someType",
		},
		Template: "test {{ .Event.EventId }} {{ .Event.EventType }}",
	}

	tpl, err := alerts.TemplateMessage(obj)
	if err != nil {
		t.Fatal(err)
	}

	if want := "test someID someType"; tpl != want {
		t.Fatalf("want=%s,got=%s", want, tpl)
	}
}
