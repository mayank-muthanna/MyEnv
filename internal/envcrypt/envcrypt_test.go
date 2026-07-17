package envcrypt

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	original := []byte("# preserve comments and order\nPORT=3000\nSECRET=hello world\n")
	key, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	payload, err := Encrypt(original, key)
	if err != nil {
		t.Fatal(err)
	}
	if payload.Ciphertext == "" || payload.Nonce == "" {
		t.Fatal("expected encrypted payload")
	}
	decrypted, err := Decrypt(payload, key)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, original) {
		t.Fatalf("round trip mismatch: got %q", decrypted)
	}
}

func TestDecryptRejectsWrongKey(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	payload, err := Encrypt([]byte("SECRET=value\n"), key)
	if err != nil {
		t.Fatal(err)
	}
	wrongKey, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decrypt(payload, wrongKey); err == nil {
		t.Fatal("expected wrong key to fail")
	}
}

func TestParseKey(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeKey(key)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := ParseKey(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded, key) {
		t.Fatal("key did not round trip")
	}
}
