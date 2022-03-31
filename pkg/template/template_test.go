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
package template_test

import (
	"testing"

	"github.com/maksim-paskal/aks-node-termination-handler/pkg/template"
	"github.com/maksim-paskal/aks-node-termination-handler/pkg/types"
)

const fakeTemplate = "{{"

func TestTemplateMessage(t *testing.T) {
	t.Parallel()

	obj := template.MessageType{
		Event: types.ScheduledEventsEvent{
			EventId:   "someID",
			EventType: "someType",
		},
		Template: "test {{ .Event.EventId }} {{ .Event.EventType }}",
	}

	tpl, err := template.Message(obj)
	if err != nil {
		t.Fatal(err)
	}

	if want := "test someID someType"; tpl != want {
		t.Fatalf("want=%s,got=%s", want, tpl)
	}
}

func TestLineBreak(t *testing.T) {
	t.Parallel()

	obj := template.MessageType{
		Event: types.ScheduledEventsEvent{
			EventId:   "someID",
			EventType: "someType",
		},
		Template: `{{ .Event.EventId }}{{ .NewLine }}{{ .Event.EventType }}`,
	}

	tpl, err := template.Message(obj)
	if err != nil {
		t.Fatal(err)
	}

	if want := "someID\nsomeType"; tpl != want {
		t.Fatalf("want=%s,got=%s", want, tpl)
	}
}

func TestFakeTemplate(t *testing.T) {
	t.Parallel()

	_, err := template.Message(template.MessageType{
		Template: fakeTemplate,
	})
	if err == nil {
		t.Fatal("must be error")
	}
}
