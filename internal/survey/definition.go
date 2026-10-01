package survey

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

// EndID is the reserved target marking the end of a survey path.
const EndID = "END"

// QuestionType enumerates the supported question kinds.
type QuestionType string

const (
	TypeSingle  QuestionType = "single"
	TypeInteger QuestionType = "integer"
)

// Option is one choice of a single-choice question.
type Option struct {
	Value string `json:"value"`
	Next  string `json:"next"`
}

// RangeRule maps an inclusive integer range to a jump target.
type RangeRule struct {
	Min  int    `json:"min"`
	Max  int    `json:"max"`
	Next string `json:"next"`
}

// Question is a single survey question node.
type Question struct {
	ID          string       `json:"id"`
	Type        QuestionType `json:"type"`
	DefaultNext string       `json:"default_next"`
	Options     []Option     `json:"options,omitempty"`
	Min         *int         `json:"min,omitempty"`
	Max         *int         `json:"max,omitempty"`
	Rules       []RangeRule  `json:"rules,omitempty"`
}

// Response is one submitted answer sheet.
type Response struct {
	ID      string                     `json:"id"`
	Answers map[string]json.RawMessage `json:"answers"`
}

// Definition is the root of the input document.
type Definition struct {
	Entry     string     `json:"entry"`
	Questions []Question `json:"questions"`
	Responses []Response `json:"responses"`
}

// ParseDefinition decodes and validates the input document.
func ParseDefinition(data []byte) (*Definition, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	var def Definition
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if err := def.Validate(); err != nil {
		return nil, err
	}
	return &def, nil
}

// Validate checks structural and graph constraints of the definition.
func (d *Definition) Validate() error {
	if d.Entry == "" {
		return fmt.Errorf("entry question id is empty")
	}
	if d.Entry == EndID {
		return fmt.Errorf("entry question id must not be %q", EndID)
	}
	byID := make(map[string]*Question, len(d.Questions))
	for i := range d.Questions {
		q := &d.Questions[i]
		if q.ID == "" {
			return fmt.Errorf("questions[%d]: id is empty", i)
		}
		if q.ID == EndID {
			return fmt.Errorf("questions[%d] (%s): id must not be %q", i, q.ID, EndID)
		}
		if _, dup := byID[q.ID]; dup {
			return fmt.Errorf("questions[%d]: duplicate question id %q", i, q.ID)
		}
		if err := q.validateShape(); err != nil {
			return fmt.Errorf("questions[%d] (%s): %w", i, q.ID, err)
		}
		byID[q.ID] = q
	}
	if _, ok := byID[d.Entry]; !ok {
		return fmt.Errorf("entry question %q does not exist", d.Entry)
	}
	for i := range d.Questions {
		q := &d.Questions[i]
		if err := q.validateTargets(byID); err != nil {
			return fmt.Errorf("question %q: %w", q.ID, err)
		}
	}
	if err := checkAcyclicAndReachable(d.Entry, byID); err != nil {
		return err
	}
	seenResp := make(map[string]bool, len(d.Responses))
	for i, r := range d.Responses {
		if r.ID == "" {
			return fmt.Errorf("responses[%d]: id is empty", i)
		}
		if seenResp[r.ID] {
			return fmt.Errorf("responses[%d]: duplicate response id %q", i, r.ID)
		}
		seenResp[r.ID] = true
	}
	return nil
}

func (q *Question) validateShape() error {
	if q.DefaultNext == "" {
		return fmt.Errorf("default_next is empty")
	}
	switch q.Type {
	case TypeSingle:
		if len(q.Options) == 0 {
			return fmt.Errorf("single-choice question must have options")
		}
		seen := make(map[string]bool, len(q.Options))
		for i, o := range q.Options {
			if o.Value == "" {
				return fmt.Errorf("options[%d]: value is empty", i)
			}
			if seen[o.Value] {
				return fmt.Errorf("options[%d]: duplicate option value %q", i, o.Value)
			}
			seen[o.Value] = true
			if o.Next == "" {
				return fmt.Errorf("options[%d]: next is empty", i)
			}
		}
	case TypeInteger:
		if q.Min == nil || q.Max == nil {
			return fmt.Errorf("integer question requires min and max")
		}
		if *q.Min > *q.Max {
			return fmt.Errorf("invalid range: min %d > max %d", *q.Min, *q.Max)
		}
		for i, r := range q.Rules {
			if r.Min > r.Max {
				return fmt.Errorf("rules[%d]: invalid range [%d, %d]", i, r.Min, r.Max)
			}
			if r.Min < *q.Min || r.Max > *q.Max {
				return fmt.Errorf("rules[%d]: range [%d, %d] outside legal range [%d, %d]", i, r.Min, r.Max, *q.Min, *q.Max)
			}
			if r.Next == "" {
				return fmt.Errorf("rules[%d]: next is empty", i)
			}
			for j := 0; j < i; j++ {
				p := q.Rules[j]
				if r.Min <= p.Max && p.Min <= r.Max {
					return fmt.Errorf("rules[%d] overlaps rules[%d]", i, j)
				}
			}
		}
	default:
		return fmt.Errorf("unknown question type %q", q.Type)
	}
	return nil
}

func (q *Question) validateTargets(byID map[string]*Question) error {
	check := func(target, where string) error {
		if target == EndID {
			return nil
		}
		if _, ok := byID[target]; !ok {
			return fmt.Errorf("%s: unknown target %q", where, target)
		}
		return nil
	}
	if err := check(q.DefaultNext, "default_next"); err != nil {
		return err
	}
	for i, o := range q.Options {
		if err := check(o.Next, fmt.Sprintf("options[%d]", i)); err != nil {
			return err
		}
	}
	for i, r := range q.Rules {
		if err := check(r.Next, fmt.Sprintf("rules[%d]", i)); err != nil {
			return err
		}
	}
	return nil
}

// checkAcyclicAndReachable walks the jump graph from the entry, rejecting
// cycles and questions unreachable from the entry.
func checkAcyclicAndReachable(entry string, byID map[string]*Question) error {
	const (
		white = 0 // unvisited
		gray  = 1 // on current DFS stack
		black = 2 // fully explored
	)
	color := make(map[string]int, len(byID))
	var visit func(id string) error
	visit = func(id string) error {
		color[id] = gray
		for _, next := range targetsOf(byID[id]) {
			if next == EndID {
				continue
			}
			switch color[next] {
			case gray:
				return fmt.Errorf("cycle detected involving question %q", next)
			case white:
				if err := visit(next); err != nil {
					return err
				}
			}
		}
		color[id] = black
		return nil
	}
	if err := visit(entry); err != nil {
		return err
	}
	unreachable := []string{}
	for id := range byID {
		if color[id] == white {
			unreachable = append(unreachable, id)
		}
	}
	if len(unreachable) > 0 {
		sort.Strings(unreachable)
		return fmt.Errorf("questions unreachable from entry %q: %v", entry, unreachable)
	}
	return nil
}

func targetsOf(q *Question) []string {
	targets := []string{q.DefaultNext}
	for _, o := range q.Options {
		targets = append(targets, o.Next)
	}
	for _, r := range q.Rules {
		targets = append(targets, r.Next)
	}
	return targets
}
