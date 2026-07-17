package infer

import (
	"regexp"
	"testing"

	"github.com/myenv-cli/myenv/internal/schema"
)

func TestMergePreservesConfiguredRulesAndSynchronizesKeys(t *testing.T) {
	pattern := regexp.MustCompile(`^https://`)
	existing := schema.Schema{
		"KEEP":   {Key: "KEEP", Type: "string", Required: true, Pattern: pattern},
		"REMOVE": {Key: "REMOVE", Type: "bool"},
	}
	inferred := schema.Schema{
		"KEEP": {Key: "KEEP", Type: "int"},
		"ADD":  {Key: "ADD", Type: "string"},
	}

	merged, change := Merge(existing, inferred)
	if len(merged) != 2 || merged["KEEP"].Pattern == nil || merged["KEEP"].Type != "string" {
		t.Fatalf("configured KEEP rule was not preserved: %#v", merged["KEEP"])
	}
	if _, found := merged["REMOVE"]; found {
		t.Fatal("removed dotenv key remained in merged schema")
	}
	if change != (Change{Added: 1, Removed: 1, Preserved: 1}) {
		t.Fatalf("unexpected change summary: %#v", change)
	}
}
