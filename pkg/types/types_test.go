package types_test

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/maksim-paskal/aks-node-termination-handler/pkg/types"
)

func TestScheduledEventsType(t *testing.T) {
	t.Parallel()

	messageBytes, err := ioutil.ReadFile("testdata/ScheduledEventsType.json")
	if err != nil {
		t.Fatal(err)
	}

	message := types.ScheduledEventsType{}

	err = json.Unmarshal(messageBytes, &message)
	if err != nil {
		t.Fatal(err)
	}

	if len(message.Events) == 0 {
		t.Fatal("events is empty")
	}

	if want := "VirtualMachine"; message.Events[0].ResourceType != want {
		t.Fatalf("want=%s, got=%s", want, message.Events[0].ResourceType)
	}
}
