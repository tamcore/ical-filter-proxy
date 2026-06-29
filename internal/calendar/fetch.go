package calendar

import (
	"context"
	"fmt"
	"net/http"

	ics "github.com/arran4/golang-ical"
)

// Fetch retrieves and parses the upstream iCalendar feed at url. There is no
// caching: the feed is fetched fresh on every call, matching upstream behavior.
func Fetch(ctx context.Context, client *http.Client, url string) (*ics.Calendar, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %s: %w", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: upstream returned status %d", url, resp.StatusCode)
	}
	cal, err := ics.ParseCalendar(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", url, err)
	}
	return cal, nil
}
