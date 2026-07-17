package ignore

import (
	"path/filepath"
	"testing"

	"github.com/myenv-cli/myenv/internal/diagnostic"
)

func TestConfigFiltersPathsRulesAndEnv(t *testing.T) {
	root := t.TempDir()
	config := Config{Paths: []string{".nuxt/"}, Rules: []string{"dynamic-*"}, Env: []string{"NITRO_*"}}
	diagnostics := []diagnostic.Diagnostic{
		{Rule: "dynamic-env-access", Path: filepath.Join(root, "src", "app.ts")},
		{Rule: "undeclared-code-env", Key: "NITRO_ENV_PREFIX", Path: filepath.Join(root, "src", "app.ts")},
		{Rule: "undeclared-code-env", Key: "KEEP", Path: filepath.Join(root, ".nuxt", "dev", "index.mjs")},
		{Rule: "undeclared-code-env", Key: "KEEP", Path: filepath.Join(root, "src", "app.ts")},
	}

	filtered := config.Filter(root, diagnostics)
	if len(filtered) != 1 || filtered[0].Key != "KEEP" || filtered[0].Path != filepath.Join(root, "src", "app.ts") {
		t.Fatalf("unexpected diagnostics: %#v", filtered)
	}
}
