package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEncryptDecryptCommandsRoundTrip(t *testing.T) {
	temporaryDirectory := t.TempDir()
	envPath := filepath.Join(temporaryDirectory, ".env.local")
	schemaPath := filepath.Join(temporaryDirectory, ".myenv.yaml")
	outputPath := filepath.Join(temporaryDirectory, "restored.env")
	original := []byte("# retain comments and spaces\nPORT=3000\nTOKEN=value with spaces\n")
	if err := os.WriteFile(envPath, original, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(schemaPath, []byte("PORT:\n  type: int\nTOKEN:\n  type: string\n  secret: true\n"), 0600); err != nil {
		t.Fatal(err)
	}

	key := strings.Repeat("A", 43)
	encrypt := encryptCommand()
	encrypt.SetArgs([]string{"--env", envPath, "--schema", schemaPath, "--key", key})
	encrypt.SetOut(&bytes.Buffer{})
	if err := encrypt.Execute(); err != nil {
		t.Fatal(err)
	}

	decrypt := decryptCommand()
	decrypt.SetArgs([]string{"--schema", schemaPath, "--key", key, "--output", outputPath})
	decrypt.SetOut(&bytes.Buffer{})
	if err := decrypt.Execute(); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(restored, original) {
		t.Fatalf("round trip changed dotenv bytes: got %q", restored)
	}
}
