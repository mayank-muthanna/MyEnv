package validate

import (
	"regexp"
	"strings"
	"testing"

	"github.com/myenv-cli/myenv/internal/schema"
)

func TestValue(t *testing.T) {
	min, max := 1.0, 10.0
	rule := schema.Rule{Key: "PORT", Type: "int", Required: true, Range: &schema.Range{Min: &min, Max: &max}}
	if errors := Value(rule, "11", true); len(errors) != 1 {
		t.Fatalf("want range error, got %v", errors)
	}
	if errors := Value(rule, "8", true); len(errors) != 0 {
		t.Fatalf("want valid value, got %v", errors)
	}
	if errors := Value(rule, "", false); len(errors) != 1 {
		t.Fatalf("want required error, got %v", errors)
	}
}

func TestValueChecksPattern(t *testing.T) {
	rule := schema.Rule{Key: "KEY", Type: "string", Pattern: regexp.MustCompile("^sk_")}
	if errors := Value(rule, "pk_value", true); len(errors) != 1 {
		t.Fatalf("want pattern error, got %v", errors)
	}
}

func TestPatternHintExplainsMissingLiteralPrefix(t *testing.T) {
	rule := schema.Rule{Key: "BETTER_AUTH_URL", Pattern: regexp.MustCompile("^https://")}
	hint := PatternHint(rule, "http://localhost:3000")
	if !strings.Contains(hint, `start with "https://"`) {
		t.Fatalf("expected HTTPS prefix hint, got %q", hint)
	}
}

func TestPatternHintDoesNotExposeValue(t *testing.T) {
	rule := schema.Rule{Key: "SECRET", Pattern: regexp.MustCompile("^sk_")}
	hint := PatternHint(rule, "pk_private_value")
	if strings.Contains(hint, "pk_private_value") {
		t.Fatalf("hint exposed secret value: %q", hint)
	}
}

func TestPatternHintListsAllowedChoices(t *testing.T) {
	rule := schema.Rule{Key: "BETTER_AUTH_URL", Pattern: regexp.MustCompile(`^https://(hoqan.com|hoqan.info)`)}
	hint := PatternHint(rule, "https://hoqan.dom")
	for _, expected := range []string{`Allowed values:`, `"https://hoqan.com"`, `"https://hoqan.info"`} {
		if !strings.Contains(hint, expected) {
			t.Fatalf("expected %q in hint %q", expected, hint)
		}
	}
}

func TestPatternHintExplainsLengthAndAllowedCharacters(t *testing.T) {
	rule := schema.Rule{Key: "TOKEN", Pattern: regexp.MustCompile(`^[A-Za-z0-9]{24,48}$`)}
	hint := PatternHint(rule, "not-valid!")
	for _, expected := range []string{"at least 24 characters", "Allowed characters: letters, numbers"} {
		if !strings.Contains(hint, expected) {
			t.Fatalf("expected %q in hint %q", expected, hint)
		}
	}
}

func TestPatternHintExplainsMaximumLength(t *testing.T) {
	rule := schema.Rule{Key: "TOKEN", Pattern: regexp.MustCompile(`^[A-Za-z0-9]{2,4}$`)}
	hint := PatternHint(rule, "abcdef")
	if !strings.Contains(hint, "at most 4 characters") {
		t.Fatalf("expected maximum length hint, got %q", hint)
	}
}
