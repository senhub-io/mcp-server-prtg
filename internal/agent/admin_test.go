package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/senhub-io/mcp-server-prtg/internal/cliargs"
	"github.com/senhub-io/mcp-server-prtg/internal/services/configuration"
	"github.com/senhub-io/mcp-server-prtg/internal/services/logger"
)

func testRuntime(t *testing.T) adminRuntime {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := "config_version: 1\nserver:\n  api_key: test-key-1234567890\n  bind_address: 0.0.0.0\n  port: 8443\n  enable_tls: true\n" +
		"  cert_file: " + filepath.Join(dir, "server.crt") + "\n"
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	args := &cliargs.ParsedArgs{ConfigPath: path, LogLevel: "error", LogFile: filepath.Join(dir, "test.log")}
	cfg, err := configuration.LoadConfiguration(args, logger.NewSilentLogger())
	require.NoError(t, err)

	l := zerolog.Nop()

	return adminRuntime{config: cfg, started: time.Now(), logger: &l}
}

func TestAdminClientConfig(t *testing.T) {
	rt := testRuntime(t)

	text, resolved, err := rt.clientConfig("mcp.example.lan")
	require.NoError(t, err)
	assert.Equal(t, "mcp.example.lan", resolved)
	assert.Contains(t, text, "https://mcp.example.lan:8443/mcp")
	assert.Equal(t, 4, strings.Count(text, "test-key-1234567890"))
}

func TestAdminTestPRTGEmptyURL(t *testing.T) {
	rt := testRuntime(t)

	res := rt.testPRTG(context.Background(), configuration.PRTGSettings{})
	assert.False(t, res.OK)
	assert.Equal(t, "Enter the PRTG URL", res.Message)
}

func TestAdminStatusWithoutDatabase(t *testing.T) {
	rt := testRuntime(t)

	status, ok := rt.status(context.Background()).(map[string]any)
	require.True(t, ok)
	db, ok := status["database"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "not_configured", db["status"])
}
