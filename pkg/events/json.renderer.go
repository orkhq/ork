package events

import (
	"encoding/json"
	"fmt"
)

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
