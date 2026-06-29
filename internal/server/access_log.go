package server

import (
	"net"
	"net/http"
	"strings"
	"time"
)

// statusRecorder captures the response status code and byte count for logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// ServeHTTP wraps routing with a structured access log line per request. The
// api key is never logged: only the path (without query string) and the
// resolved calendar name are recorded.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rec := &statusRecorder{ResponseWriter: w}

	s.route(rec, r)

	if rec.status == 0 {
		rec.status = http.StatusOK
	}
	s.logger.Info("request",
		"method", r.Method,
		"path", r.URL.Path,
		"status", rec.status,
		"bytes", rec.bytes,
		"duration_ms", time.Since(start).Milliseconds(),
		"client_ip", clientIP(r),
		"user_agent", r.UserAgent(),
	)
}

// clientIP returns the originating client address, preferring the leftmost
// X-Forwarded-For entry set by the ingress, then X-Real-IP, then RemoteAddr.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		first, _, _ := strings.Cut(xff, ",")
		return strings.TrimSpace(first)
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return strings.TrimSpace(xr)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
