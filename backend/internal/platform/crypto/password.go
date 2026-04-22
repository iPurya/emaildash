package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

type PasswordHasher struct{}

func NewPasswordHasher() PasswordHasher {
	return PasswordHasher{}
}

func (PasswordHasher) Hash(password string) (string, string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", "", fmt.Errorf("read salt: %w", err)
	}
	memory := uint32(64 * 1024)
	iterations := uint32(3)
	parallelism := uint8(2)
	keyLength := uint32(32)
	key := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLength)
	params := fmt.Sprintf("memory=%d,time=%d,parallelism=%d,salt=%s", memory, iterations, parallelism, base64.RawStdEncoding.EncodeToString(salt))
	return base64.RawStdEncoding.EncodeToString(key), params, nil
}

func (PasswordHasher) Verify(password, hash, params string) error {
	parsed, err := parseParams(params)
	if err != nil {
		return err
	}
	expected, err := base64.RawStdEncoding.DecodeString(hash)
	if err != nil {
		return fmt.Errorf("decode hash: %w", err)
	}
	actual := argon2.IDKey([]byte(password), parsed.salt, parsed.time, parsed.memory, parsed.parallelism, uint32(len(expected)))
	if subtle.ConstantTimeCompare(actual, expected) != 1 {
		return fmt.Errorf("invalid password")
	}
	return nil
}

type passwordParams struct {
	memory      uint32
	time        uint32
	parallelism uint8
	salt        []byte
}

func parseParams(params string) (passwordParams, error) {
	parsed := passwordParams{}
	parts := strings.Split(params, ",")
	if len(parts) != 4 {
		return parsed, fmt.Errorf("invalid password params")
	}
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return parsed, fmt.Errorf("invalid password params")
		}
		switch kv[0] {
		case "memory":
			value, err := strconv.ParseUint(kv[1], 10, 32)
			if err != nil {
				return parsed, fmt.Errorf("parse memory: %w", err)
			}
			parsed.memory = uint32(value)
		case "time":
			value, err := strconv.ParseUint(kv[1], 10, 32)
			if err != nil {
				return parsed, fmt.Errorf("parse time: %w", err)
			}
			parsed.time = uint32(value)
		case "parallelism":
			value, err := strconv.ParseUint(kv[1], 10, 8)
			if err != nil {
				return parsed, fmt.Errorf("parse parallelism: %w", err)
			}
			parsed.parallelism = uint8(value)
		case "salt":
			salt, err := base64.RawStdEncoding.DecodeString(kv[1])
			if err != nil {
				return parsed, fmt.Errorf("decode salt: %w", err)
			}
			parsed.salt = salt
		}
	}
	if parsed.memory == 0 || parsed.time == 0 || parsed.parallelism == 0 || len(parsed.salt) == 0 {
		return parsed, fmt.Errorf("invalid password params")
	}
	return parsed, nil
}
