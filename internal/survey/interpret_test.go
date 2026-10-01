package survey

import (
	"reflect"
	"strings"
	"testing"
)

func parse(t *testing.T, doc string) *Definition {
	t.Helper()
	def, err := ParseDefinition([]byte(doc))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return def
}

func TestInterpretCompleted(t *testing.T) {
	def := parse(t, `{
	  "entry": "q1",
	  "questions": [
	    {"id": "q1", "type": "single", "default_next": "q2",
	     "options": [{"value": "A", "next": "q2"}, {"value": "B", "next": "q3"}]},
	    {"id": "q2", "type": "integer", "min": 0, "max": 100, "default_next": "END",
	     "rules": [{"min": 0, "max": 10, "next": "q3"}]},
	    {"id": "q3", "type": "single", "default_next": "END",
	     "options": [{"value": "X", "next": "END"}]}
	  ],
	  "responses": [
	    {"id": "r2", "answers": {"q1": "B", "q3": "X", "q9": "ignored"}},
	    {"id": "r1", "answers": {"q1": "A", "q2": 5, "q3": "X"}}
	  ]
	}`)
	results := def.InterpretAll()
	if results[0].ID != "r1" || results[1].ID != "r2" {
		t.Fatalf("results not sorted by id: %v", []string{results[0].ID, results[1].ID})
	}
	r1 := results[0]
	if r1.Status != "completed" || r1.Error != nil {
		t.Fatalf("r1: %+v", r1)
	}
	wantPath := []struct{ q, matched, next string }{
		{"q1", `option "A"`, "q2"},
		{"q2", "range [0, 10]", "q3"},
		{"q3", `option "X"`, "END"},
	}
	if len(r1.Path) != len(wantPath) {
		t.Fatalf("r1 path: %+v", r1.Path)
	}
	for i, w := range wantPath {
		if r1.Path[i].Question != w.q || r1.Path[i].Matched != w.matched || r1.Path[i].Next != w.next {
			t.Errorf("step %d: got %+v want %+v", i, r1.Path[i], w)
		}
	}
	r2 := results[1]
	if r2.Status != "completed" {
		t.Fatalf("r2: %+v", r2)
	}
	if !reflect.DeepEqual(r2.Unused, []string{"q9"}) {
		t.Errorf("r2 unused: %v", r2.Unused)
	}
}

func TestInterpretErrors(t *testing.T) {
	def := parse(t, `{
	  "entry": "q1",
	  "questions": [
	    {"id": "q1", "type": "integer", "min": 0, "max": 10, "default_next": "q2"},
	    {"id": "q2", "type": "single", "default_next": "END",
	     "options": [{"value": "A", "next": "END"}]}
	  ],
	  "responses": [
	    {"id": "missing", "answers": {}},
	    {"id": "bool", "answers": {"q1": true}},
	    {"id": "float", "answers": {"q1": 1.5}},
	    {"id": "range", "answers": {"q1": 99}},
	    {"id": "badopt", "answers": {"q1": 5, "q2": "ZZ"}},
	    {"id": "unusedbad", "answers": {"q1": 5, "q2": "A", "q9": "whatever"}}
	  ]
	}`)
	results := def.InterpretAll()
	byID := map[string]Result{}
	for _, r := range results {
		byID[r.ID] = r
	}
	checks := []struct{ id, wantSub string }{
		{"missing", "missing answer"},
		{"bool", "must be an integer"},
		{"float", "must be an integer"},
		{"range", "outside legal range"},
		{"badopt", "not a valid option"},
	}
	for _, c := range checks {
		r := byID[c.id]
		if r.Status != "error" || r.Error == nil {
			t.Fatalf("%s: expected error, got %+v", c.id, r)
		}
		if got := r.Error.Message; !strings.Contains(got, c.wantSub) {
			t.Errorf("%s: message %q, want substring %q", c.id, got, c.wantSub)
		}
	}
	if r := byID["unusedbad"]; r.Status != "completed" || !reflect.DeepEqual(r.Unused, []string{"q9"}) {
		t.Errorf("unusedbad: %+v", r)
	}
}

func TestOrderIndependence(t *testing.T) {
	docA := `{
	  "entry": "q1",
	  "questions": [
	    {"id": "q1", "type": "integer", "min": 0, "max": 10, "default_next": "END",
	     "rules": [{"min": 0, "max": 5, "next": "q2"}, {"min": 6, "max": 10, "next": "END"}]},
	    {"id": "q2", "type": "single", "default_next": "END",
	     "options": [{"value": "A", "next": "END"}, {"value": "B", "next": "END"}]}
	  ],
	  "responses": [{"id": "r", "answers": {"q1": 3, "q2": "B"}}]
	}`
	docB := `{
	  "entry": "q1",
	  "questions": [
	    {"id": "q2", "type": "single", "default_next": "END",
	     "options": [{"value": "B", "next": "END"}, {"value": "A", "next": "END"}]},
	    {"id": "q1", "type": "integer", "min": 0, "max": 10, "default_next": "END",
	     "rules": [{"min": 6, "max": 10, "next": "END"}, {"min": 0, "max": 5, "next": "q2"}]}
	  ],
	  "responses": [{"id": "r", "answers": {"q1": 3, "q2": "B"}}]
	}`
	ra := parse(t, docA).InterpretAll()
	rb := parse(t, docB).InterpretAll()
	if !reflect.DeepEqual(ra, rb) {
		t.Errorf("order changed result:\nA=%+v\nB=%+v", ra, rb)
	}
}
