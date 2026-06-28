package events

import (
	"ork/pkg/utils"
)

// RendererEmitter selects a TTY or JSON renderer based on the current output
// stream and implements Emitter for lifecycle producers.
type RendererEmitter struct {
	r Renderer
}

// NewRendererEmitter returns an emitter appropriate for the current terminal.
func NewRendererEmitter() *RendererEmitter {
	var r Renderer
	if utils.IsTTY() {
		r = NewTTYRenderer()
	} else {
		r = NewJSONRenderer()
	}

	return &RendererEmitter{r: r}
}

func (e RendererEmitter) Emit(event Event) {
	e.r.Render(event)
}
