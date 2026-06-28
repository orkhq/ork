package events

// Renderer presents lifecycle events and terminal errors to a user or machine.
type Renderer interface {
	Render(e Event)
	RenderError(err error)
}
