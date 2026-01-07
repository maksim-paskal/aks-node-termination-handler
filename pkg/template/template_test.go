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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/vince-riv/aks-node-termination-handler/pkg/template"
	"github.com/vince-riv/aks-node-termination-handler/pkg/types"
)

const fakeTemplate = "{{"

func TestTemplateMessage(t *testing.T) {
	t.Parallel()

	obj := &template.MessageType{
		Event: types.ScheduledEventsEvent{
			EventId:   "someID",
			EventType: "someType",
		},
		NodePods: []string{"pod1", "pod2"},
		Template: "test {{ .Event.EventId }} {{ .Event.EventType }} {{ .NodePods}}",
	}

	tpl, err := template.Message(obj)
	if err != nil {
		t.Fatal(err)
	}

	if want := "test someID someType [pod1 pod2]"; tpl != want {
		t.Fatalf("want=%s,got=%s", want, tpl)
	}
}

func TestFakeTemplate(t *testing.T) {
	t.Parallel()

	_, err := template.Message(&template.MessageType{
		Template: fakeTemplate,
	})
	if err == nil {
		t.Fatal("must be error")
	}
}

func TestFakeTemplateFunc(t *testing.T) {
	t.Parallel()

	_, err := template.Message(&template.MessageType{
		Template: "{{ .DDD }}",
	})
	if err == nil {
		t.Fatal("must be error")
	}

	t.Log(err)
}

func TestTemplateMarkdown(t *testing.T) {
	t.Parallel()

	message := template.MessageType{}

	messageBytes, err := os.ReadFile("testdata/message.json")
	if err != nil {
		t.Fatal(err)
	}

	if err := json.Unmarshal(messageBytes, &message); err != nil {
		t.Fatal(err)
	}

	printType("", message)

	if err = os.WriteFile("README.md.tmp", []byte(buf.String()), 0o644); err != nil { //nolint:gosec
		t.Fatal(err)
	}
}

var buf strings.Builder

func printType(prefix string, message interface{}) {
	v := reflect.ValueOf(message)
	typeOfS := v.Type()

	for i := range v.NumField() {
		switch typeOfS.Field(i).Name {
		case "Template":
		case "Event":
			printType(typeOfS.Field(i).Name+".", v.Field(i).Interface())
		default:
			value := v.Field(i).Interface()

			switch v.Field(i).Type().Kind() { //nolint:exhaustive
			case reflect.Slice:
				a := reflect.ValueOf(value).Interface().([]string) //nolint:forcetypeassert
				if len(a) > 0 {
					value = fmt.Sprintf("[ %s ...]", a[0])
				}
			case reflect.Int:
				value = fmt.Sprintf("%d", value)
			case reflect.Map:
				a := reflect.ValueOf(value).Interface().(map[string]string) //nolint:forcetypeassert
				for k, v := range a {
					value = fmt.Sprintf("%s:%s ...", k, v)

					break
				}
			}

			buf.WriteString(fmt.Sprintf(
				"| `{{ .%s%s }}` | %v | %v |\n",
				prefix,
				typeOfS.Field(i).Name,
				typeOfS.Field(i).Tag.Get("description"),
				value,
			))
		}
	}
}

func TestNewMessageType(t *testing.T) {
	t.Parallel()

	if _, err := template.NewMessageType(context.TODO(), "!!invalid!!GetNodeLabels", types.ScheduledEventsEvent{}); err == nil {
		t.Fatal("error expected")
	}

	if _, err := template.NewMessageType(context.TODO(), "!!invalid!!GetNodePods", types.ScheduledEventsEvent{}); err == nil {
		t.Fatal("error expected")
	}

	messageType, err := template.NewMessageType(context.TODO(), "somenode", types.ScheduledEventsEvent{})
	if err != nil {
		t.Fatal(err)
	}

	if messageType.NodeName != "somenode" {
		t.Fatal("NodePods is nil")
	}
}
