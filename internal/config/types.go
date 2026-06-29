// Package config defines the YAML configuration schema for ical-filter-proxy
// and loads it with environment-variable substitution and startup validation.
//
// The schema mirrors the upstream Ruby darkphnx/ical-filter-proxy so existing
// config.yml files work unchanged.
package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Config maps a calendar name to its definition.
type Config map[string]*Calendar

// Calendar describes one proxied, filtered calendar feed.
type Calendar struct {
	ICalURL  string  `yaml:"ical_url"`
	APIKey   string  `yaml:"api_key"`
	Timezone string  `yaml:"timezone"`
	Rules    []Rule  `yaml:"rules"`
	Alarms   *Alarms `yaml:"alarms"`
}

// Rule is a single filter condition. An event is kept only when every rule of
// its calendar matches (logical AND).
type Rule struct {
	Field    string `yaml:"field"`
	Operator string `yaml:"operator"`
	Val      Values `yaml:"val"`
}

// Alarms controls VALARM manipulation on the filtered output.
type Alarms struct {
	ClearExisting bool     `yaml:"clear_existing"`
	Triggers      []string `yaml:"triggers"`
}

// Values normalizes a YAML scalar or sequence of scalars into a list of
// strings. A list is OR-ed within a rule. Booleans/numbers are stringified
// (e.g. blocking: true -> "true").
type Values []string

// UnmarshalYAML accepts either a scalar or a flat sequence of scalars.
func (v *Values) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		*v = Values{node.Value}
	case yaml.SequenceNode:
		out := make(Values, 0, len(node.Content))
		for _, c := range node.Content {
			if c.Kind != yaml.ScalarNode {
				return fmt.Errorf("rule val list entries must be scalars, got kind %d", c.Kind)
			}
			out = append(out, c.Value)
		}
		*v = out
	default:
		return fmt.Errorf("rule val must be a scalar or a list of scalars, got kind %d", node.Kind)
	}
	return nil
}
