package survey

import (
	"encoding/json"
	"fmt"
	"sort"
)

// Step records one visited question and the reason for the jump taken.
type Step struct {
	Question string          `json:"question"`
	Answer   json.RawMessage `json:"answer"`
	Matched  string          `json:"matched"`
	Next     string          `json:"next"`
}

// AnswerError describes the first problem found while interpreting a response.
type AnswerError struct {
	Question string `json:"question"`
	Message  string `json:"message"`
}

// Result is the interpretation outcome of one response.
type Result struct {
	ID      string       `json:"id"`
	Status  string       `json:"status"` // "completed" or "error"
	Path    []Step       `json:"path"`
	Error   *AnswerError `json:"error,omitempty"`
	Unused  []string     `json:"unused"`
}

// Interpret walks the survey graph for one response starting at the entry.
func (d *Definition) Interpret(r *Response) Result {
	byID := make(map[string]*Question, len(d.Questions))
	for i := range d.Questions {
		byID[d.Questions[i].ID] = &d.Questions[i]
	}
	res := Result{ID: r.ID, Status: "completed", Path: []Step{}, Unused: []string{}}
	visited := map[string]bool{}
	current := d.Entry
	for current != EndID {
		q := byID[current]
		visited[current] = true
		raw, ok := r.Answers[current]
		if !ok {
			res.fail(current, "missing answer")
			break
		}
		step, err := resolveStep(q, raw)
		if err != nil {
			res.fail(current, err.Error())
			break
		}
		res.Path = append(res.Path, step)
		current = step.Next
	}
	for id := range r.Answers {
		if !visited[id] {
			res.Unused = append(res.Unused, id)
		}
	}
	sort.Strings(res.Unused)
	return res
}

func (r *Result) fail(question, message string) {
	r.Status = "error"
	r.Error = &AnswerError{Question: question, Message: message}
}

func resolveStep(q *Question, raw json.RawMessage) (Step, error) {
	switch q.Type {
	case TypeSingle:
		var answer string
		if err := json.Unmarshal(raw, &answer); err != nil {
			return Step{}, fmt.Errorf("answer must be a string option: %v", err)
		}
		for _, o := range q.Options {
			if o.Value == answer {
				return Step{Question: q.ID, Answer: raw, Matched: fmt.Sprintf("option %q", answer), Next: o.Next}, nil
			}
		}
		return Step{}, fmt.Errorf("answer %q is not a valid option", answer)
	case TypeInteger:
		var answer int
		if err := json.Unmarshal(raw, &answer); err != nil {
			return Step{}, fmt.Errorf("answer must be an integer: %v", err)
		}
		if answer < *q.Min || answer > *q.Max {
			return Step{}, fmt.Errorf("answer %d outside legal range [%d, %d]", answer, *q.Min, *q.Max)
		}
		for _, rule := range q.Rules {
			if answer >= rule.Min && answer <= rule.Max {
				return Step{Question: q.ID, Answer: raw, Matched: fmt.Sprintf("range [%d, %d]", rule.Min, rule.Max), Next: rule.Next}, nil
			}
		}
		return Step{Question: q.ID, Answer: raw, Matched: "default", Next: q.DefaultNext}, nil
	}
	return Step{}, fmt.Errorf("unknown question type %q", q.Type)
}

// InterpretAll interprets every response and returns results sorted by id.
func (d *Definition) InterpretAll() []Result {
	results := make([]Result, 0, len(d.Responses))
	for i := range d.Responses {
		results = append(results, d.Interpret(&d.Responses[i]))
	}
	sort.Slice(results, func(i, j int) bool { return results[i].ID < results[j].ID })
	return results
}
