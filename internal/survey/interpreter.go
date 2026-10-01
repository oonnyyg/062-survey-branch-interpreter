package survey

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ValidateResponses checks that every response has a unique non-empty id.
// Answer values themselves are validated during interpretation.
func (s *Survey) ValidateResponses() error {
	seen := make(map[string]int)
	for i := range s.Responses {
		r := &s.Responses[i]
		loc := fmt.Sprintf("responses[%d].id", i)
		if strings.TrimSpace(r.ID) == "" {
			return defErr(loc, "response id must be non-empty")
		}
		if first, ok := seen[r.ID]; ok {
			return defErr(loc, "duplicate response id %q (first used at responses[%d])", r.ID, first)
		}
		seen[r.ID] = i
	}
	return nil
}

func (s *Survey) question(id string) *Question {
	for i := range s.Questions {
		if s.Questions[i].ID == id {
			return &s.Questions[i]
		}
	}
	return nil
}

// Interpret walks one response from the entry question. Unvisited questions
// are never value-checked; answers given for them are listed as unused.
func (s *Survey) Interpret(resp *Response) ResponseResult {
	res := ResponseResult{
		ResponseID:         resp.ID,
		Path:               []Step{},
		UnusedAnswers:      []string{},
		UnvisitedQuestions: []string{},
	}

	visited := make(map[string]bool)
	current := s.Entry
	for current != END {
		q := s.question(current)
		visited[current] = true

		raw, answered := resp.Answers[current]
		if !answered {
			res.failed(current, "missing answer for question %q", current)
			break
		}

		if q.Type == "single" {
			optID, err := decodeString(raw)
			if err != nil {
				res.failed(current, "question %q expects a single-choice option id (string), got %s", current, describeJSON(raw))
				break
			}
			opt := findOption(q, optID)
			if opt == nil {
				res.failed(current, "answer %q is not a valid option of question %q", optID, current)
				break
			}
			res.Path = append(res.Path, Step{
				QuestionID: current,
				Answer:     optID,
				Basis:      "option:" + optID,
				Next:       opt.Next,
			})
			current = opt.Next
			continue
		}

		n, err := decodeInt(raw)
		if err != nil {
			res.failed(current, "question %q expects an integer, got %s", current, describeJSON(raw))
			break
		}
		if n < *q.Min || n > *q.Max {
			res.failed(current, "answer %d for question %q is outside legal range [%d,%d]", n, current, *q.Min, *q.Max)
			break
		}
		rule := matchRule(q, n)
		var basis, next string
		if rule != nil {
			basis = fmt.Sprintf("rule:[%d,%d]", rule.Min, rule.Max)
			next = rule.Next
		} else {
			basis = "default"
			next = q.DefaultNext
		}
		res.Path = append(res.Path, Step{
			QuestionID: current,
			Answer:     n,
			Basis:      basis,
			Next:       next,
		})
		current = next
	}

	if res.Status == "" {
		res.Status = "ok"
	}

	for _, q := range s.Questions {
		if !visited[q.ID] {
			res.UnvisitedQuestions = append(res.UnvisitedQuestions, q.ID)
		}
	}
	sort.Strings(res.UnvisitedQuestions)

	for ansID := range resp.Answers {
		if !visited[ansID] {
			res.UnusedAnswers = append(res.UnusedAnswers, ansID)
		}
	}
	sort.Strings(res.UnusedAnswers)
	return res
}

func (r *ResponseResult) failed(questionID, format string, args ...any) {
	r.Status = "error"
	r.ErrorQuestionID = questionID
	r.Error = fmt.Sprintf(format, args...)
	r.Path = append(r.Path, Step{QuestionID: questionID})
}

func findOption(q *Question, id string) *Option {
	for j := range q.Options {
		if q.Options[j].ID == id {
			return &q.Options[j]
		}
	}
	return nil
}

func matchRule(q *Question, n int) *RangeRule {
	for j := range q.Rules {
		if n >= q.Rules[j].Min && n <= q.Rules[j].Max {
			return &q.Rules[j]
		}
	}
	return nil
}

func decodeString(raw json.RawMessage) (string, error) {
	var v string
	err := json.Unmarshal(raw, &v)
	return v, err
}

// decodeInt accepts only JSON integers (or numbers with no fraction/exponent).
// JSON booleans are explicitly rejected: unmarshalling into json.Number first
// keeps the raw token, then we require an integer literal.
func decodeInt(raw json.RawMessage) (int, error) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return 0, err
	}
	num, ok := v.(json.Number)
	if !ok {
		return 0, fmt.Errorf("not a number")
	}
	if _, err := num.Int64(); err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(num.String())
	if err != nil {
		return 0, err
	}
	return n, nil
}

func describeJSON(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if len(s) > 40 {
		return s[:37] + "..."
	}
	return s
}

// InterpretAll validates ids and interprets every response independently,
// returning results ordered lexicographically by response id.
func (s *Survey) InterpretAll() ([]ResponseResult, error) {
	if err := s.ValidateResponses(); err != nil {
		return nil, err
	}
	out := make([]ResponseResult, 0, len(s.Responses))
	for i := range s.Responses {
		out = append(out, s.Interpret(&s.Responses[i]))
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ResponseID < out[b].ResponseID })
	return out, nil
}
