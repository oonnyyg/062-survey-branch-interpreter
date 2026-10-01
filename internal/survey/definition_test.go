package survey

import (
	"strings"
	"testing"
)

const validDoc = `{
  "entry": "q1",
  "questions": [
    {"id": "q1", "type": "single", "default_next": "q2",
     "options": [{"value": "A", "next": "q2"}, {"value": "B", "next": "q3"}]},
    {"id": "q2", "type": "integer", "min": 0, "max": 100, "default_next": "END",
     "rules": [{"min": 0, "max": 10, "next": "q3"}]},
    {"id": "q3", "type": "single", "default_next": "END",
     "options": [{"value": "X", "next": "END"}]}
  ],
  "responses": []
}`

func TestParseValid(t *testing.T) {
	if _, err := ParseDefinition([]byte(validDoc)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDefinitionErrors(t *testing.T) {
	cases := []struct {
		name    string
		old     string
		new     string
		wantSub string
	}{
		{"duplicate id", `{"id": "q3"`, `{"id": "q1"`, "duplicate question id"},
		{"END as id", `{"id": "q3"`, `{"id": "END"`, "must not be"},
		{"unknown target", `"next": "q3"}]}`, `"next": "nope"}]}`, "unknown target"},
		{"duplicate option", `{"value": "B", "next": "q3"}`, `{"value": "A", "next": "q3"}`, "duplicate option"},
		{"empty option", `{"value": "B", "next": "q3"}`, `{"value": "", "next": "q3"}`, "value is empty"},
		{"overlapping rules", `{"min": 0, "max": 10, "next": "q3"}`, `{"min": 0, "max": 10, "next": "q3"}, {"min": 5, "max": 20, "next": "q3"}`, "overlaps"},
		{"rule out of range", `{"min": 0, "max": 10, "next": "q3"}`, `{"min": 0, "max": 200, "next": "q3"}`, "outside legal range"},
		{"bad legal range", `"min": 0, "max": 100`, `"min": 100, "max": 0`, "invalid range"},
		{"missing entry", `"entry": "q1"`, `"entry": "nope"`, "does not exist"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseDefinition([]byte(replace(validDoc, tc.old, tc.new)))
			if err == nil || !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("got %v, want error containing %q", err, tc.wantSub)
			}
		})
	}
}

func TestCycleRejected(t *testing.T) {
	doc := replace(validDoc, `{"value": "X", "next": "END"}`, `{"value": "X", "next": "q1"}`)
	_, err := ParseDefinition([]byte(doc))
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("got %v, want cycle error", err)
	}
}

func TestUnreachableRejected(t *testing.T) {
	doc := replace(validDoc, `{"value": "B", "next": "q3"}`, `{"value": "B", "next": "q2"}`)
	doc = replace(doc, `{"min": 0, "max": 10, "next": "q3"}`, `{"min": 0, "max": 10, "next": "END"}`)
	_, err := ParseDefinition([]byte(doc))
	if err == nil || !strings.Contains(err.Error(), "unreachable") {
		t.Fatalf("got %v, want unreachable error", err)
	}
}

func replace(s, old, new string) string {
	return strings.Replace(s, old, new, 1)
}
