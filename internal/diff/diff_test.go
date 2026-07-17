package diff

import (
	"testing"

	"github.com/myenv-cli/myenv/internal/scanner"
	"github.com/myenv-cli/myenv/internal/schema"
)

func TestCompareChecksCodeDotenvAndSchema(t *testing.T) {
	rules := schema.Schema{
		"SECRET": {Key: "SECRET", Type: "string", Secret: true},
		"UNUSED": {Key: "UNUSED", Type: "string"},
	}
	values := map[string]string{
		"SECRET":   "value",
		"UNUSED":   "value",
		"ENV_ONLY": "value",
	}
	accesses := []scanner.Access{
		{Key: "SECRET", Path: "app.ts", Line: 2, ClientSide: true},
		{Key: "MISSING", Path: "app.ts", Line: 3},
	}

	diagnostics := Compare(rules, values, accesses)
	if len(diagnostics) != 3 {
		t.Fatalf("want three diagnostics, got %#v", diagnostics)
	}

	seen := map[string]bool{}
	for _, item := range diagnostics {
		seen[item.Rule] = true
	}
	for _, rule := range []string{"undeclared-code-env", "client-secret-exposure", "unused-config-env"} {
		if !seen[rule] {
			t.Errorf("missing %s diagnostic: %#v", rule, diagnostics)
		}
	}
}

func TestCompareReportsCodeKeyMissingFromDotenv(t *testing.T) {
	rules := schema.Schema{"PORT": {Key: "PORT", Type: "int"}}
	diagnostics := Compare(rules, map[string]string{}, []scanner.Access{{Key: "PORT", Path: "app.ts", Line: 1}})
	if len(diagnostics) != 1 || diagnostics[0].Rule != "undeclared-code-dotenv" {
		t.Fatalf("want dotenv error, got %#v", diagnostics)
	}
}
