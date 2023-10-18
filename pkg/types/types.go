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
	EventId           string //nolint:golint,revive,stylecheck
	EventType         ScheduledEventsEventType
	ResourceType      string
	Resources         []string
	EventStatus       string
	NotBefore         string // Mon, 19 Sep 2016 18:29:47 GMT
	Description       string
	EventSource       string
	DurationInSeconds int
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
