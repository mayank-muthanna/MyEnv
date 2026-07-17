package ignore

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/myenv-cli/myenv/internal/diagnostic"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Paths []string `yaml:"paths"`
	Rules []string `yaml:"rules"`
	Env   []string `yaml:"env"`
}

func Load(filePath string) (Config, error) {
	contents, err := os.ReadFile(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var config Config
	if err := yaml.Unmarshal(contents, &config); err != nil {
		return Config{}, err
	}
	return config, nil
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
		if matchesAny(config.Rules, item.Rule) || matchesAny(config.Env, item.Key) {
			continue
		}
		if item.Path != "" && config.SkipPath(root, item.Path) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
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
