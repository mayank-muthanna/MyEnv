package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanFindsStaticReferences(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "app.ts"), []byte("process.env.FIRST\nprocess.env['SECOND']\nimport.meta.env.THIRD\nprocess.env[key]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	accesses, diagnostics, err := Scan(root)
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

	accesses, _, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(accesses) != 1 || accesses[0].Key != "LARGE_LINE" {
		t.Fatalf("unexpected accesses: %#v", accesses)
	}
}
