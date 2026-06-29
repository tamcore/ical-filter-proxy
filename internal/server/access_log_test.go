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

func TestAccessLogRecordsRequestWithoutKey(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	cfg := config.Config{"work": &config.Calendar{ICalURL: "https://e.com/c.ics", APIKey: "supersecret"}}
	s, err := New(cfg, nil, logger)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Wrong key -> 403, but the request must still be logged.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/work?key=wrong-guess", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")
	s.ServeHTTP(rec, req)

	line := buf.String()
	if !strings.Contains(line, `"msg":"request"`) {
		t.Fatalf("no access log line: %s", line)
	}
	if !strings.Contains(line, `"status":403`) {
		t.Fatalf("status not logged: %s", line)
	}
	if !strings.Contains(line, `"path":"/work"`) {
		t.Fatalf("path not logged: %s", line)
	}
	if !strings.Contains(line, `"client_ip":"203.0.113.7"`) {
		t.Fatalf("client_ip not taken from X-Forwarded-For: %s", line)
	}
	// The api key must never appear in logs.
	if strings.Contains(line, "supersecret") {
		t.Fatalf("api key leaked into access log: %s", line)
	}
}

func TestClientIPFallbacks(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.0.2.5:54321"
	if got := clientIP(r); got != "192.0.2.5" {
		t.Fatalf("RemoteAddr fallback: got %q", got)
	}
	r.Header.Set("X-Real-IP", "198.51.100.9")
	if got := clientIP(r); got != "198.51.100.9" {
		t.Fatalf("X-Real-IP: got %q", got)
	}
}
