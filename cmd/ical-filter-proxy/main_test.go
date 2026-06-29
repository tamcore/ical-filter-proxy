package main

import "testing"

func TestEnvOr(t *testing.T) {
	t.Setenv("ICAL_FILTER_PROXY_TEST_KEY", "set-value")
	if got := envOr("ICAL_FILTER_PROXY_TEST_KEY", "fallback"); got != "set-value" {
		t.Fatalf("got %q want set-value", got)
	}
	if got := envOr("ICAL_FILTER_PROXY_UNSET_KEY", "fallback"); got != "fallback" {
		t.Fatalf("got %q want fallback", got)
	}
}
