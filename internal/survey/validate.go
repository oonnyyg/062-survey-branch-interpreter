package survey

import (
	"fmt"
	"strings"
)

// END is the reserved terminal target and must not be used as a question ID.
const END = "END"

// DefError describes an invalid survey definition and where the problem is.
type DefError struct {
	Location string
	Message  string
}

func (e *DefError) Error() string {
	if e.Location == "" {
		return e.Message
	}
	return e.Location + ": " + e.Message
}

func defErr(location, format string, args ...any) error {
	return &DefError{Location: location, Message: fmt.Sprintf(format, args...)}
}

// Validate checks all static constraints of the survey definition:
// unique IDs, valid question types, distinct non-empty options with valid
// jumps, and integer bounds with non-overlapping in-range rules.
// Graph-level checks (unknown targets, cycles, reachability) are separate.
func (s *Survey) Validate() error {
	if strings.TrimSpace(s.Entry) == "" {
		return defErr("entry", "entry question must be a non-empty question id")
	}
	if s.Entry == END {
		return defErr("entry", "%q is reserved and cannot be a question id", END)
	}
	if len(s.Questions) == 0 {
		return defErr("questions", "at least one question is required")
	}

	seen := make(map[string]int)
	for i := range s.Questions {
		q := &s.Questions[i]
		loc := fmt.Sprintf("questions[%d]", i)
		idLoc := loc + ".id"
		if strings.TrimSpace(q.ID) == "" {
			return defErr(idLoc, "question id must be non-empty")
		}
		if q.ID == END {
			return defErr(idLoc, "%q is reserved and cannot be a question id", END)
		}
		if first, ok := seen[q.ID]; ok {
			return defErr(idLoc, "duplicate question id %q (first used at questions[%d])", q.ID, first)
		}
		seen[q.ID] = i

		switch q.Type {
		case "single":
			if err := validateSingle(loc, q); err != nil {
				return err
			}
		case "integer":
			if err := validateInteger(loc, q); err != nil {
				return err
			}
		default:
			return defErr(loc+".type", "unknown question type %q (want \"single\" or \"integer\")", q.Type)
		}
	}
	return nil
}

func validateSingle(loc string, q *Question) error {
	if len(q.Options) == 0 {
		return defErr(loc+".options", "single-choice question %q must have at least one option", q.ID)
	}
	ids := make(map[string]int)
	for j, opt := range q.Options {
		oloc := fmt.Sprintf("%s.options[%d]", loc, j)
		if strings.TrimSpace(opt.ID) == "" {
			return defErr(oloc+".id", "option id must be non-empty (question %q)", q.ID)
		}
		if first, ok := ids[opt.ID]; ok {
			return defErr(oloc+".id", "duplicate option id %q in question %q (first used at options[%d])", opt.ID, q.ID, first)
		}
		ids[opt.ID] = j
		if strings.TrimSpace(opt.Next) == "" {
			return defErr(oloc+".next", "option %q of question %q must have a jump target", opt.ID, q.ID)
		}
	}
	return checkDefault(loc, q)
}

func validateInteger(loc string, q *Question) error {
	if q.Min == nil {
		return defErr(loc+".min", "integer question %q requires min", q.ID)
	}
	if q.Max == nil {
		return defErr(loc+".max", "integer question %q requires max", q.ID)
	}
	if *q.Min > *q.Max {
		return defErr(loc, "integer question %q has min %d greater than max %d", q.ID, *q.Min, *q.Max)
	}
	for j, r := range q.Rules {
		rloc := fmt.Sprintf("%s.rules[%d]", loc, j)
		if r.Min > r.Max {
			return defErr(rloc, "rule interval [%d,%d] is invalid: min > max", r.Min, r.Max)
		}
		if r.Min < *q.Min || r.Max > *q.Max {
			return defErr(rloc, "rule interval [%d,%d] is outside legal range [%d,%d]", r.Min, r.Max, *q.Min, *q.Max)
		}
		if strings.TrimSpace(r.Next) == "" {
			return defErr(rloc+".next", "rule must have a jump target")
		}
	}
	for j := 1; j < len(q.Rules); j++ {
		for k := 0; k < j; k++ {
			a, b := q.Rules[k], q.Rules[j]
			if a.Min <= b.Max && b.Min <= a.Max {
				return defErr(fmt.Sprintf("%s.rules[%d]", loc, j),
					"rule interval [%d,%d] overlaps rules[%d] interval [%d,%d]",
					b.Min, b.Max, k, a.Min, a.Max)
			}
		}
	}
	return checkDefault(loc, q)
}

func checkDefault(loc string, q *Question) error {
	if strings.TrimSpace(q.DefaultNext) == "" {
		return defErr(loc+".default_next", "question %q must declare default_next", q.ID)
	}
	return nil
}
