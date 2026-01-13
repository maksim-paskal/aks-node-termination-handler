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
package types

import (
	"fmt"
	"regexp"
	"time"

	"github.com/pkg/errors"
)

type ScheduledEventsEventType string

const (
	// The Virtual Machine is scheduled to pause for a few seconds. CPU and network connectivity
	// may be suspended, but there's no impact on memory or open files.
	EventTypeFreeze = "Freeze"
	// The Virtual Machine is scheduled for reboot (non-persistent memory is lost).
	// This event is made available on a best effort basis.
	EventTypeReboot = "Reboot"
	// The Virtual Machine is scheduled to move to another node (ephemeral disks are lost).
	// This event is delivered on a best effort basis.
	EventTypeRedeploy = "Redeploy"
	// The Spot Virtual Machine is being deleted (ephemeral disks are lost).
	EventTypePreempt = "Preempt"
	// The virtual machine is scheduled to be deleted.
	EventTypeTerminate = "Terminate"
)

// https://docs.microsoft.com/en-us/azure/virtual-machines/linux/scheduled-events
type ScheduledEventsEvent struct {
	EventId           string                   `description:"Globally unique identifier for this event."` //nolint:golint,revive,stylecheck
	EventType         ScheduledEventsEventType `description:"Impact this event causes."`
	ResourceType      string                   `description:"Type of resource this event affects."`
	Resources         []string                 `description:"List of resources this event affects."`
	EventStatus       string                   `description:"Status of this event."`
	NotBefore         string                   `description:"Time after which this event can start. The event is guaranteed to not start before this time. Will be blank if the event has already started"` //nolint:lll
	Description       string                   `description:"Description of this event."`
	EventSource       string                   `description:"Initiator of the event."`
	DurationInSeconds int                      `description:"The expected duration of the interruption caused by the event."`
}

// NotBeforeTime parses the NotBefore field and returns it as time.Time.
// Returns zero time if NotBefore is empty (event has already started).
func (e *ScheduledEventsEvent) NotBeforeTime() (time.Time, error) {
	if e.NotBefore == "" {
		return time.Time{}, nil
	}

	t, err := time.Parse(time.RFC1123, e.NotBefore)
	if err != nil {
		return time.Time{}, errors.Wrap(err, "failed to parse NotBefore time")
	}

	return t, nil
}

var (
	virtualMachineScaleSetsRe = regexp.MustCompile("^azure:///subscriptions/(.+)/resourceGroups/(.+)/providers/Microsoft.Compute/virtualMachineScaleSets/(.+)/virtualMachines/(.+)$")
	virtualMachineRe          = regexp.MustCompile("^azure:///subscriptions/(.+)/resourceGroups/(.+)/providers/Microsoft.Compute/virtualMachines/(.+)$")
)

type AzureResource struct {
	ProviderID        string
	EventResourceName string
	SubscriptionID    string
	ResourceGroup     string
}

func NewAzureResource(providerID string) (*AzureResource, error) {
	resource := &AzureResource{
		ProviderID: providerID,
	}

	switch {
	case virtualMachineScaleSetsRe.MatchString(providerID):
		v := virtualMachineScaleSetsRe.FindAllStringSubmatch(providerID, 1)

		resource.SubscriptionID = v[0][1]
		resource.ResourceGroup = v[0][2]
		resource.EventResourceName = fmt.Sprintf("%s_%s", v[0][3], v[0][4])

	case virtualMachineRe.MatchString(providerID):
		v := virtualMachineRe.FindAllStringSubmatch(providerID, 1)

		resource.SubscriptionID = v[0][1]
		resource.ResourceGroup = v[0][2]
		resource.EventResourceName = v[0][3]

	default:
		return nil, errors.Errorf("providerID not recognized: %s", providerID)
	}

	return resource, nil
}

// api-version=2020-07-01.
type ScheduledEventsType struct {
	DocumentIncarnation int
	Events              []ScheduledEventsEvent
}

type EventMessage struct {
	Type    string
	Reason  string
	Message string
}
