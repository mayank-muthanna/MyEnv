package schema

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type RawRule struct {
	Type     string  `yaml:"type"`
	Required bool    `yaml:"required"`
	Default  *string `yaml:"default"`
	Pattern  string  `yaml:"pattern"`
	Range    *Range  `yaml:"range"`
	Secret   bool    `yaml:"secret"`
}

type Range struct {
	Min *float64 `yaml:"min"`
	Max *float64 `yaml:"max"`
}

type Rule struct {
	Key      string
	Type     string
	Required bool
	Default  *string
	Pattern  *regexp.Regexp
	Range    *Range
	Secret   bool
}

type Schema map[string]Rule

func Load(path string) (Schema, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(contents)
}

func Parse(contents []byte) (Schema, error) {
	var raw map[string]RawRule
	if err := yaml.Unmarshal(contents, &raw); err != nil {
		return nil, fmt.Errorf("parse schema: %w", err)
	}
	result := make(Schema, len(raw))
	for key, candidate := range raw {
		if !regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`).MatchString(key) {
			return nil, fmt.Errorf("invalid environment variable name %q", key)
		}
		if candidate.Type == "" {
			candidate.Type = "string"
		}
		if candidate.Type != "string" && candidate.Type != "int" && candidate.Type != "float" && candidate.Type != "bool" {
			return nil, fmt.Errorf("%s: unsupported type %q", key, candidate.Type)
		}
		if candidate.Range != nil && candidate.Type != "int" && candidate.Type != "float" {
			return nil, fmt.Errorf("%s: range requires int or float type", key)
		}
		if candidate.Range != nil && candidate.Range.Min != nil && candidate.Range.Max != nil && *candidate.Range.Min > *candidate.Range.Max {
			return nil, fmt.Errorf("%s: range min cannot exceed max", key)
		}
		var pattern *regexp.Regexp
		if candidate.Pattern != "" {
			compiledPattern, compileErr := regexp.Compile(candidate.Pattern)
			if compileErr != nil {
				return nil, fmt.Errorf("%s: invalid pattern: %w", key, compileErr)
			}
			pattern = compiledPattern
		}
		result[key] = Rule{Key: key, Type: candidate.Type, Required: candidate.Required, Default: candidate.Default, Pattern: pattern, Range: candidate.Range, Secret: candidate.Secret}
	}
	return result, nil
}

func (schema Schema) Keys() []string {
	keys := make([]string, 0, len(schema))
	for key := range schema {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func Render(rules Schema) ([]byte, error) {
	raw := make(map[string]RawRule, len(rules))
	for key, rule := range rules {
		pattern := ""
		if rule.Pattern != nil {
			pattern = rule.Pattern.String()
		}
		raw[key] = RawRule{Type: rule.Type, Required: rule.Required, Default: rule.Default, Pattern: pattern, Range: rule.Range, Secret: rule.Secret}
	}
	return yaml.Marshal(raw)
}

func LooksSecretName(key string) bool {
	upper := strings.ToUpper(key)
	return strings.Contains(upper, "SECRET") || strings.Contains(upper, "TOKEN") || strings.Contains(upper, "PASSWORD") || strings.Contains(upper, "PRIVATE") || strings.HasSuffix(upper, "_KEY")
}
