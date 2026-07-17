package diff

import (
	"testing"

	"github.com/myenv-cli/myenv/internal/scanner"
	"github.com/myenv-cli/myenv/internal/schema"
)

func TestCompareReportsUndeclaredUnusedAndClientSecret(t *testing.T) {
	rules := schema.Schema{"SECRET": {Key: "SECRET", Type: "string", Secret: true}, "UNUSED": {Key: "UNUSED", Type: "string"}}
	diagnostics := Compare(rules, []scanner.Access{{Key: "SECRET", Path: "app.ts", Line: 2, ClientSide: true}, {Key: "MISSING", Path: "app.ts", Line: 3}})
	if len(diagnostics) != 3 {
		t.Fatalf("want three diagnostics, got %#v", diagnostics)
	}
}
