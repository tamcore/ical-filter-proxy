// Package filter implements the event filtering rule engine: it compiles the
// config rules once at startup (validating operators, fields and regexes) and
// evaluates them against calendar events.
package filter

// Event is the field projection a rule is evaluated against. The calendar
// package builds it from a VEVENT, resolving time/date fields into the
// calendar's timezone. This keeps the filter package free of any iCal library
// dependency.
type Event struct {
	Summary     string
	Description string
	StartTime   string // HH:MM (24h) in the calendar timezone
	EndTime     string // HH:MM (24h) in the calendar timezone
	StartDate   string // YYYY-MM-DD in the calendar timezone
	EndDate     string // YYYY-MM-DD in the calendar timezone
	Blocking    bool   // true when TRANSP is OPAQUE or absent
}

// Supported rule field names.
const (
	FieldSummary     = "summary"
	FieldDescription = "description"
	FieldStartTime   = "start_time"
	FieldEndTime     = "end_time"
	FieldStartDate   = "start_date"
	FieldEndDate     = "end_date"
	FieldBlocking    = "blocking"
)

// fieldValue returns the string projection of the named field for matching.
// Blocking is stringified to "true"/"false" so it composes with the standard
// string operators (upstream uses `equals` on it).
func (e Event) fieldValue(field string) (string, bool) {
	switch field {
	case FieldSummary:
		return e.Summary, true
	case FieldDescription:
		return e.Description, true
	case FieldStartTime:
		return e.StartTime, true
	case FieldEndTime:
		return e.EndTime, true
	case FieldStartDate:
		return e.StartDate, true
	case FieldEndDate:
		return e.EndDate, true
	case FieldBlocking:
		if e.Blocking {
			return "true", true
		}
		return "false", true
	default:
		return "", false
	}
}

func isKnownField(field string) bool {
	switch field {
	case FieldSummary, FieldDescription, FieldStartTime, FieldEndTime,
		FieldStartDate, FieldEndDate, FieldBlocking:
		return true
	default:
		return false
	}
}
