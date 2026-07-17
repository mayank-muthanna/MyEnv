package diff

import (
	"sort"

	"github.com/myenv-cli/myenv/internal/diagnostic"
	"github.com/myenv-cli/myenv/internal/scanner"
	"github.com/myenv-cli/myenv/internal/schema"
)

func Compare(rules schema.Schema, values map[string]string, accesses []scanner.Access) []diagnostic.Diagnostic {
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
		rule, inSchema := rules[key]
		_, inDotenv := values[key]
		switch {
		case !inSchema && !inDotenv:
			diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.Error, Rule: "undeclared-code-env", Message: key + " is used in code but absent from both .env and .myenv.yaml", Path: access.Path, Line: access.Line})
		case !inSchema:
			diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.Error, Rule: "undeclared-code-schema", Message: key + " is used in code but absent from .myenv.yaml", Path: access.Path, Line: access.Line})
		case !inDotenv:
			diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.Error, Rule: "undeclared-code-dotenv", Message: key + " is used in code but absent from .env", Path: access.Path, Line: access.Line})
		}
		if inSchema && rule.Secret && access.ClientSide {
			diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.Error, Rule: "client-secret-exposure", Message: key + " is secret and must not be referenced through import.meta.env", Path: access.Path, Line: access.Line})
		}
	}

	for _, key := range rules.Keys() {
		if _, exists := used[key]; !exists {
			message := key + " is declared in .myenv.yaml but not statically used in source"
			if _, exists := values[key]; exists {
				message = key + " exists in both .env and .myenv.yaml but is not statically used in source"
			}
			diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.Warning, Rule: "unused-config-env", Message: message})
		}
	}

	return diagnostics
}
