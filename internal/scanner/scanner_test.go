package scanner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/myenv-cli/myenv/internal/ignore"
)

func TestScanFindsStaticReferences(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "app.ts"), []byte("process.env.FIRST\nprocess.env['SECOND']\nimport.meta.env.THIRD\nprocess.env[key]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	accesses, diagnostics, err := Scan(root, ignore.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(accesses) != 3 || !accesses[2].ClientSide {
		t.Fatalf("unexpected accesses: %#v", accesses)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("want dynamic warning, got %#v", diagnostics)
	}
}

func TestScanHandlesLongSourceLines(t *testing.T) {
	root := t.TempDir()
	contents := strings.Repeat("x", 128*1024) + " process.env.LARGE_LINE\n"
	if err := os.WriteFile(filepath.Join(root, "bundle.js"), []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}

	accesses, _, err := Scan(root, ignore.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(accesses) != 1 || accesses[0].Key != "LARGE_LINE" {
		t.Fatalf("unexpected accesses: %#v", accesses)
	}
}

func TestScanSkipsCustomIgnoredPath(t *testing.T) {
	root := t.TempDir()
	ignoredPath := filepath.Join(root, ".nuxt", "dev")
	if err := os.MkdirAll(ignoredPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ignoredPath, "index.mjs"), []byte("process.env.IGNORED\n"), 0644); err != nil {
		t.Fatal(err)
	}

	accesses, diagnostics, err := Scan(root, ignore.Config{Paths: []string{".nuxt/"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(accesses) != 0 || len(diagnostics) != 0 {
		t.Fatalf("ignored path was scanned: accesses=%#v diagnostics=%#v", accesses, diagnostics)
	}
}

func TestScanSkipsGitIgnoredPath(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	root := t.TempDir()
	if output, err := exec.Command("git", "init", "--quiet", root).CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v: %s", err, output)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("generated/\n"), 0644); err != nil {
		t.Fatal(err)
	}
	generatedPath := filepath.Join(root, "generated")
	if err := os.MkdirAll(generatedPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(generatedPath, "bundle.js"), []byte("process.env.IGNORED\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app.js"), []byte("process.env.KEPT\n"), 0644); err != nil {
		t.Fatal(err)
	}

	accesses, _, err := Scan(root, ignore.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(accesses) != 1 || accesses[0].Key != "KEPT" {
		t.Fatalf("unexpected accesses: %#v", accesses)
	}
}
