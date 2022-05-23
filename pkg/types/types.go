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

// https://docs.microsoft.com/en-us/azure/virtual-machines/linux/scheduled-events
type ScheduledEventsEvent struct {
	EventId           string //nolint:golint,revive,stylecheck
	EventType         string
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
