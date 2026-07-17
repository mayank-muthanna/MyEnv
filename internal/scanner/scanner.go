package scanner

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/myenv-cli/myenv/internal/diagnostic"
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

func Scan(root string) ([]Access, []diagnostic.Diagnostic, error) {
	var accesses []Access
	var diagnostics []diagnostic.Diagnostic
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", "vendor", "dist", "build", ".next", "coverage":
				return filepath.SkipDir
			}
			return nil
		}
		extension := strings.ToLower(filepath.Ext(path))
		if extension != ".ts" && extension != ".tsx" && extension != ".js" && extension != ".jsx" && extension != ".mjs" && extension != ".cjs" {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		lineNumber := 0
		lines := bufio.NewScanner(file)
		lines.Buffer(make([]byte, 64*1024), maxSourceLineBytes)
		for lines.Scan() {
			lineNumber++
			line := lines.Text()
			add := func(matches [][]string, client bool) {
				for _, match := range matches {
					accesses = append(accesses, Access{Key: match[1], Path: path, Line: lineNumber, ClientSide: client})
				}
			}
			add(processDot.FindAllStringSubmatch(line, -1), false)
			bracketMatches := processKey.FindAllStringSubmatch(line, -1)
			add(bracketMatches, false)
			add(importMeta.FindAllStringSubmatch(line, -1), true)
			if dynamicEnv.MatchString(line) && len(bracketMatches) == 0 {
				diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.Warning, Rule: "dynamic-env-access", Message: "dynamic environment access cannot be statically verified", Path: path, Line: lineNumber})
			}
		}
		return lines.Err()
	})
	return accesses, diagnostics, err
}
