package configuration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/senhub-io/mcp-server-prtg/internal/cliargs"
	"github.com/senhub-io/mcp-server-prtg/internal/secrets"
	"github.com/senhub-io/mcp-server-prtg/internal/services/logger"
)

func newTestLogger(dir string) *logger.Logger {
	return logger.NewLogger(&cliargs.ParsedArgs{
		LogLevel: "error",
		LogFile:  filepath.Join(dir, "test.log"),
	})
}

func newTestArgs(path string) *cliargs.ParsedArgs {
	return &cliargs.ParsedArgs{
		ConfigPath: path,
		LogLevel:   "error",
		LogFile:    filepath.Join(filepath.Dir(path), "test.log"),
	}
}

func writeConfigFile(t *testing.T, path string, cfg ConfigData) {
	t.Helper()

	data, err := yaml.Marshal(&cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0600))
}

func readConfigData(t *testing.T, path string) ConfigData {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var cfg ConfigData
	require.NoError(t, yaml.Unmarshal(data, &cfg))

	return cfg
}

func TestLoadConfigurationEncryptsClearTextSecrets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	writeConfigFile(t, path, ConfigData{
		ConfigVersion: CurrentConfigVersion,
		Server:        ServerConfig{APIKey: "api-key-123", Port: 8443},
		Database:      DatabaseConfig{Host: "db-host", Password: "db-pass"},
		PRTG:          PRTGConfig{APIToken: "prtg-token"},
	})

	cfg, err := LoadConfiguration(newTestArgs(path), newTestLogger(dir))
	require.NoError(t, err)

	assert.Equal(t, "api-key-123", cfg.GetAPIKey())
	assert.Contains(t, cfg.GetDatabaseConnectionString(), "db-pass")
	assert.Equal(t, "prtg-token", cfg.GetPRTGAPIToken())

	onDisk := readConfigData(t, path)
	assert.True(t, secrets.IsEncrypted(onDisk.Server.APIKey))
	assert.True(t, secrets.IsEncrypted(onDisk.Database.Password))
	assert.True(t, secrets.IsEncrypted(onDisk.PRTG.APIToken))
}

func TestLoadConfigurationWithEncryptedSecrets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	args := newTestArgs(path)
	args.AuthKey = "api-key-123"
	args.DBPassword = "db-pass"
	args.PRTGToken = "prtg-token"

	_, err := InitConfiguration(args, newTestLogger(dir))
	require.NoError(t, err)

	cfg, err := LoadConfiguration(newTestArgs(path), newTestLogger(dir))
	require.NoError(t, err)

	assert.Equal(t, "api-key-123", cfg.GetAPIKey())
	assert.Contains(t, cfg.GetDatabaseConnectionString(), "db-pass")
	assert.Equal(t, "prtg-token", cfg.GetPRTGAPIToken())
}

func TestInitConfigurationCreatesEncryptedFileWithPRTG(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	args := newTestArgs(path)
	args.DBPassword = "db-pass"
	args.PRTGToken = "prtg-token"
	args.PRTGURL = "https://prtg.example.com:1616"

	cfg, err := InitConfiguration(args, newTestLogger(dir))
	require.NoError(t, err)
	assert.True(t, cfg.IsPRTGEnabled())

	onDisk := readConfigData(t, path)
	assert.True(t, secrets.IsEncrypted(onDisk.Server.APIKey))
	assert.True(t, secrets.IsEncrypted(onDisk.Database.Password))
	assert.True(t, secrets.IsEncrypted(onDisk.PRTG.APIToken))
	assert.True(t, onDisk.PRTG.Enabled)
	assert.Equal(t, "https://prtg.example.com:1616", onDisk.PRTG.BaseURL)
}

func TestInitConfigurationOverridesOnlyGivenSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	args := newTestArgs(path)
	args.AuthKey = "api-key-123"
	args.DBHost = "first-host"
	args.DBPassword = "db-pass"

	_, err := InitConfiguration(args, newTestLogger(dir))
	require.NoError(t, err)

	args2 := newTestArgs(path)
	args2.DBHost = "second-host"

	cfg, err := InitConfiguration(args2, newTestLogger(dir))
	require.NoError(t, err)

	assert.Equal(t, "api-key-123", cfg.GetAPIKey())
	assert.Equal(t, "second-host", cfg.GetDatabaseHost())
	assert.Contains(t, cfg.GetDatabaseConnectionString(), "db-pass")

	onDisk := readConfigData(t, path)
	assert.Equal(t, "second-host", onDisk.Database.Host)
	assert.True(t, secrets.IsEncrypted(onDisk.Server.APIKey))
	assert.True(t, secrets.IsEncrypted(onDisk.Database.Password))
}

func TestLoadConfigurationDoesNotCreateMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	_, err := LoadConfiguration(newTestArgs(path), newTestLogger(dir))
	require.Error(t, err)

	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr))
}

func TestReloadOfBrokenFileKeepsCurrentSecrets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	writeConfigFile(t, path, ConfigData{
		ConfigVersion: CurrentConfigVersion,
		Server:        ServerConfig{APIKey: "key-1234567890abcdef"},
		Database:      DatabaseConfig{Password: "db-secret"},
	})

	cfg, err := LoadConfiguration(newTestArgs(path), newTestLogger(dir))
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(path, []byte("server: [not a map"), 0600))
	require.Error(t, cfg.loadConfiguration())

	assert.Equal(t, "key-1234567890abcdef", cfg.GetAPIKey())
	assert.Contains(t, cfg.GetDatabaseConnectionString(), "password='db-secret'")
}
