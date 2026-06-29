package server

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamcore/ical-filter-proxy/internal/config"
)

const upstreamICS = "BEGIN:VCALENDAR\r\n" +
	"VERSION:2.0\r\n" +
	"PRODID:-//test//EN\r\n" +
	"BEGIN:VEVENT\r\nUID:1\r\nDTSTART:20260115T090000Z\r\nSUMMARY:Daily Standup\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:2\r\nDTSTART:20260115T140000Z\r\nSUMMARY:Planning\r\nEND:VEVENT\r\n" +
	"END:VCALENDAR\r\n"

// newTestServer builds a Server whose single calendar "work" points at a local
// upstream serving upstreamICS, filtered to summaries starting with "Daily".
func newTestServer(t *testing.T, apiKey string, upstreamURL string) *Server {
	t.Helper()
	cfg := config.Config{
		"work": &config.Calendar{
			ICalURL: upstreamURL,
			APIKey:  apiKey,
			Rules:   []config.Rule{{Field: "summary", Operator: "startswith", Val: config.Values{"Daily"}}},
		},
	}
	s, err := New(cfg, nil, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

func upstream(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(upstreamICS))
	}))
}

func TestWelcomeRoot(t *testing.T) {
	s := newTestServer(t, "", "https://e.com/c.ics")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Welcome") {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

func TestFilteredCalendarSuccess(t *testing.T) {
	up := upstream(t)
	defer up.Close()
	s := newTestServer(t, "secret", up.URL)

	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/work?key=secret", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/calendar") {
		t.Fatalf("Content-Type=%q want text/calendar", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Daily Standup") || strings.Contains(body, "Planning") {
		t.Fatalf("filter not applied, body:\n%s", body)
	}
}

func TestUnknownCalendar404(t *testing.T) {
	s := newTestServer(t, "", "https://e.com/c.ics")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rec.Code)
	}
}

func TestWrongKey403(t *testing.T) {
	s := newTestServer(t, "secret", "https://e.com/c.ics")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/work?key=wrong", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
}

func TestMissingKey403(t *testing.T) {
	s := newTestServer(t, "secret", "https://e.com/c.ics")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/work", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
}

func TestNoKeyConfiguredAllowsAccess(t *testing.T) {
	up := upstream(t)
	defer up.Close()
	s := newTestServer(t, "", up.URL)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/work", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", rec.Code)
	}
}

func TestUpstreamFailure5xx(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer up.Close()
	s := newTestServer(t, "", up.URL)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/work", nil))
	if rec.Code < 500 {
		t.Fatalf("status=%d want 5xx on upstream failure", rec.Code)
	}
}

func TestNewRejectsBadConfig(t *testing.T) {
	cfg := config.Config{"bad": &config.Calendar{
		ICalURL: "https://e.com/c.ics",
		Rules:   []config.Rule{{Field: "nope", Operator: "equals", Val: config.Values{"x"}}},
	}}
	if _, err := New(cfg, nil, nil); err == nil {
		t.Fatal("New should fail fast on an invalid rule")
	}
}

func TestNewLogsConfigSummary(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	cfg := config.Config{
		"work": &config.Calendar{
			ICalURL: "https://e.com/work.ics",
			APIKey:  "k",
			Rules:   []config.Rule{{Field: "summary", Operator: "equals", Val: config.Values{"x"}}},
			Alarms:  &config.Alarms{Triggers: []string{"10 minutes"}},
		},
		"public": &config.Calendar{ICalURL: "https://e.com/pub.ics"},
	}
	if _, err := New(cfg, nil, logger); err != nil {
		t.Fatalf("New: %v", err)
	}
	line := buf.String()
	for _, want := range []string{`"msg":"configuration loaded"`, `"calendars":2`, `"rules":1`, `"with_auth":1`, `"with_alarms":1`} {
		if !strings.Contains(line, want) {
			t.Fatalf("missing %s in startup log: %s", want, line)
		}
	}
}
