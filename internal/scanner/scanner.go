package scanner

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/myenv-cli/myenv/internal/diagnostic"
	"github.com/myenv-cli/myenv/internal/ignore"
)

type Access struct {
	Key        string
	Path       string
	Line       int
	ClientSide bool
}

var (
	processDot         = regexp.MustCompile(`\bprocess\.env\.([A-Z_][A-Z0-9_]*)\b`)
	processKey         = regexp.MustCompile("\\bprocess\\.env\\[\\s*['\"]([A-Z_][A-Z0-9_]*)['\"]\\s*\\]")
	importMeta         = regexp.MustCompile(`\bimport\.meta\.env\.([A-Z_][A-Z0-9_]*)\b`)
	dynamicEnv         = regexp.MustCompile(`\b(process\.env\s*\[|import\.meta\.env\s*\[)`)
	maxSourceLineBytes = 10 * 1024 * 1024
)

func Scan(root string, policy ignore.Config) ([]Access, []diagnostic.Diagnostic, error) {
	var sourcePaths []string
	err := filepath.WalkDir(root, func(filePath string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if policy.SkipPath(root, filePath) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			if isGitIgnored(root, filePath) {
				return filepath.SkipDir
			}
			switch entry.Name() {
			case ".git", "node_modules", "vendor", "dist", "build", ".next", "coverage":
				return filepath.SkipDir
			}
			return nil
		}
		extension := strings.ToLower(filepath.Ext(filePath))
		if extension == ".ts" || extension == ".tsx" || extension == ".js" || extension == ".jsx" || extension == ".mjs" || extension == ".cjs" {
			sourcePaths = append(sourcePaths, filePath)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	sourcePaths = filterGitIgnored(root, sourcePaths)
	var accesses []Access
	var diagnostics []diagnostic.Diagnostic
	for _, filePath := range sourcePaths {
		fileAccesses, fileDiagnostics, err := scanFile(filePath)
		if err != nil {
			return nil, nil, err
		}
		accesses = append(accesses, fileAccesses...)
		diagnostics = append(diagnostics, fileDiagnostics...)
	}
	return accesses, diagnostics, nil
}

func scanFile(filePath string) ([]Access, []diagnostic.Diagnostic, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	var accesses []Access
	var diagnostics []diagnostic.Diagnostic
	lineNumber := 0
	lines := bufio.NewScanner(file)
	lines.Buffer(make([]byte, 64*1024), maxSourceLineBytes)
	for lines.Scan() {
		lineNumber++
		line := lines.Text()
		add := func(matches [][]string, client bool) {
			for _, match := range matches {
				accesses = append(accesses, Access{Key: match[1], Path: filePath, Line: lineNumber, ClientSide: client})
			}
		}
		add(processDot.FindAllStringSubmatch(line, -1), false)
		bracketMatches := processKey.FindAllStringSubmatch(line, -1)
		add(bracketMatches, false)
		add(importMeta.FindAllStringSubmatch(line, -1), true)
		if dynamicEnv.MatchString(line) && len(bracketMatches) == 0 {
			diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.Warning, Rule: "dynamic-env-access", Message: "dynamic environment access cannot be statically verified", Path: filePath, Line: lineNumber})
		}
	}
	return accesses, diagnostics, lines.Err()
}

func isGitIgnored(root, filePath string) bool {
	relativePath, err := filepath.Rel(root, filePath)
	if err != nil || relativePath == "." {
		return false
	}
	command := exec.Command("git", "-C", root, "check-ignore", "-q", "--no-index", "--", filepath.ToSlash(relativePath))
	return command.Run() == nil
}
func filterGitIgnored(root string, sourcePaths []string) []string {
	filtered := make([]string, 0, len(sourcePaths))
	for _, filePath := range sourcePaths {
		if isGitIgnored(root, filePath) {
			continue
		}
		filtered = append(filtered, filePath)
	}
	return filtered
}
