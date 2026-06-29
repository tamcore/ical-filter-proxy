// Package calendar fetches upstream iCalendar feeds, applies the compiled
// filters and alarm rules, and serializes the result. It is the only package
// that depends on the iCal library.
package calendar

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/tamcore/ical-filter-proxy/internal/config"
)

// AlarmSet is the compiled, validated alarm configuration for a calendar.
type AlarmSet struct {
	ClearExisting bool
	Triggers      []string // canonical ICS TRIGGER duration values
}

var (
	// iso8601Duration is a permissive RFC5545 DURATION matcher (weeks, or
	// days plus an optional time part). Case-insensitive; sign optional.
	iso8601Duration = regexp.MustCompile(`(?i)^[+-]?P(\d+W|(\d+D)?(T(\d+H)?(\d+M)?(\d+S)?)?)$`)
	hasDigit        = regexp.MustCompile(`\d`)
	// naturalTrigger matches "2 days", "5 hours", "10 minutes", etc.
	naturalTrigger = regexp.MustCompile(`(?i)^(\d+)\s+(second|minute|hour|day|week)s?$`)
)

// CompileAlarms validates the alarm config at startup and returns the canonical
// trigger values. Natural-language triggers are converted to negative
// (before-event) ISO8601 durations, matching upstream semantics.
func CompileAlarms(a *config.Alarms) (*AlarmSet, error) {
	if a == nil {
		return nil, nil
	}
	set := &AlarmSet{ClearExisting: a.ClearExisting}
	for _, t := range a.Triggers {
		canonical, err := parseTrigger(t)
		if err != nil {
			return nil, err
		}
		set.Triggers = append(set.Triggers, canonical)
	}
	return set, nil
}

func parseTrigger(s string) (string, error) {
	s = strings.TrimSpace(s)
	if iso8601Duration.MatchString(s) && hasDigit.MatchString(s) {
		return strings.ToUpper(s), nil
	}
	if m := naturalTrigger.FindStringSubmatch(s); m != nil {
		n, unit := m[1], strings.ToLower(m[2])
		switch unit {
		case "week":
			return "-P" + n + "W", nil
		case "day":
			return "-P" + n + "D", nil
		case "hour":
			return "-PT" + n + "H", nil
		case "minute":
			return "-PT" + n + "M", nil
		case "second":
			return "-PT" + n + "S", nil
		}
	}
	return "", fmt.Errorf("unknown trigger pattern: %q", s)
}
