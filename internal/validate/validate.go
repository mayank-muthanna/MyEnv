package validate

import (
	"fmt"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/myenv-cli/myenv/internal/diagnostic"
	"github.com/myenv-cli/myenv/internal/schema"
)

func LoadDotenv(path string) (map[string]string, error) { return godotenv.Read(path) }

func Value(rule schema.Rule, raw string, present bool) []string {
	if !present {
		if rule.Default != nil {
			raw, present = *rule.Default, true
		} else if rule.Required {
			return []string{"is required but missing"}
		} else {
			return nil
		}
	}
	var errors []string
	var numeric float64
	switch rule.Type {
	case "int":
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			errors = append(errors, "must be an integer")
		} else {
			numeric = float64(value)
		}
	case "float":
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			errors = append(errors, "must be a number")
		} else {
			numeric = value
		}
	case "bool":
		if _, err := strconv.ParseBool(raw); err != nil {
			errors = append(errors, "must be a boolean")
		}
	}
	if rule.Pattern != nil && !rule.Pattern.MatchString(raw) {
		errors = append(errors, "does not match required pattern")
	}
	if rule.Range != nil && (rule.Type == "int" || rule.Type == "float") && len(errors) == 0 {
		if rule.Range.Min != nil && numeric < *rule.Range.Min {
			errors = append(errors, fmt.Sprintf("must be at least %v", *rule.Range.Min))
		}
		if rule.Range.Max != nil && numeric > *rule.Range.Max {
			errors = append(errors, fmt.Sprintf("must be at most %v", *rule.Range.Max))
		}
	}
	return errors
}

func Env(rules schema.Schema, values map[string]string) []diagnostic.Diagnostic {
	var diagnostics []diagnostic.Diagnostic
	for _, key := range rules.Keys() {
		rule := rules[key]
		raw, present := values[key]
		for _, message := range Value(rule, raw, present) {
			diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.Error, Rule: "invalid-value", Key: key, Message: key + " " + message})
		}
	}
	for key := range values {
		if _, ok := rules[key]; !ok {
			diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.Error, Rule: "undeclared-dotenv", Key: key, Message: key + " exists in .env but is absent from .myenv.yaml"})
		}
	}
	return diagnostics
}
