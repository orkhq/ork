package events

// Emitter accepts structured lifecycle events without coupling producers to a
// terminal or JSON renderer.
type Emitter interface {
	Emit(Event)
}
