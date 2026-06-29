package filter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/tamcore/ical-filter-proxy/internal/config"
)

// Operator base names (each may be prefixed with "not-" to negate).
const (
	OpEquals     = "equals"
	OpStartsWith = "startswith"
	OpIncludes   = "includes"
	OpMatches    = "matches"

	negPrefix = "not-"
)

// compiledRule is a single rule prepared for evaluation.
type compiledRule struct {
	field   string
	op      string // base operator, without the not- prefix
	negate  bool
	vals    []string         // for equals/startswith/includes
	regexes []*regexp.Regexp // for matches
}

// Matcher evaluates a calendar's compiled rules against events. The zero value
// (no rules) keeps every event, matching upstream behavior.
type Matcher struct {
	rules []compiledRule
}

// Compile prepares and validates a calendar's rules at startup: it checks field
// and operator names and compiles any regexes, so bad config fails fast instead
// of crashing on the first request.
func Compile(rules []config.Rule) (*Matcher, error) {
	compiled := make([]compiledRule, 0, len(rules))
	for i, r := range rules {
		if !isKnownField(r.Field) {
			return nil, fmt.Errorf("rule %d: unknown field %q", i, r.Field)
		}
		base, negate := strings.TrimPrefix(r.Operator, negPrefix), strings.HasPrefix(r.Operator, negPrefix)
		switch base {
		case OpEquals, OpStartsWith, OpIncludes, OpMatches:
		default:
			return nil, fmt.Errorf("rule %d: unknown operator %q", i, r.Operator)
		}
		if len(r.Val) == 0 {
			return nil, fmt.Errorf("rule %d: val must not be empty", i)
		}
		cr := compiledRule{field: r.Field, op: base, negate: negate}
		if base == OpMatches {
			for _, v := range r.Val {
				re, err := translateRegex(v)
				if err != nil {
					return nil, fmt.Errorf("rule %d: %w", i, err)
				}
				cr.regexes = append(cr.regexes, re)
			}
		} else {
			cr.vals = r.Val
		}
		compiled = append(compiled, cr)
	}
	return &Matcher{rules: compiled}, nil
}

// Match reports whether the event satisfies every rule (logical AND).
func (m *Matcher) Match(e Event) bool {
	for _, r := range m.rules {
		if !r.eval(e) {
			return false
		}
	}
	return true
}

// eval applies one rule: the base operator is OR-ed across the rule's values,
// then the not- prefix (if any) negates that result.
func (r compiledRule) eval(e Event) bool {
	value, ok := e.fieldValue(r.field)
	if !ok {
		return false
	}
	matched := r.anyMatch(value)
	if r.negate {
		return !matched
	}
	return matched
}

func (r compiledRule) anyMatch(value string) bool {
	if r.op == OpMatches {
		for _, re := range r.regexes {
			if re.MatchString(value) {
				return true
			}
		}
		return false
	}
	for _, v := range r.vals {
		switch r.op {
		case OpEquals:
			if value == v {
				return true
			}
		case OpStartsWith:
			if strings.HasPrefix(value, v) {
				return true
			}
		case OpIncludes:
			if strings.Contains(value, v) {
				return true
			}
		}
	}
	return false
}
