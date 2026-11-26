package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaskSensitiveDataUUID(t *testing.T) {
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	masked := MaskSensitiveData("api key: " + uuid)

	assert.NotContains(t, masked, uuid)
	assert.Contains(t, masked, maskValue(uuid))
}

func TestMaskSensitiveDataUUIDCaseInsensitive(t *testing.T) {
	uuid := "550E8400-E29B-41D4-A716-446655440000"
	masked := MaskSensitiveData("key " + uuid)

	assert.NotContains(t, masked, uuid)
}

func TestMaskSensitiveDataVarious(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"json password", `{"password":"supersecret"}`},
		{"kv password", "password=hunter2"},
		{"json token", `{"token":"abc123def456"}`},
		{"kv api key", "api_key=abc123def456"},
		{"authorization", "Authorization: Bearer abc123def456"},
		{"postgres url", "postgres://user:secretpw@host:5432/db"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			masked := MaskSensitiveData(tc.input)
			assert.NotEqual(t, tc.input, masked)
			assert.NotContains(t, masked, "secre")
			assert.NotContains(t, masked, "hunter")
			assert.NotContains(t, masked, "abc123")
		})
	}
}

func TestMaskSensitiveDataPlainText(t *testing.T) {
	input := "the quick brown fox jumps over the lazy dog"
	assert.Equal(t, input, MaskSensitiveData(input))
}

type recordingWriter struct {
	written []string
}

func (rw *recordingWriter) Write(p []byte) (int, error) {
	rw.written = append(rw.written, string(p))
	return len(p), nil
}

func TestMaskingWriterReturnsFullLength(t *testing.T) {
	rw := &recordingWriter{}
	mw := NewMaskingWriter(rw)

	input := `{"password":"supersecret"}`
	n, err := mw.Write([]byte(input))

	require.NoError(t, err)
	assert.Equal(t, len(input), n)

	require.Len(t, rw.written, 1)
	assert.NotContains(t, rw.written[0], "supersecret")
	assert.True(t, len(rw.written[0]) < len(input))
}
