//go:build !windows

package secrets

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAESRoundTrip(t *testing.T) {
	p, err := NewProtector(t.TempDir())
	require.NoError(t, err)

	enc, err := p.Encrypt("s3cret-value")
	require.NoError(t, err)
	assert.True(t, IsEncrypted(enc))
	assert.NotContains(t, enc, "s3cret-value")

	dec, err := p.Decrypt(enc)
	require.NoError(t, err)
	assert.Equal(t, "s3cret-value", dec)
}

func TestKeyFileCreatedWith0600(t *testing.T) {
	dir := t.TempDir()

	_, err := NewProtector(dir)
	require.NoError(t, err)

	keyPath := filepath.Join(dir, keyFileName)
	info, err := os.Stat(keyPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	key, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Len(t, key, keySize)
}

func TestReuseExistingKeyFile(t *testing.T) {
	dir := t.TempDir()

	p1, err := NewProtector(dir)
	require.NoError(t, err)

	enc, err := p1.Encrypt("shared-secret")
	require.NoError(t, err)

	p2, err := NewProtector(dir)
	require.NoError(t, err)

	dec, err := p2.Decrypt(enc)
	require.NoError(t, err)
	assert.Equal(t, "shared-secret", dec)
}

func TestWrongLengthKeyFileRejected(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, keyFileName), []byte("too-short"), 0600))

	_, err := NewProtector(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid length")
}

func TestTamperedCiphertextRejected(t *testing.T) {
	dir := t.TempDir()

	p, err := NewProtector(dir)
	require.NoError(t, err)

	enc, err := p.Encrypt("s3cret-value")
	require.NoError(t, err)

	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(enc, aesPrefix))
	require.NoError(t, err)
	raw[len(raw)-1] ^= 0xff

	tampered := aesPrefix + base64.StdEncoding.EncodeToString(raw)

	_, err = p.Decrypt(tampered)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be decrypted")
}

func TestPlainValueReturnedUnchanged(t *testing.T) {
	p, err := NewProtector(t.TempDir())
	require.NoError(t, err)

	dec, err := p.Decrypt("plain-value")
	require.NoError(t, err)
	assert.Equal(t, "plain-value", dec)
}

func TestEmptyValueStaysEmpty(t *testing.T) {
	p, err := NewProtector(t.TempDir())
	require.NoError(t, err)

	enc, err := p.Encrypt("")
	require.NoError(t, err)
	assert.Equal(t, "", enc)

	dec, err := p.Decrypt("")
	require.NoError(t, err)
	assert.Equal(t, "", dec)
}
