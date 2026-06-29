package calendar

import (
	"time"

	ics "github.com/arran4/golang-ical"

	"github.com/tamcore/ical-filter-proxy/internal/config"
	"github.com/tamcore/ical-filter-proxy/internal/filter"
)

// Transformer holds the compiled filter, alarm set and timezone for one
// calendar. It is built once at startup and reused across requests; Transform
// operates on a freshly parsed (request-scoped) calendar, so its in-place
// mutations are not shared.
type Transformer struct {
	matcher *filter.Matcher
	alarms  *AlarmSet
	loc     *time.Location
}

// Compile prepares a calendar's transformer at startup, validating its rules,
// timezone and alarm triggers so bad config fails fast.
func Compile(cal *config.Calendar) (*Transformer, error) {
	matcher, err := filter.Compile(cal.Rules)
	if err != nil {
		return nil, err
	}
	loc, err := cal.Location()
	if err != nil {
		return nil, err
	}
	alarms, err := CompileAlarms(cal.Alarms)
	if err != nil {
		return nil, err
	}
	return &Transformer{matcher: matcher, alarms: alarms, loc: loc}, nil
}

// Transform returns a new calendar containing only the events that pass the
// filter (with alarms applied), preserving all other components (e.g.
// VTIMEZONE) and the calendar-level properties.
func (t *Transformer) Transform(src *ics.Calendar) *ics.Calendar {
	out := &ics.Calendar{CalendarProperties: src.CalendarProperties}
	for _, comp := range src.Components {
		ev, ok := comp.(*ics.VEvent)
		if !ok {
			out.Components = append(out.Components, comp)
			continue
		}
		if !t.matcher.Match(buildEvent(ev, t.loc)) {
			continue
		}
		t.applyAlarms(ev)
		out.Components = append(out.Components, ev)
	}
	return out
}

func (t *Transformer) applyAlarms(ev *ics.VEvent) {
	if t.alarms == nil {
		return
	}
	if t.alarms.ClearExisting {
		kept := make([]ics.Component, 0, len(ev.Components))
		for _, c := range ev.Components {
			if _, isAlarm := c.(*ics.VAlarm); !isAlarm {
				kept = append(kept, c)
			}
		}
		ev.Components = kept
	}
	summary := propVal(ev, ics.ComponentPropertySummary)
	for _, trigger := range t.alarms.Triggers {
		a := ev.AddAlarm()
		a.SetAction(ics.ActionDisplay)
		a.SetProperty(ics.ComponentPropertyDescription, summary)
		a.SetTrigger(trigger)
	}
}

// buildEvent projects a VEvent into the filter.Event used by the rule engine,
// resolving time/date fields into the calendar's timezone.
func buildEvent(ev *ics.VEvent, loc *time.Location) filter.Event {
	e := filter.Event{
		Summary:     propVal(ev, ics.ComponentPropertySummary),
		Description: propVal(ev, ics.ComponentPropertyDescription),
		Blocking:    isBlocking(ev),
	}
	if start, ok := eventStart(ev); ok {
		s := start.In(loc)
		e.StartTime = s.Format("15:04")
		e.StartDate = s.Format("2006-01-02")
	}
	if end, ok := eventEnd(ev); ok {
		en := end.In(loc)
		e.EndTime = en.Format("15:04")
		e.EndDate = en.Format("2006-01-02")
	}
	return e
}

func propVal(ev *ics.VEvent, p ics.ComponentProperty) string {
	if prop := ev.GetProperty(p); prop != nil {
		return prop.Value
	}
	return ""
}

// isBlocking reports whether the event occupies time: true when TRANSP is
// OPAQUE or absent, false only when explicitly TRANSPARENT.
func isBlocking(ev *ics.VEvent) bool {
	prop := ev.GetProperty(ics.ComponentPropertyTransp)
	return prop == nil || prop.Value != string(ics.TransparencyTransparent)
}

func eventStart(ev *ics.VEvent) (time.Time, bool) {
	if t, err := ev.GetStartAt(); err == nil {
		return t, true
	}
	if t, err := ev.GetAllDayStartAt(); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func eventEnd(ev *ics.VEvent) (time.Time, bool) {
	if t, err := ev.GetEndAt(); err == nil {
		return t, true
	}
	if t, err := ev.GetAllDayEndAt(); err == nil {
		return t, true
	}
	return time.Time{}, false
}

// Serialize renders a calendar with RFC5545-compliant CRLF line endings, as the
// upstream Ruby implementation does.
func Serialize(cal *ics.Calendar) string {
	return cal.Serialize(ics.WithNewLineWindows)
}
