package config

import (
	"fmt"
	"net/url"
	"time"
)

// Validate performs structural, dependency-free startup checks: every calendar
// needs a usable ical_url and (if set) a loadable timezone. Operator/field/regex
// and alarm-trigger validation happen when the filter and alarm sets are
// compiled at startup (see internal/filter and internal/calendar), keeping this
// package free of import cycles.
func (c Config) Validate() error {
	if len(c) == 0 {
		return fmt.Errorf("config defines no calendars")
	}
	for name, cal := range c {
		if cal == nil {
			return fmt.Errorf("calendar %q is empty", name)
		}
		if cal.ICalURL == "" {
			return fmt.Errorf("calendar %q: ical_url is required", name)
		}
		u, err := url.Parse(cal.ICalURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("calendar %q: ical_url must be an absolute http(s) URL", name)
		}
		if cal.Timezone != "" {
			if _, err := time.LoadLocation(cal.Timezone); err != nil {
				return fmt.Errorf("calendar %q: invalid timezone %q: %w", name, cal.Timezone, err)
			}
		}
	}
	return nil
}

// Location returns the calendar's timezone, defaulting to UTC when unset.
func (cal *Calendar) Location() (*time.Location, error) {
	if cal.Timezone == "" {
		return time.UTC, nil
	}
	return time.LoadLocation(cal.Timezone)
}
