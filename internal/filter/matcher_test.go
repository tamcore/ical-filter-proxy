package filter

import (
	"testing"

	"github.com/tamcore/ical-filter-proxy/internal/config"
)

func mustCompile(t *testing.T, rules []config.Rule) *Matcher {
	t.Helper()
	m, err := Compile(rules)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	return m
}

func TestMatchOperators(t *testing.T) {
	ev := Event{Summary: "Daily Standup", Description: "team sync", StartTime: "09:00", Blocking: true}
	cases := []struct {
		name string
		rule config.Rule
		want bool
	}{
		{"equals true", config.Rule{Field: "summary", Operator: "equals", Val: config.Values{"Daily Standup"}}, true},
		{"equals false", config.Rule{Field: "summary", Operator: "equals", Val: config.Values{"Planning"}}, false},
		{"not-equals", config.Rule{Field: "summary", Operator: "not-equals", Val: config.Values{"Planning"}}, true},
		{"startswith", config.Rule{Field: "summary", Operator: "startswith", Val: config.Values{"Daily"}}, true},
		{"not-startswith", config.Rule{Field: "summary", Operator: "not-startswith", Val: config.Values{"Daily"}}, false},
		{"includes", config.Rule{Field: "description", Operator: "includes", Val: config.Values{"sync"}}, true},
		{"includes false", config.Rule{Field: "description", Operator: "includes", Val: config.Values{"nope"}}, false},
		{"array OR hit", config.Rule{Field: "summary", Operator: "startswith", Val: config.Values{"Planning", "Daily"}}, true},
		{"array OR miss", config.Rule{Field: "summary", Operator: "startswith", Val: config.Values{"Planning", "Sprint"}}, false},
		{"not-equals array", config.Rule{Field: "summary", Operator: "not-equals", Val: config.Values{"Planning", "Sprint"}}, true},
		{"start_time equals", config.Rule{Field: "start_time", Operator: "equals", Val: config.Values{"09:00"}}, true},
		{"blocking true", config.Rule{Field: "blocking", Operator: "equals", Val: config.Values{"true"}}, true},
		{"blocking false miss", config.Rule{Field: "blocking", Operator: "equals", Val: config.Values{"false"}}, false},
		{"matches i flag", config.Rule{Field: "summary", Operator: "matches", Val: config.Values{"/daily standup/i"}}, true},
		{"matches no flag miss", config.Rule{Field: "summary", Operator: "matches", Val: config.Values{"/daily standup/"}}, false},
		{"not-matches", config.Rule{Field: "summary", Operator: "not-matches", Val: config.Values{"/^Planning/"}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := mustCompile(t, []config.Rule{tc.rule})
			if got := m.Match(ev); got != tc.want {
				t.Fatalf("Match=%v want %v", got, tc.want)
			}
		})
	}
}

func TestMatchAllRulesAnded(t *testing.T) {
	ev := Event{Summary: "Daily Standup", StartTime: "09:00"}
	m := mustCompile(t, []config.Rule{
		{Field: "summary", Operator: "startswith", Val: config.Values{"Daily"}},
		{Field: "start_time", Operator: "equals", Val: config.Values{"10:00"}},
	})
	if m.Match(ev) {
		t.Fatal("want false: second rule should fail the AND")
	}
}

func TestEmptyMatcherKeepsAll(t *testing.T) {
	m := mustCompile(t, nil)
	if !m.Match(Event{}) {
		t.Fatal("empty matcher should keep every event")
	}
}

func TestCompileErrors(t *testing.T) {
	cases := map[string]config.Rule{
		"unknown field":    {Field: "location", Operator: "equals", Val: config.Values{"x"}},
		"unknown operator": {Field: "summary", Operator: "regex", Val: config.Values{"x"}},
		"empty val":        {Field: "summary", Operator: "equals", Val: config.Values{}},
		"bad regex form":   {Field: "summary", Operator: "matches", Val: config.Values{"no-slashes"}},
		"bad regex flag":   {Field: "summary", Operator: "matches", Val: config.Values{"/x/x"}},
		"bad regex syntax": {Field: "summary", Operator: "matches", Val: config.Values{"/([/"}},
	}
	for name, rule := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Compile([]config.Rule{rule}); err == nil {
				t.Fatalf("want compile error for %q", name)
			}
		})
	}
}

func TestTranslateRegexMFlagDotall(t *testing.T) {
	// Ruby /m maps to Go (?s): dot should match a newline.
	re, err := translateRegex("/a.b/m")
	if err != nil {
		t.Fatalf("translateRegex: %v", err)
	}
	if !re.MatchString("a\nb") {
		t.Fatal("Ruby /m should make . match newline")
	}
}
