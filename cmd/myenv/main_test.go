package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/myenv-cli/myenv/internal/diagnostic"
	"github.com/myenv-cli/myenv/internal/envcrypt"
	"github.com/myenv-cli/myenv/internal/schema"
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

func TestCICommandChecksCodeAndEncryptedValuesInMemory(t *testing.T) {
	temporaryDirectory := t.TempDir()
	schemaPath := filepath.Join(temporaryDirectory, ".myenv.yaml")
	sourcePath := filepath.Join(temporaryDirectory, "app.ts")
	if err := os.WriteFile(sourcePath, []byte("process.env.PORT\n"), 0600); err != nil {
		t.Fatal(err)
	}

	key := []byte(strings.Repeat("a", 32))
	payload, err := envcrypt.Encrypt([]byte("PORT=3000\n"), key)
	if err != nil {
		t.Fatal(err)
	}
	document := schema.Document{Schema: schema.Schema{
		"PORT": {Key: "PORT", Type: "int", Required: true},
	}, EncryptedEnv: payload}
	contents, err := schema.RenderDocument(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(schemaPath, contents, 0600); err != nil {
		t.Fatal(err)
	}

	encodedKey, err := envcrypt.EncodeKey(key)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MYENV_DECRYPT_KEY", encodedKey)
	command := ciCommand()
	command.SetArgs([]string{"--schema", schemaPath, "--root", temporaryDirectory})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(temporaryDirectory, ".env.decrypted")); !os.IsNotExist(err) {
		t.Fatalf("CI command wrote plaintext dotenv file: %v", err)
	}

	payload, err = envcrypt.Encrypt([]byte("PORT=not-a-number\n"), key)
	if err != nil {
		t.Fatal(err)
	}
	document.EncryptedEnv = payload
	contents, err = schema.RenderDocument(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(schemaPath, contents, 0600); err != nil {
		t.Fatal(err)
	}
	invalidCommand := ciCommand()
	invalidCommand.SetArgs([]string{"--schema", schemaPath, "--root", temporaryDirectory})
	if err := invalidCommand.Execute(); !errors.Is(err, errChecksFailed) {
		t.Fatalf("expected encrypted dotenv validation failure, got %v", err)
	}
}

func TestReportJSONFailsWhenDiagnosticsContainErrors(t *testing.T) {
	diagnostics := []diagnostic.Diagnostic{{Severity: diagnostic.Error, Rule: "test-error", Message: "expected failure"}}
	if err := report("ci", diagnostics, "json"); !errors.Is(err, errChecksFailed) {
		t.Fatalf("expected JSON report failure, got %v", err)
	}
}
