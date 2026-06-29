package filter

import (
	"fmt"
	"regexp"
	"strings"
)

// translateRegex converts a Ruby-style /pattern/flags literal into a compiled
// Go (RE2) regexp.
//
// Flag mapping (Ruby -> Go inline flags):
//   - i (ignore case)      -> i
//   - m (dot matches \n)   -> s   (Ruby /m is dotall; ^$ are always line anchors)
//
// The Ruby x (extended/free-spacing) flag has no RE2 equivalent and is rejected
// so misconfiguration fails loudly at startup rather than silently misbehaving.
func translateRegex(pattern string) (*regexp.Regexp, error) {
	if len(pattern) < 2 || pattern[0] != '/' {
		return nil, fmt.Errorf("regex %q must be of the form /pattern/flags", pattern)
	}
	close := strings.LastIndex(pattern, "/")
	if close == 0 {
		return nil, fmt.Errorf("regex %q is missing its closing slash", pattern)
	}
	body := pattern[1:close]
	flags := pattern[close+1:]

	var inline strings.Builder
	for _, f := range flags {
		switch f {
		case 'i':
			inline.WriteByte('i')
		case 'm':
			inline.WriteByte('s')
		default:
			return nil, fmt.Errorf("regex %q uses unsupported flag %q", pattern, string(f))
		}
	}

	expr := body
	if inline.Len() > 0 {
		expr = "(?" + inline.String() + ")" + body
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("regex %q: %w", pattern, err)
	}
	return re, nil
}
