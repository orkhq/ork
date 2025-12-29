package events

type Emitter interface {
	Emit(Event)
}
