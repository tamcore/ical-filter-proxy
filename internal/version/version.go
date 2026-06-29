// Package version holds build-time version metadata injected via -ldflags.
package version

import "fmt"

// Populated at build time by goreleaser via -ldflags -X.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
	Branch    = "unknown"
)

// String returns a human-readable version line.
func String() string {
	return fmt.Sprintf("ical-filter-proxy %s (commit %s, branch %s, built %s)",
		Version, Commit, Branch, BuildDate)
}
