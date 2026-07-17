package diff

import (
	"sort"

	"github.com/myenv-cli/myenv/internal/diagnostic"
	"github.com/myenv-cli/myenv/internal/scanner"
	"github.com/myenv-cli/myenv/internal/schema"
)

func Compare(rules schema.Schema, accesses []scanner.Access) []diagnostic.Diagnostic {
	used := map[string]scanner.Access{}
	for _, access := range accesses {
		if _, exists := used[access.Key]; !exists {
			used[access.Key] = access
		}
	}
	keys := make([]string, 0, len(used))
	for key := range used {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var diagnostics []diagnostic.Diagnostic
	for _, key := range keys {
		access := used[key]
		rule, declared := rules[key]
		if !declared {
			diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.Error, Rule: "undeclared-code-env", Message: key + " is used in code but absent from .myenv.yaml; add it or run myenv infer", Path: access.Path, Line: access.Line})
			continue
		}
		if rule.Secret && access.ClientSide {
			diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.Error, Rule: "client-secret-exposure", Message: key + " is secret and must not be referenced through import.meta.env", Path: access.Path, Line: access.Line})
		}
	}
	for _, key := range rules.Keys() {
		if _, exists := used[key]; !exists {
			diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.Warning, Rule: "unused-schema-env", Message: key + " is declared in .myenv.yaml but not statically used in source"})
		}
	}
	return diagnostics
}
