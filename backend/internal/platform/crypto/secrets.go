package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

type SecretSealer struct {
	key []byte
}

func NewSecretSealer(masterKeyPath string) (*SecretSealer, error) {
	key, err := loadOrCreateKey(masterKeyPath)
	if err != nil {
		return nil, err
	}
	return &SecretSealer{key: key}, nil
}

func (s SecretSealer) Seal(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("read nonce: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

func (s SecretSealer) Open(ciphertext string) (string, error) {
	payload, err := base64.RawStdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode secret: %w", err)
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}
	if len(payload) < gcm.NonceSize() {
		return "", fmt.Errorf("invalid ciphertext")
	}
	nonce, body := payload[:gcm.NonceSize()], payload[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, body, nil)
	if err != nil {
		return "", fmt.Errorf("open secret: %w", err)
	}
	return string(plaintext), nil
}

func loadOrCreateKey(masterKeyPath string) ([]byte, error) {
	if data, err := os.ReadFile(masterKeyPath); err == nil {
		decoded, err := base64.RawStdEncoding.DecodeString(string(data))
		if err != nil {
			return nil, fmt.Errorf("decode master key: %w", err)
		}
		return decoded, nil
	}
	if err := os.MkdirAll(filepath.Dir(masterKeyPath), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir master key dir: %w", err)
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("read master key: %w", err)
	}
	encoded := base64.RawStdEncoding.EncodeToString(key)
	if err := os.WriteFile(masterKeyPath, []byte(encoded), 0o600); err != nil {
		return nil, fmt.Errorf("write master key: %w", err)
	}
	return key, nil
}
