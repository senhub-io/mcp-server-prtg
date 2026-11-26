package secrets

import (
	"errors"
	"strings"
)

const encryptedPrefix = "enc:"

var (
	errOtherPlatform = errors.New("value was encrypted on another platform and cannot be decrypted on this machine; the secret must be re-entered")
	errCannotDecrypt = errors.New("secret cannot be decrypted on this machine and must be re-entered")
)

// Protector encrypts and decrypts configuration secrets bound to this machine.
type Protector interface {
	Encrypt(plain string) (string, error)
	Decrypt(value string) (string, error)
}

// IsEncrypted reports whether value carries the "enc:" prefix.
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, encryptedPrefix)
}

// NewProtector returns the machine-bound protector for the given config directory.
func NewProtector(configDir string) (Protector, error) {
	return newPlatformProtector(configDir)
}
