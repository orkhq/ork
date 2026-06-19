package events

import "time"

// Type classifies the kind of lifecycle event emitted during orchestration.
type Type string

const (
	// EventStart signals the beginning of a component operation.
	EventStart Type = "start"
	// EventSuccess signals a component operation completed successfully.
	EventSuccess Type = "success"
	// EventFailure signals a component operation failed.
	EventFailure Type = "failure"
	// EventInfo carries an informational message about an operation.
	EventInfo Type = "info"
	// EventWarning signals a non-fatal issue encountered during an operation.
	EventWarning Type = "warning"
)

// Event represents a single lifecycle event emitted during orchestration of a
// component. It captures the context (component, adapter, runner, stage) and
// optional details such as error information or elapsed duration.
type Event struct {
	Type      Type          // The category of this event.
	Component string        // Name of the component this event relates to.
	Adapter   string        // Adapter type handling the component (e.g. "terraform").
	Runner    string        // Runner executing the component (e.g. "local", "ssh").
	Stage     string        // Lifecycle stage (e.g. "apply", "destroy").
	Message   string        // Human-readable description of what happened.
	Hint      string        // Optional suggestion for resolving warnings.
	Err       error         // Error details when Type is EventFailure.
	Duration  time.Duration // Elapsed time for the operation, if applicable.
}
