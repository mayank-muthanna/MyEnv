package validate

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

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
			hint := ""
			if message == "does not match required pattern" {
				hint = PatternHint(rule, raw)
			}
			diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.Error, Rule: "invalid-value", Key: key, Message: key + " " + message, Hint: hint})
		}
	}
	for key := range values {
		if _, ok := rules[key]; !ok {
			diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.Error, Rule: "undeclared-dotenv", Key: key, Message: key + " exists in .env but is absent from .myenv.yaml"})
		}
	}
	return diagnostics
}

// PatternHint explains recognized regex constraints without exposing env values.
func PatternHint(rule schema.Rule, raw string) string {
	pattern := rule.Pattern.String()
	if choices, ok := allowedChoices(pattern); ok {
		return "Allowed values: " + quoteChoices(choices) + "."
	}
	if hint, ok := characterRuleHint(pattern, raw); ok {
		return hint
	}
	prefix := literalAnchoredPrefix(pattern)
	if prefix != "" && !strings.HasPrefix(raw, prefix) {
		return fmt.Sprintf("Value must start with %q. Update %s or change its pattern if this value is intentional.", prefix, rule.Key)
	}
	return "Pattern is custom. Check required prefix, allowed characters, and length."
}

var (
	choicePattern        = regexp.MustCompile(`^\^(.+)\(([^()|]+(?:\|[^()|]+)+)\)\$?$`)
	characterRulePattern = regexp.MustCompile(`^\^([^\[\]{}()*+?|^$\\]*)\[([^\]]+)\]\{([0-9]+)(?:,([0-9]*))?\}\$?$`)
)

func allowedChoices(pattern string) ([]string, bool) {
	match := choicePattern.FindStringSubmatch(pattern)
	if len(match) == 0 || !strings.Contains(match[2], "|") {
		return nil, false
	}
	choices := strings.Split(match[2], "|")
	for index, choice := range choices {
		if choice == "" || strings.ContainsAny(choice, `\\^$*+?[]{}()|`) {
			return nil, false
		}
		choices[index] = match[1] + choice
	}
	return choices, true
}

func characterRuleHint(pattern, raw string) (string, bool) {
	match := characterRulePattern.FindStringSubmatch(pattern)
	if len(match) == 0 {
		return "", false
	}
	prefix, class, minimumText, maximumText := match[1], match[2], match[3], match[4]
	if prefix != "" && !strings.HasPrefix(raw, prefix) {
		return fmt.Sprintf("Value must start with %q.", prefix), true
	}
	value := strings.TrimPrefix(raw, prefix)
	minimum, _ := strconv.Atoi(minimumText)
	maximum := -1
	if maximumText != "" {
		maximum, _ = strconv.Atoi(maximumText)
	}
	var problems []string
	length := len([]rune(value))
	if length < minimum {
		problems = append(problems, fmt.Sprintf("Must contain at least %d characters", minimum))
	}
	if maximum >= 0 && length > maximum {
		problems = append(problems, fmt.Sprintf("Must contain at most %d characters", maximum))
	}
	if !regexp.MustCompile("^[" + class + "]+$").MatchString(value) {
		problems = append(problems, "Allowed characters: "+describeCharacterClass(class))
	}
	if len(problems) == 0 {
		return "Value does not meet its required character format.", true
	}
	return strings.Join(problems, ". ") + ".", true
}

func describeCharacterClass(class string) string {
	var descriptions []string
	if strings.Contains(class, "A-Z") || strings.Contains(class, "a-z") {
		descriptions = append(descriptions, "letters")
	}
	if strings.Contains(class, "0-9") {
		descriptions = append(descriptions, "numbers")
	}
	if strings.Contains(class, "_") {
		descriptions = append(descriptions, "underscores")
	}
	if strings.Contains(class, "-") {
		descriptions = append(descriptions, "hyphens")
	}
	if len(descriptions) == 0 {
		return "characters from the configured allowed set"
	}
	return strings.Join(descriptions, ", ")
}

func quoteChoices(choices []string) string {
	quoted := make([]string, len(choices))
	for index, choice := range choices {
		quoted[index] = fmt.Sprintf("%q", choice)
	}
	return strings.Join(quoted, ", ")
}

func literalAnchoredPrefix(pattern string) string {
	if !strings.HasPrefix(pattern, "^") {
		return ""
	}
	var prefix strings.Builder
	for _, character := range pattern[1:] {
		if strings.ContainsRune(`\\.^$*+?([{)|`, character) {
			break
		}
		prefix.WriteRune(character)
	}
	return prefix.String()
}
