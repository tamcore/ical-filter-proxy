package calendar

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ics "github.com/arran4/golang-ical"

	"github.com/tamcore/ical-filter-proxy/internal/config"
)

const sampleICS = "BEGIN:VCALENDAR\r\n" +
	"VERSION:2.0\r\n" +
	"PRODID:-//test//EN\r\n" +
	"BEGIN:VTIMEZONE\r\n" +
	"TZID:Europe/Berlin\r\n" +
	"END:VTIMEZONE\r\n" +
	"BEGIN:VEVENT\r\n" +
	"UID:1\r\n" +
	"DTSTART:20260115T090000Z\r\n" +
	"DTEND:20260115T093000Z\r\n" +
	"SUMMARY:Daily Standup\r\n" +
	"TRANSP:OPAQUE\r\n" +
	"BEGIN:VALARM\r\n" +
	"ACTION:DISPLAY\r\n" +
	"TRIGGER:-PT15M\r\n" +
	"END:VALARM\r\n" +
	"END:VEVENT\r\n" +
	"BEGIN:VEVENT\r\n" +
	"UID:2\r\n" +
	"DTSTART:20260115T140000Z\r\n" +
	"DTEND:20260115T150000Z\r\n" +
	"SUMMARY:Planning\r\n" +
	"TRANSP:TRANSPARENT\r\n" +
	"END:VEVENT\r\n" +
	"END:VCALENDAR\r\n"

func parseSample(t *testing.T) *ics.Calendar {
	t.Helper()
	cal, err := ics.ParseCalendar(strings.NewReader(sampleICS))
	if err != nil {
		t.Fatalf("parse sample: %v", err)
	}
	return cal
}

func TestTransformFiltersAndPreservesNonEvents(t *testing.T) {
	tr, err := Compile(&config.Calendar{
		ICalURL: "https://e.com/c.ics",
		Rules:   []config.Rule{{Field: "summary", Operator: "startswith", Val: config.Values{"Daily"}}},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	out := tr.Transform(parseSample(t))
	if got := len(out.Events()); got != 1 {
		t.Fatalf("want 1 event after filter, got %d", got)
	}
	if out.Events()[0].GetProperty(ics.ComponentPropertySummary).Value != "Daily Standup" {
		t.Fatal("wrong event survived the filter")
	}
	// VTIMEZONE must be preserved.
	if !strings.Contains(Serialize(out), "BEGIN:VTIMEZONE") {
		t.Fatal("VTIMEZONE not preserved")
	}
}

func TestTransformBlockingFilter(t *testing.T) {
	tr, _ := Compile(&config.Calendar{
		ICalURL: "https://e.com/c.ics",
		Rules:   []config.Rule{{Field: "blocking", Operator: "equals", Val: config.Values{"true"}}},
	})
	out := tr.Transform(parseSample(t))
	if got := len(out.Events()); got != 1 {
		t.Fatalf("want 1 blocking event, got %d", got)
	}
}

func TestTransformTimezoneAwareTimeFilter(t *testing.T) {
	// 09:00Z is 10:00 in Europe/Berlin (UTC+1 in January).
	tr, err := Compile(&config.Calendar{
		ICalURL:  "https://e.com/c.ics",
		Timezone: "Europe/Berlin",
		Rules:    []config.Rule{{Field: "start_time", Operator: "equals", Val: config.Values{"10:00"}}},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if got := len(tr.Transform(parseSample(t)).Events()); got != 1 {
		t.Fatalf("want 1 event matching 10:00 Berlin, got %d", got)
	}
}

func TestTransformAlarmsClearAndAdd(t *testing.T) {
	tr, err := Compile(&config.Calendar{
		ICalURL: "https://e.com/c.ics",
		Alarms:  &config.Alarms{ClearExisting: true, Triggers: []string{"2 days", "10 minutes"}},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	out := tr.Transform(parseSample(t))
	serialized := Serialize(out)
	// Original -PT15M alarm cleared.
	if strings.Contains(serialized, "-PT15M") {
		t.Fatal("existing alarm not cleared")
	}
	if !strings.Contains(serialized, "TRIGGER:-P2D") || !strings.Contains(serialized, "TRIGGER:-PT10M") {
		t.Fatalf("new alarm triggers missing:\n%s", serialized)
	}
	if !strings.Contains(serialized, "DESCRIPTION:Daily Standup") {
		t.Fatal("alarm description should be the event summary")
	}
}

func TestSerializeUsesCRLF(t *testing.T) {
	tr, _ := Compile(&config.Calendar{ICalURL: "https://e.com/c.ics"})
	if !strings.Contains(Serialize(tr.Transform(parseSample(t))), "\r\n") {
		t.Fatal("serialized output must use CRLF line endings")
	}
}

func TestParseTrigger(t *testing.T) {
	ok := map[string]string{
		"-P1DT0H0M0S": "-P1DT0H0M0S",
		"P2W":         "P2W",
		"2 days":      "-P2D",
		"5 hours":     "-PT5H",
		"10 minutes":  "-PT10M",
		"1 week":      "-P1W",
		"30 seconds":  "-PT30S",
	}
	for in, want := range ok {
		got, err := parseTrigger(in)
		if err != nil {
			t.Errorf("parseTrigger(%q) error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("parseTrigger(%q)=%q want %q", in, got, want)
		}
	}
	for _, bad := range []string{"lorem ipsum", "P", "PT", "5 days 10 minutes", "-10 days", ""} {
		if _, err := parseTrigger(bad); err == nil {
			t.Errorf("parseTrigger(%q) should fail", bad)
		}
	}
}

func TestCompileAlarmsNil(t *testing.T) {
	set, err := CompileAlarms(nil)
	if err != nil || set != nil {
		t.Fatalf("nil alarms should compile to nil set, got %v %v", set, err)
	}
}

func TestFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/calendar")
		_, _ = w.Write([]byte(sampleICS))
	}))
	defer srv.Close()

	cal, err := Fetch(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if got := len(cal.Events()); got != 2 {
		t.Fatalf("want 2 events, got %d", got)
	}
}

func TestFetchNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	if _, err := Fetch(context.Background(), srv.Client(), srv.URL); err == nil {
		t.Fatal("want error on non-200 upstream")
	}
}
