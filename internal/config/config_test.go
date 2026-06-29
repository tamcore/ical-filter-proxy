package config

import (
	"testing"
)

func TestParseScalarAndListVal(t *testing.T) {
	raw := []byte(`
work:
  ical_url: https://example.com/work.ics
  api_key: secret
  timezone: Europe/London
  rules:
    - field: summary
      operator: startswith
      val:
        - Planning
        - Standup
    - field: start_time
      operator: not-equals
      val: "09:00"
    - field: blocking
      operator: equals
      val: true
`)
	cfg, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	cal := cfg["work"]
	if cal == nil {
		t.Fatal("calendar \"work\" missing")
	}
	if cal.ICalURL != "https://example.com/work.ics" || cal.APIKey != "secret" {
		t.Fatalf("unexpected calendar fields: %+v", cal)
	}
	if len(cal.Rules) != 3 {
		t.Fatalf("want 3 rules, got %d", len(cal.Rules))
	}
	if got := cal.Rules[0].Val; len(got) != 2 || got[0] != "Planning" || got[1] != "Standup" {
		t.Fatalf("list val not normalized: %v", got)
	}
	if got := cal.Rules[1].Val; len(got) != 1 || got[0] != "09:00" {
		t.Fatalf("scalar val not normalized: %v", got)
	}
	if got := cal.Rules[2].Val; len(got) != 1 || got[0] != "true" {
		t.Fatalf("bool val not stringified: %v", got)
	}
}

func TestParseEnvSubstitution(t *testing.T) {
	t.Setenv("ICAL_FILTER_PROXY_URL", "https://env.example.com/c.ics")
	t.Setenv("ICAL_FILTER_PROXY_KEY", "envkey")
	raw := []byte(`
home:
  ical_url: ${ICAL_FILTER_PROXY_URL}
  api_key: ${ICAL_FILTER_PROXY_KEY}
`)
	cfg, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg["home"].ICalURL != "https://env.example.com/c.ics" {
		t.Fatalf("env url not substituted: %q", cfg["home"].ICalURL)
	}
	if cfg["home"].APIKey != "envkey" {
		t.Fatalf("env key not substituted: %q", cfg["home"].APIKey)
	}
}

func TestParseUnsetEnvBecomesEmpty(t *testing.T) {
	// Unset placeholder -> empty ical_url -> validation failure.
	raw := []byte("home:\n  ical_url: ${ICAL_FILTER_PROXY_DOES_NOT_EXIST}\n")
	if _, err := Parse(raw); err == nil {
		t.Fatal("want validation error for empty ical_url, got nil")
	}
}

func TestValidateErrors(t *testing.T) {
	cases := map[string]string{
		"no calendars": ``,
		"missing url":  "work:\n  api_key: x\n",
		"non-http url": "work:\n  ical_url: ftp://example.com/c.ics\n",
		"relative url": "work:\n  ical_url: /local/path.ics\n",
		"bad timezone": "work:\n  ical_url: https://e.com/c.ics\n  timezone: Mars/Olympus\n",
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(raw)); err == nil {
				t.Fatalf("want error for %q, got nil", name)
			}
		})
	}
}

func TestLocationDefault(t *testing.T) {
	cal := &Calendar{}
	loc, err := cal.Location()
	if err != nil {
		t.Fatalf("Location: %v", err)
	}
	if loc.String() != "UTC" {
		t.Fatalf("want UTC default, got %s", loc)
	}
}

func TestLocationNamed(t *testing.T) {
	cal := &Calendar{Timezone: "Europe/Berlin"}
	loc, err := cal.Location()
	if err != nil {
		t.Fatalf("Location: %v", err)
	}
	if loc.String() != "Europe/Berlin" {
		t.Fatalf("want Europe/Berlin, got %s", loc)
	}
}
