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

type Document struct {
	Schema       Schema
	IgnoreCode   []string
	IgnoreUnused []string
	IgnorePaths  []string
	IgnoreRules  []string
}

func Load(filePath string) (Schema, error) {
	document, err := LoadDocument(filePath)
	if err != nil {
		return nil, err
	}
	return document.Schema, nil
}

func LoadDocument(filePath string) (Document, error) {
	contents, err := os.ReadFile(filePath)
	if err != nil {
		return Document{}, err
	}
	return ParseDocument(contents)
}

func Parse(contents []byte) (Schema, error) {
	document, err := ParseDocument(contents)
	if err != nil {
		return nil, err
	}
	return document.Schema, nil
}

func ParseDocument(contents []byte) (Document, error) {
	var nodes map[string]yaml.Node
	if err := yaml.Unmarshal(contents, &nodes); err != nil {
		return Document{}, fmt.Errorf("parse schema: %w", err)
	}

	rawRules := make(map[string]RawRule, len(nodes))
	document := Document{}
	for key, node := range nodes {
		switch key {
		case "ignoreCode":
			if err := node.Decode(&document.IgnoreCode); err != nil {
				return Document{}, fmt.Errorf("ignoreCode must be a list of environment names: %w", err)
			}
		case "ignoreUnused":
			if err := node.Decode(&document.IgnoreUnused); err != nil {
				return Document{}, fmt.Errorf("ignoreUnused must be a list of environment names: %w", err)
			}
		case "ignorePaths":
			if err := node.Decode(&document.IgnorePaths); err != nil {
				return Document{}, fmt.Errorf("ignorePaths must be a list of paths: %w", err)
			}
		case "ignoreRules":
			if err := node.Decode(&document.IgnoreRules); err != nil {
				return Document{}, fmt.Errorf("ignoreRules must be a list of diagnostic rules: %w", err)
			}
		default:
			var candidate RawRule
			if err := node.Decode(&candidate); err != nil {
				return Document{}, fmt.Errorf("%s: invalid rule: %w", key, err)
			}
			rawRules[key] = candidate
		}
	}

	rules, err := normalize(rawRules)
	if err != nil {
		return Document{}, err
	}
	document.Schema = rules
	return document, nil
}

func normalize(raw map[string]RawRule) (Schema, error) {
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
	return RenderDocument(Document{Schema: rules})
}

func RenderDocument(document Document) ([]byte, error) {
	nodes := make(map[string]any, len(document.Schema)+4)
	if len(document.IgnoreCode) > 0 {
		nodes["ignoreCode"] = document.IgnoreCode
	}
	if len(document.IgnoreUnused) > 0 {
		nodes["ignoreUnused"] = document.IgnoreUnused
	}
	if len(document.IgnorePaths) > 0 {
		nodes["ignorePaths"] = document.IgnorePaths
	}
	if len(document.IgnoreRules) > 0 {
		nodes["ignoreRules"] = document.IgnoreRules
	}
	for key, rule := range document.Schema {
		pattern := ""
		if rule.Pattern != nil {
			pattern = rule.Pattern.String()
		}
		nodes[key] = RawRule{Type: rule.Type, Required: rule.Required, Default: rule.Default, Pattern: pattern, Range: rule.Range, Secret: rule.Secret}
	}
	contents, err := yaml.Marshal(nodes)
	if err != nil {
		return nil, err
	}
	return append([]byte(inferIgnoreTemplate), contents...), nil
}

const inferIgnoreTemplate = `# Pattern guide (Go/RE2 regular expressions; pattern checks whole value only when you use ^ and $).
# pattern: '^sk_(live|test)_[A-Za-z0-9]{24,}$' # ^ start, $ end, (...) choose one, [A-Za-z0-9] allowed chars, {24,} 24+ chars.
# Examples: '^https://.+$' = URL starting https; '^[0-9]{4}$' = exactly four digits; '^(dev|staging|prod)$' = one allowed value.
# Escape regex symbols when literal: '\.' matches a dot; use single quotes so YAML keeps backslashes unchanged.
# Add pattern: '...' under any environment variable. Remove it when any string value is valid.
#
# Optional scan ignore policy. Uncomment only entries you need.
# ignorePaths:
#   - .nuxt/
# ignoreRules:
#   - dynamic-env-access
# ignoreCode:
#   - EXTERNAL_PROVIDER_SECRET
# ignoreUnused:
#   - DEPLOYMENT_ONLY_SETTING

`

func LooksSecretName(key string) bool {
	upper := strings.ToUpper(key)
	return strings.Contains(upper, "SECRET") || strings.Contains(upper, "TOKEN") || strings.Contains(upper, "PASSWORD") || strings.Contains(upper, "PRIVATE") || strings.HasSuffix(upper, "_KEY")
}
