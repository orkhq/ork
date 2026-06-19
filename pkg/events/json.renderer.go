package events

import (
	"encoding/json"
	"fmt"
)

// JSONRenderer renders events as JSON objects to stdout, suitable for
// machine-readable output in CI pipelines or non-TTY environments.
type JSONRenderer struct{}

func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{}
}

func (r *JSONRenderer) Render(e Event) {
	data, _ := json.Marshal(e)
	fmt.Println(string(data))
}

func (r *JSONRenderer) RenderError(err error) {
	out := map[string]string{"error": err.Error()}
	data, _ := json.Marshal(out)
	fmt.Println(string(data))
}
