package survey

import (
	"encoding/json"
	"testing"
)

func intPtr(n int) *int { return &n }

func validSurvey() *Survey {
	return &Survey{
		Entry: "q1",
		Questions: []Question{
			{
				ID: "q1", Type: "single",
				Options: []Option{
					{ID: "a", Text: "A", Next: "q2"},
					{ID: "b", Text: "B", Next: "END"},
				},
				DefaultNext: "q2",
			},
			{
				ID: "q2", Type: "integer",
				Min: intPtr(0), Max: intPtr(100),
				Rules: []RangeRule{
					{Min: 0, Max: 17, Next: "END"},
					{Min: 65, Max: 100, Next: "q3"},
				},
				DefaultNext: "q3",
			},
			{
				ID: "q3", Type: "single",
				Options: []Option{
					{ID: "x", Text: "X", Next: "END"},
				},
				DefaultNext: "END",
			},
		},
	}
}

func TestValidDefinition(t *testing.T) {
	s := validSurvey()
	if err := s.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if err := s.ValidateGraph(); err != nil {
		t.Fatalf("ValidateGraph: %v", err)
	}
}

func TestDefinitionErrors(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(s *Survey)
		wantLoc string
	}{
		{"entry missing", func(s *Survey) { s.Entry = "nope" }, "entry"},
		{"entry is END", func(s *Survey) { s.Entry = END }, "entry"},
		{"duplicate question id", func(s *Survey) { s.Questions[2].ID = "q1" }, "questions[2].id"},
		{"question id END", func(s *Survey) { s.Questions[2].ID = END }, "questions[2].id"},
		{"unknown type", func(s *Survey) { s.Questions[0].Type = "bool" }, "questions[0].type"},
		{"duplicate option", func(s *Survey) { s.Questions[0].Options[1].ID = "a" }, "questions[0].options[1].id"},
		{"empty option id", func(s *Survey) { s.Questions[0].Options[0].ID = " " }, "questions[0].options[0].id"},
		{"option unknown target", func(s *Survey) { s.Questions[0].Options[0].Next = "zzz" }, "questions[0].options[0].next"},
		{"default unknown target", func(s *Survey) { s.Questions[0].DefaultNext = "zzz" }, "questions[0].default_next"},
		{"rule out of range", func(s *Survey) { s.Questions[1].Rules[1].Max = 200 }, "questions[1].rules[1]"},
		{"rules overlap", func(s *Survey) { s.Questions[1].Rules[1].Min = 10 }, "questions[1].rules[1]"},
		{"min greater max", func(s *Survey) { *s.Questions[1].Min = 999 }, "questions[1]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := validSurvey()
			tc.mutate(s)
			var err error
			if err = s.Validate(); err == nil {
				err = s.ValidateGraph()
			}
			if err == nil {
				t.Fatalf("expected error at %s, got nil", tc.wantLoc)
			}
			de, ok := err.(*DefError)
			if !ok || de.Location != tc.wantLoc {
				t.Fatalf("expected location %q, got %v", tc.wantLoc, err)
			}
		})
	}
}

func TestCycleDetected(t *testing.T) {
	s := validSurvey()
	s.Questions[0].Options[0].Next = "q3"
	s.Questions[2].Options[0].Next = "q2"
	s.Questions[1].DefaultNext = "q3"
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	err := s.ValidateGraph()
	if err == nil {
		t.Fatal("expected cycle error")
	}
}

func TestUnreachableDetected(t *testing.T) {
	s := validSurvey()
	s.Questions[0].DefaultNext = "END"
	s.Questions[0].Options[0].Next = "END"
	s.Questions[0].Options[1].Next = "END"
	err := s.ValidateGraph()
	if err == nil {
		t.Fatal("expected unreachable error")
	}
}

