package scanner

import (
	"os"
	"path/filepath"
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
