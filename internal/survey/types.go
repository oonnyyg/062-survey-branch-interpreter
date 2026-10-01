package survey

import "encoding/json"

// Option is one selectable choice of a single-choice question.
type Option struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Next string `json:"next"`
}

// RangeRule maps a closed integer interval to a jump target.
type RangeRule struct {
	Min  int    `json:"min"`
	Max  int    `json:"max"`
	Next string `json:"next"`
}

// Question is either type "single" or "integer".
type Question struct {
	ID          string      `json:"id"`
	Type        string      `json:"type"`
	Text        string      `json:"text"`
	Options     []Option    `json:"options,omitempty"`
	Min         *int        `json:"min,omitempty"`
	Max         *int        `json:"max,omitempty"`
	Rules       []RangeRule `json:"rules,omitempty"`
	DefaultNext string      `json:"default_next"`
}

// Response is one filled-in survey, keyed by its unique ID. Answers are keyed
// by question ID and are decoded lazily so that type errors can be reported
// precisely during interpretation.
type Response struct {
	ID      string                     `json:"id"`
	Answers map[string]json.RawMessage `json:"answers"`
}

// Survey is the complete offline input document.
type Survey struct {
	Entry     string     `json:"entry"`
	Questions []Question `json:"questions"`
	Responses []Response `json:"responses"`
}
