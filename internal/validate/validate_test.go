package validate

import (
	"regexp"
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
