package leaks

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/myenv-cli/myenv/internal/diagnostic"
)

type signature struct {
	name    string
	pattern *regexp.Regexp
}

var signatures = []signature{
	{"stripe-key", regexp.MustCompile(`\b(?:sk|rk)_(?:live|test)_[A-Za-z0-9]{16,}\b`)},
	{"aws-access-key", regexp.MustCompile(`\b(?:AKIA|ASIA)[A-Z0-9]{16}\b`)},
	{"slack-token", regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{10,}\b`)},
	{"private-key", regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`)},
}

func ScanTracked(root string) ([]diagnostic.Diagnostic, error) {
	command := exec.Command("git", "-C", root, "ls-files", "--", ".env*")
	output, err := command.Output()
	if err != nil {
		return nil, nil
	}
	var diagnostics []diagnostic.Diagnostic
	for _, relative := range strings.Fields(string(output)) {
		path := filepath.Join(root, filepath.FromSlash(relative))
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		scanner := bufio.NewScanner(file)
		line := 0
		for scanner.Scan() {
			line++
			for _, signature := range signatures {
				if signature.pattern.MatchString(scanner.Text()) {
					diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.Error, Rule: "likely-secret-" + signature.name, Message: "likely committed " + signature.name + "; replace it with a placeholder", Path: path, Line: line})
				}
			}
		}
		closeErr := file.Close()
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
	}
	return diagnostics, nil
}
