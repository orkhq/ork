package events

// Renderer formats and outputs events to a specific medium. Implementations
// handle the presentation of lifecycle events (e.g. human-readable TTY output
// or machine-readable JSON).
type Renderer interface {
	Render(e Event)
	RenderError(err error)
}
