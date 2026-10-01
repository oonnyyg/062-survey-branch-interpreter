package survey

import (
	"fmt"
	"sort"
	"strings"
)

// targetRef pairs a jump target with the definition location that uses it.
type targetRef struct {
	target string
	loc    string
}

// ValidateGraph verifies that every jump target exists (or is END), that the
// directed graph has no cycle, and that every question is reachable from the
// entry question.
func (s *Survey) ValidateGraph() error {
	index := make(map[string]int, len(s.Questions))
	for i := range s.Questions {
		index[s.Questions[i].ID] = i
	}
	if _, ok := index[s.Entry]; !ok {
		return defErr("entry", "entry question %q does not exist", s.Entry)
	}

	for i := range s.Questions {
		q := &s.Questions[i]
		loc := fmt.Sprintf("questions[%d]", i)
		refs := []targetRef{{q.DefaultNext, loc + ".default_next"}}
		if q.Type == "single" {
			for j := range q.Options {
				refs = append(refs, targetRef{
					q.Options[j].Next,
					fmt.Sprintf("%s.options[%d].next", loc, j),
				})
			}
		} else {
			for j := range q.Rules {
				refs = append(refs, targetRef{
					q.Rules[j].Next,
					fmt.Sprintf("%s.rules[%d].next", loc, j),
				})
			}
		}
		for _, ref := range refs {
			if ref.target == END {
				continue
			}
			if _, ok := index[ref.target]; !ok {
				return defErr(ref.loc, "unknown jump target %q in question %q", ref.target, q.ID)
			}
		}
	}

	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int, len(s.Questions))
	var stack []string
	var dfs func(id string) error
	dfs = func(id string) error {
		color[id] = gray
		stack = append(stack, id)
		for _, next := range s.outgoing(id) {
			if next == END {
				continue
			}
			switch color[next] {
			case gray:
				cycleStart := 0
				for stack[cycleStart] != next {
					cycleStart++
				}
				path := append(append([]string{}, stack[cycleStart:]...), next)
				return defErr("questions", "jump cycle detected: %s", strings.Join(path, " -> "))
			case white:
				if err := dfs(next); err != nil {
					return err
				}
			}
		}
		stack = stack[:len(stack)-1]
		color[id] = black
		return nil
	}
	if err := dfs(s.Entry); err != nil {
		return err
	}

	var unreachable []string
	for _, q := range s.Questions {
		if color[q.ID] == white {
			unreachable = append(unreachable, q.ID)
		}
	}
	if len(unreachable) > 0 {
		sort.Strings(unreachable)
		return defErr("questions", "question(s) unreachable from entry %q: %s", s.Entry, strings.Join(unreachable, ", "))
	}
	return nil
}

// outgoing returns jump targets of a question in a deterministic order:
// default first, then options or rules in declared order.
func (s *Survey) outgoing(id string) []string {
	for i := range s.Questions {
		q := &s.Questions[i]
		if q.ID != id {
			continue
		}
		out := []string{q.DefaultNext}
		if q.Type == "single" {
			for j := range q.Options {
				out = append(out, q.Options[j].Next)
			}
		} else {
			for j := range q.Rules {
				out = append(out, q.Rules[j].Next)
			}
		}
		return out
	}
	return nil
}
