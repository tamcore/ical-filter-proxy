package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// envPlaceholder matches ${ICAL_FILTER_PROXY_NAME} tokens. The full name inside
// the braces is the environment variable that supplies the replacement.
var envPlaceholder = regexp.MustCompile(`\$\{(ICAL_FILTER_PROXY_[A-Za-z0-9_]+)\}`)

// substituteEnv replaces every ${ICAL_FILTER_PROXY_NAME} token with the value of
// the matching environment variable (empty string when unset), mirroring the
// upstream behavior.
func substituteEnv(raw string) string {
	return envPlaceholder.ReplaceAllStringFunc(raw, func(match string) string {
		name := envPlaceholder.FindStringSubmatch(match)[1]
		return os.Getenv(name)
	})
}

// Load reads, env-substitutes, parses and validates the config at path.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	return Parse(raw)
}

// Parse env-substitutes, unmarshals and validates raw YAML bytes. Exposed
// separately from Load to keep the loader testable without touching disk.
func Parse(raw []byte) (Config, error) {
	var cfg Config
	if err := yaml.Unmarshal([]byte(substituteEnv(string(raw))), &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}
