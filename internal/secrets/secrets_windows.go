//go:build windows

package secrets

import (
	"encoding/base64"
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	dpapiPrefix  = "enc:dpapi:"
	dpapiEntropy = "mcp-server-prtg"
)

type dpapiProtector struct{}

func newPlatformProtector(_ string) (Protector, error) {
	return &dpapiProtector{}, nil
}

func (p *dpapiProtector) Encrypt(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}

	data := []byte(plain)
	entropy := []byte(dpapiEntropy)

	var in, ent, out windows.DataBlob
	in.Data, in.Size = &data[0], uint32(len(data))
	ent.Data, ent.Size = &entropy[0], uint32(len(entropy))

	flags := uint32(windows.CRYPTPROTECT_LOCAL_MACHINE | windows.CRYPTPROTECT_UI_FORBIDDEN)
	if err := windows.CryptProtectData(&in, nil, &ent, 0, nil, flags, &out); err != nil {
		return "", fmt.Errorf("DPAPI encryption failed: %w", err)
	}

	blob := make([]byte, out.Size)
	copy(blob, unsafe.Slice(out.Data, out.Size))
	if _, err := windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data))); err != nil {
		return "", fmt.Errorf("failed to free DPAPI blob: %w", err)
	}

	return dpapiPrefix + base64.StdEncoding.EncodeToString(blob), nil
}

func (p *dpapiProtector) Decrypt(value string) (string, error) {
	if !IsEncrypted(value) {
		return value, nil
	}

	if !strings.HasPrefix(value, dpapiPrefix) {
		return "", errOtherPlatform
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, dpapiPrefix))
	if err != nil || len(raw) == 0 {
		return "", fmt.Errorf("%w: %v", errCannotDecrypt, err)
	}

	entropy := []byte(dpapiEntropy)

	var in, ent, out windows.DataBlob
	in.Data, in.Size = &raw[0], uint32(len(raw))
	ent.Data, ent.Size = &entropy[0], uint32(len(entropy))

	flags := uint32(windows.CRYPTPROTECT_LOCAL_MACHINE | windows.CRYPTPROTECT_UI_FORBIDDEN)
	if err := windows.CryptUnprotectData(&in, nil, &ent, 0, nil, flags, &out); err != nil {
		return "", fmt.Errorf("%w: %w", errCannotDecrypt, err)
	}

	plain := make([]byte, out.Size)
	copy(plain, unsafe.Slice(out.Data, out.Size))
	if _, err := windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data))); err != nil {
		return "", fmt.Errorf("failed to free DPAPI blob: %w", err)
	}

	return string(plain), nil
}
