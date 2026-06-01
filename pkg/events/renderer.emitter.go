package events

import (
	"orch/pkg/utils"
)

type RendererEmitter struct {
	r Renderer
}

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
