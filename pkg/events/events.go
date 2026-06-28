package events

import "time"

// Type classifies a lifecycle event for renderers and machine consumers.
type Type string

const (
	EventStart   Type = "start"
	EventSuccess Type = "success"
	EventFailure Type = "failure"
	EventInfo    Type = "info"
	EventWarning Type = "warning"
)

// Event is a user-visible lifecycle observation emitted by orchestration,
// adapters, or runners. Err is rendered but should never contain credentials.
type Event struct {
	Type      Type
	Component string
	Adapter   string
	Runner    string
	Stage     string

	Message  string
	Hint     string
	Err      error
	Duration time.Duration
}
