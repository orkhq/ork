package events

import (
	"ork/pkg/utils"
)

// RendererEmitter is an Emitter that delegates event output to a Renderer.
// It automatically selects a TTY or JSON renderer based on whether stdout is a
// terminal.
type RendererEmitter struct {
	r Renderer
}

// NewRendererEmitter creates a RendererEmitter, choosing a TTYRenderer when
// stdout is a terminal and a JSONRenderer otherwise.
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
