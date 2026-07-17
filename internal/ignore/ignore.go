package ignore

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/myenv-cli/myenv/internal/diagnostic"
)

type Config struct {
	Code   []string
	Unused []string
	Paths  []string
	Rules  []string
}

func (config Config) SkipPath(root, filePath string) bool {
	relativePath, err := filepath.Rel(root, filePath)
	if err != nil {
		return false
	}
	return matchesAny(config.Paths, normalize(relativePath))
}

func (config Config) Filter(root string, diagnostics []diagnostic.Diagnostic) []diagnostic.Diagnostic {
	filtered := make([]diagnostic.Diagnostic, 0, len(diagnostics))
	for _, item := range diagnostics {
		if matchesAny(config.Rules, item.Rule) || isIgnoredCode(config, item) || isIgnoredUnused(config, item) {
			continue
		}
		if item.Path != "" && config.SkipPath(root, item.Path) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func isIgnoredCode(config Config, item diagnostic.Diagnostic) bool {
	if !matchesAny(config.Code, item.Key) {
		return false
	}
	return strings.HasPrefix(item.Rule, "undeclared-code-") || item.Rule == "client-secret-exposure"
}

func isIgnoredUnused(config Config, item diagnostic.Diagnostic) bool {
	return item.Rule == "unused-config-env" && matchesAny(config.Unused, item.Key)
}

func matchesAny(patterns []string, value string) bool {
	for _, pattern := range patterns {
		if matches(normalize(pattern), value) {
			return true
		}
	}
	return false
}

func matches(pattern, value string) bool {
	pattern = strings.TrimPrefix(pattern, "./")
	pattern = strings.TrimSpace(pattern)
	if pattern == "" || value == "" {
		return false
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return value == prefix || strings.HasPrefix(value, prefix+"/")
	}
	if strings.HasSuffix(pattern, "/") {
		prefix := strings.TrimSuffix(pattern, "/")
		return value == prefix || strings.HasPrefix(value, prefix+"/")
	}
	if !strings.ContainsAny(pattern, "*?[") && (value == pattern || strings.HasPrefix(value, pattern+"/")) {
		return true
	}
	matched, err := path.Match(pattern, value)
	return err == nil && matched
}

func normalize(value string) string {
	return strings.TrimPrefix(filepath.ToSlash(filepath.Clean(value)), "./")
}
