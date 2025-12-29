package events

type Renderer interface {
	Render(e Event)
	RenderError(err error)
}
