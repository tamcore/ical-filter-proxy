// Package server wires the HTTP surface of ical-filter-proxy: a welcome route
// and one filtered-calendar route per configured calendar.
package server

import (
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/tamcore/ical-filter-proxy/internal/calendar"
	"github.com/tamcore/ical-filter-proxy/internal/config"
)

const welcomeMessage = "Welcome to ical-filter-proxy"

// entry is the per-calendar runtime state, compiled once at startup.
type entry struct {
	apiKey      string
	icalURL     string
	transformer *calendar.Transformer
}

// Server serves filtered calendars over HTTP. It implements http.Handler.
type Server struct {
	calendars map[string]*entry
	client    *http.Client
	logger    *slog.Logger
}

// New compiles every calendar's filters/alarms (failing fast on bad config) and
// returns a ready handler.
func New(cfg config.Config, client *http.Client, logger *slog.Logger) (*Server, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{calendars: make(map[string]*entry, len(cfg)), client: client, logger: logger}
	var totalRules, withAuth, withAlarms int
	for name, c := range cfg {
		tr, err := calendar.Compile(c)
		if err != nil {
			return nil, fmt.Errorf("calendar %q: %w", name, err)
		}
		s.calendars[name] = &entry{apiKey: c.APIKey, icalURL: c.ICalURL, transformer: tr}
		totalRules += len(c.Rules)
		if c.APIKey != "" {
			withAuth++
		}
		if c.Alarms != nil {
			withAlarms++
		}
	}
	logger.Info("configuration loaded",
		"calendars", len(cfg),
		"rules", totalRules,
		"with_auth", withAuth,
		"with_alarms", withAlarms,
	)
	return s, nil
}

// route handles requests: "/" returns a welcome string, "/<name>" returns the
// authenticated, filtered calendar. ServeHTTP wraps it with access logging.
func (s *Server) route(w http.ResponseWriter, r *http.Request) {
	name := strings.Trim(r.URL.Path, "/")
	if name == "" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(welcomeMessage))
		return
	}

	e, ok := s.calendars[name]
	if !ok {
		http.Error(w, "Calendar not found", http.StatusNotFound)
		return
	}
	if !authorized(e.apiKey, r.URL.Query().Get("key")) {
		http.Error(w, "Authentication Incorrect", http.StatusForbidden)
		return
	}

	src, err := calendar.Fetch(r.Context(), s.client, e.icalURL)
	if err != nil {
		s.logger.Error("upstream fetch failed", "calendar", name, "error", err)
		http.Error(w, "Upstream calendar unavailable", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	_, _ = w.Write([]byte(calendar.Serialize(e.transformer.Transform(src))))
}

// authorized reports whether the supplied key is accepted. An empty configured
// key disables authentication; otherwise a constant-time comparison is used.
func authorized(configured, supplied string) bool {
	if configured == "" {
		return true
	}
	return subtle.ConstantTimeCompare([]byte(configured), []byte(supplied)) == 1
}
