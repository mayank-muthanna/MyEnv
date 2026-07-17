package envcrypt

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/myenv-cli/myenv/internal/schema"
)

const (
	keyLength     = 32
	formatVersion = 1
	algorithm     = "AES-256-GCM"
	compression   = "gzip"
)

func GenerateKey() ([]byte, error) {
	key := make([]byte, keyLength)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate encryption key: %w", err)
	}
	return key, nil
}

func ParseKey(encoded string) ([]byte, error) {
	key, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("key must be a base64url-encoded 32-byte key")
	}
	if len(key) != keyLength {
		return nil, fmt.Errorf("key must decode to exactly 32 bytes")
	}
	return key, nil
}

func EncodeKey(key []byte) (string, error) {
	if len(key) != keyLength {
		return "", fmt.Errorf("key must be exactly 32 bytes")
	}
	return base64.RawURLEncoding.EncodeToString(key), nil
}

func Encrypt(plaintext, key []byte) (*schema.EncryptedEnv, error) {
	if len(key) != keyLength {
		return nil, fmt.Errorf("key must be exactly 32 bytes")
	}
	compressed, err := compress(plaintext)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create authenticated cipher: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, compressed, nil)
	return &schema.EncryptedEnv{
		Version:     formatVersion,
		Algorithm:   algorithm,
		Compression: compression,
		Nonce:       base64.RawURLEncoding.EncodeToString(nonce),
		Ciphertext:  base64.RawURLEncoding.EncodeToString(ciphertext),
	}, nil
}

func Decrypt(payload *schema.EncryptedEnv, key []byte) ([]byte, error) {
	if payload == nil {
		return nil, fmt.Errorf("no encryptedEnv payload found in schema")
	}
	if len(key) != keyLength {
		return nil, fmt.Errorf("key must be exactly 32 bytes")
	}
	if payload.Version != formatVersion || payload.Algorithm != algorithm || payload.Compression != compression {
		return nil, fmt.Errorf("unsupported encryptedEnv format")
	}
	nonce, err := base64.RawURLEncoding.DecodeString(payload.Nonce)
	if err != nil {
		return nil, fmt.Errorf("encryptedEnv nonce is invalid")
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(payload.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("encryptedEnv ciphertext is invalid")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create authenticated cipher: %w", err)
	}
	compressed, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt payload: key is wrong or encrypted data was changed")
	}
	return decompress(compressed)
}

func compress(plaintext []byte) ([]byte, error) {
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write(plaintext); err != nil {
		return nil, fmt.Errorf("compress dotenv: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finish compression: %w", err)
	}
	return buffer.Bytes(), nil
}

func decompress(compressed []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("decompress payload: %w", err)
	}
	defer reader.Close()
	plaintext, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read decompressed payload: %w", err)
	}
	return plaintext, nil
}
