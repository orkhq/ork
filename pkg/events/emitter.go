// Package events provides lifecycle event emission and rendering for ork
// orchestration runs. Events signal component state transitions (start, success,
// failure) and are rendered to the terminal or as structured JSON depending on
// the output context.
package events

// Emitter is the interface for publishing lifecycle events during an
// orchestration run. Implementations decide how events are delivered (e.g.
// rendered to a terminal, serialized to JSON, or forwarded to an external
// system).
type Emitter interface {
	Emit(Event)
}
