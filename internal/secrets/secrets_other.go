//go:build !windows

package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	aesPrefix   = "enc:aes:"
	keyFileName = "secret.key"
	keySize     = 32
)

type aesProtector struct {
	key []byte
}

func newPlatformProtector(configDir string) (Protector, error) {
	key, err := loadOrCreateKey(configDir)
	if err != nil {
		return nil, err
	}

	return &aesProtector{key: key}, nil
}

func loadOrCreateKey(configDir string) ([]byte, error) {
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create directory for secret key: %w", err)
	}

	keyPath := filepath.Join(configDir, keyFileName)

	data, err := os.ReadFile(keyPath)
	if os.IsNotExist(err) {
		if err := createKeyFile(keyPath); err != nil {
			return nil, err
		}

		data, err = os.ReadFile(keyPath)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read secret key file: %w", err)
	}

	if len(data) != keySize {
		return nil, fmt.Errorf("secret key file %s has invalid length %d, expected %d bytes", keyPath, len(data), keySize)
	}

	// A key restored from a backup may come back world-readable.
	if info, err := os.Stat(keyPath); err == nil && info.Mode().Perm()&0077 != 0 {
		if err := os.Chmod(keyPath, 0600); err != nil {
			return nil, fmt.Errorf("failed to restrict secret key file permissions: %w", err)
		}
	}

	return data, nil
}

// createKeyFile writes a new key atomically and never replaces an existing
// one: secrets encrypted with a replaced key would be lost.
func createKeyFile(keyPath string) error {
	key := make([]byte, keySize)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("failed to generate secret key: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(keyPath), keyFileName+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary secret key file: %w", err)
	}

	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to restrict temporary secret key file: %w", err)
	}

	if _, err := tmp.Write(key); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write secret key file: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to sync secret key file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close secret key file: %w", err)
	}

	if err := os.Link(tmpPath, keyPath); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to install secret key file: %w", err)
	}

	return nil
}

func (p *aesProtector) Encrypt(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}

	gcm, err := p.gcm()
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plain), nil)

	return aesPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (p *aesProtector) Decrypt(value string) (string, error) {
	if !IsEncrypted(value) {
		return value, nil
	}

	if !strings.HasPrefix(value, aesPrefix) {
		return "", errOtherPlatform
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, aesPrefix))
	if err != nil {
		return "", fmt.Errorf("%w: %w", errCannotDecrypt, err)
	}

	gcm, err := p.gcm()
	if err != nil {
		return "", err
	}

	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("%w: ciphertext too short", errCannotDecrypt)
	}

	nonce, ciphertext := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]

	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errCannotDecrypt, err)
	}

	return string(plain), nil
}

func (p *aesProtector) gcm() (cipher.AEAD, error) {
	block, err := aes.NewCipher(p.key)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GCM: %w", err)
	}

	return gcm, nil
}