func TestInterpretHappyPathAndBranches(t *testing.T) {
	s := validSurvey()
	s.Responses = []Response{
		{ID: "b-default", Answers: map[string]json.RawMessage{
			"q1": json.RawMessage(`"a"`), // -> q2
			"q2": json.RawMessage(`40`),  // no rule -> default q3
			"q3": json.RawMessage(`"x"`), // -> END
		}},
		{ID: "a-early-end", Answers: map[string]json.RawMessage{
			"q1": json.RawMessage(`"b"`), // -> END
			"q3": json.RawMessage(`"x"`), // unused answer
		}},
		{ID: "c-rule", Answers: map[string]json.RawMessage{
			"q1": json.RawMessage(`"a"`),
			"q2": json.RawMessage(`10`), // rule 0-17 -> END
		}},
	}
	results, err := s.InterpretAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("want 3 results, got %d", len(results))
	}
	// lexicographic order
	if results[0].ResponseID != "a-early-end" || results[2].ResponseID != "c-rule" {
		t.Fatalf("unexpected order: %s, %s, %s", results[0].ResponseID, results[1].ResponseID, results[2].ResponseID)
	}

	early := results[0]
	if early.Status != "ok" {
		t.Fatalf("early end status = %s (%s)", early.Status, early.Error)
	}
	if len(early.Path) != 1 || early.Path[0].Next != END || early.Path[0].Basis != "option:b" {
		t.Fatalf("unexpected early path: %+v", early.Path)
	}
	if !contains(early.UnusedAnswers, "q3") {
		t.Fatalf("q3 should be unused: %v", early.UnusedAnswers)
	}
	if !contains(early.UnvisitedQuestions, "q2") || !contains(early.UnvisitedQuestions, "q3") {
		t.Fatalf("unexpected unvisited: %v", early.UnvisitedQuestions)
	}

	def := results[1]
	if def.Status != "ok" {
		t.Fatalf("default path status = %s (%s)", def.Status, def.Error)
	}
	if def.Path[1].Basis != "default" || def.Path[1].Next != "q3" {
		t.Fatalf("expected default basis, got %+v", def.Path[1])
	}

	rule := results[2]
	if rule.Path[1].Basis != "rule:[0,17]" || rule.Path[1].Next != END {
		t.Fatalf("expected rule basis, got %+v", rule.Path[1])
	}
}

func TestInterpretAnswerErrors(t *testing.T) {
	cases := []struct {
		name    string
		answers map[string]json.RawMessage
		errQ    string
	}{
		{"missing", map[string]json.RawMessage{
			"q1": json.RawMessage(`"a"`),
		}, "q2"},
		{"wrong type string for integer", map[string]json.RawMessage{
			"q1": json.RawMessage(`"a"`),
			"q2": json.RawMessage(`"40"`),
		}, "q2"},
		{"boolean for integer", map[string]json.RawMessage{
			"q1": json.RawMessage(`"a"`),
			"q2": json.RawMessage(`true`),
		}, "q2"},
		{"float for integer", map[string]json.RawMessage{
			"q1": json.RawMessage(`"a"`),
			"q2": json.RawMessage(`4.5`),
		}, "q2"},
		{"out of range", map[string]json.RawMessage{
			"q1": json.RawMessage(`"a"`),
			"q2": json.RawMessage(`101`),
		}, "q2"},
		{"unknown option", map[string]json.RawMessage{
			"q1": json.RawMessage(`"zzz"`),
		}, "q1"},
		{"number for single", map[string]json.RawMessage{
			"q1": json.RawMessage(`1`),
		}, "q1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := validSurvey()
			s.Responses = []Response{{ID: "r", Answers: tc.answers}}
			res := s.Interpret(&s.Responses[0])
			if res.Status != "error" {
				t.Fatalf("expected error, got %s", res.Status)
			}
			if res.ErrorQuestionID != tc.errQ {
				t.Fatalf("expected error at %s, got %s: %s", tc.errQ, res.ErrorQuestionID, res.Error)
			}
			if len(res.Path) == 0 || res.Path[len(res.Path)-1].QuestionID != tc.errQ {
				t.Fatalf("last path step should record %s: %+v", tc.errQ, res.Path)
			}
		})
	}
}

func TestDuplicateResponseID(t *testing.T) {
	s := validSurvey()
	s.Responses = []Response{
		{ID: "r", Answers: map[string]json.RawMessage{}},
		{ID: "r", Answers: map[string]json.RawMessage{}},
	}
	_, err := s.InterpretAll()
	if err == nil {
		t.Fatal("expected duplicate response id error")
	}
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
