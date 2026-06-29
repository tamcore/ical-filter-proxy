// Command ical-filter-proxy is a drop-in Go replacement for the Ruby
// darkphnx/ical-filter-proxy: it serves remote iCalendar feeds filtered by
// per-calendar rules from a YAML config.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Embed the IANA timezone database so timezone-aware filtering works in a
	// distroless image that ships no system tzdata.
	_ "time/tzdata"

	"github.com/tamcore/ical-filter-proxy/internal/config"
	"github.com/tamcore/ical-filter-proxy/internal/server"
	"github.com/tamcore/ical-filter-proxy/internal/version"
)

const (
	defaultConfigPath   = "/app/config.yml"
	defaultAddr         = ":8000"
	upstreamTimeout     = 30 * time.Second
	readHeaderTimeout   = 10 * time.Second
	shutdownGracePeriod = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", envOr("ICAL_FILTER_PROXY_CONFIG", defaultConfigPath), "path to config.yml")
	addr := flag.String("addr", envOr("ICAL_FILTER_PROXY_ADDR", defaultAddr), "listen address")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if *showVersion {
		fmt.Println(version.String())
		return nil
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	srv, err := server.New(cfg, &http.Client{Timeout: upstreamTimeout}, logger)
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           srv,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", *addr, "version", version.Version)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	}
}

// envOr returns the environment variable value for key, or def when unset/empty.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
