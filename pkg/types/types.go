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
