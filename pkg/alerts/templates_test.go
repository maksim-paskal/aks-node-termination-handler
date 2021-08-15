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
