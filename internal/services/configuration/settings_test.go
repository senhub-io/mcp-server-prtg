package configuration

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func baseConfigData() ConfigData {
	var cfg ConfigData
	cfg.Server.Port = 8080
	cfg.Database.Host = "localhost"
	cfg.Database.Port = 5432
	cfg.Database.Name = "prtg"
	cfg.Database.User = "admin"
	cfg.Database.SSLMode = "disable"
	cfg.PRTG.Enabled = false
	cfg.PRTG.Timeout = 30
	cfg.Logging.Level = "info"
	return cfg
}

func validSettings() Settings {
	return Settings{
		Database: DatabaseSettings{
			Host:    "db.local",
			Port:    5432,
			Name:    "prtg",
			User:    "admin",
			SSLMode: "require",
		},
		PRTG: PRTGSettings{
			Enabled:        false,
			TimeoutSeconds: 30,
			VerifySSL:      true,
		},
		ServerPort: 9090,
		LogLevel:   "debug",
	}
}

func loadTestConfiguration(t *testing.T) (*Configuration, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	writeConfigFile(t, path, baseConfigData())
	cfg, err := LoadConfiguration(newTestArgs(path), newTestLogger(dir))
	if err != nil {
		t.Fatalf("LoadConfiguration: %v", err)
	}
	return cfg, path
}

func TestSettingsNeverExposesSecrets(t *testing.T) {
	cfg, _ := loadTestConfiguration(t)
	s := validSettings()
	pw := "s3cret-db"
	tok := "s3cret-token"
	s.Database.Password = &pw
	s.PRTG.Token = &tok
	if err := cfg.ApplySettings(s); err != nil {
		t.Fatalf("ApplySettings: %v", err)
	}

	snap := cfg.Settings()
	if snap.Database.Password != nil {
		t.Errorf("Settings exposed database password: %v", *snap.Database.Password)
	}
	if snap.PRTG.Token != nil {
		t.Errorf("Settings exposed PRTG token: %v", *snap.PRTG.Token)
	}
	if !snap.Database.PasswordSet {
		t.Error("PasswordSet should be true after setting a password")
	}
	if !snap.PRTG.TokenSet {
		t.Error("TokenSet should be true after setting a token")
	}
	if snap.Database.Host != "db.local" || snap.Database.Port != 5432 || snap.Database.SSLMode != "require" {
		t.Errorf("unexpected database snapshot: %+v", snap.Database)
	}
	if snap.ServerPort != 9090 || snap.LogLevel != "debug" || snap.PRTG.TimeoutSeconds != 30 || !snap.PRTG.VerifySSL {
		t.Errorf("unexpected snapshot: %+v", snap)
	}
}

func TestApplySettingsNilPasswordKeepsStored(t *testing.T) {
	cfg, path := loadTestConfiguration(t)
	s := validSettings()
	pw := "keepme"
	s.Database.Password = &pw
	if err := cfg.ApplySettings(s); err != nil {
		t.Fatalf("ApplySettings with password: %v", err)
	}
	stored := readConfigData(t, path).Database.Password

	s2 := validSettings()
	s2.Database.Password = nil
	if err := cfg.ApplySettings(s2); err != nil {
		t.Fatalf("ApplySettings with nil password: %v", err)
	}
	if !cfg.Settings().Database.PasswordSet {
		t.Error("nil password should keep the stored password")
	}
	if got := readConfigData(t, path).Database.Password; got != stored {
		t.Errorf("stored password changed: got %q, want %q", got, stored)
	}
}

func TestApplySettingsNewPasswordStoredEncrypted(t *testing.T) {
	cfg, path := loadTestConfiguration(t)
	s := validSettings()
	pw := "new-password"
	s.Database.Password = &pw
	if err := cfg.ApplySettings(s); err != nil {
		t.Fatalf("ApplySettings: %v", err)
	}

	onDisk := readConfigData(t, path)
	if !strings.HasPrefix(onDisk.Database.Password, "enc:") {
		t.Errorf("password on disk is not encrypted: %q", onDisk.Database.Password)
	}
	connStr := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		onDisk.Database.Host, onDisk.Database.Port, onDisk.Database.Name, onDisk.Database.User, onDisk.Database.Password, onDisk.Database.SSLMode)
	if !strings.Contains(connStr, onDisk.Database.Password) {
		t.Errorf("connection string does not contain the stored password: %s", connStr)
	}
}

func TestApplySettingsPRTGEnabledWithoutTokenRejected(t *testing.T) {
	cfg, path := loadTestConfiguration(t)
	original := readConfigData(t, path)

	s := validSettings()
	s.PRTG.Enabled = true
	s.PRTG.BaseURL = "https://prtg.example.com"
	s.PRTG.Token = nil
	err := cfg.ApplySettings(s)
	if err == nil {
		t.Fatal("expected error when enabling PRTG without a token")
	}
	if !errors.Is(err, ErrInvalidSettings) {
		t.Errorf("error should wrap ErrInvalidSettings, got %v", err)
	}
	if after := readConfigData(t, path); !reflect.DeepEqual(after, original) {
		t.Errorf("config file changed after rejected apply: got %+v, want %+v", after, original)
	}
}

func TestApplySettingsInvalidPortRejected(t *testing.T) {
	cfg, _ := loadTestConfiguration(t)

	s := validSettings()
	s.ServerPort = 0
	if err := cfg.ApplySettings(s); !errors.Is(err, ErrInvalidSettings) {
		t.Errorf("server port 0: expected ErrInvalidSettings, got %v", err)
	}

	s = validSettings()
	s.Database.Port = 70000
	if err := cfg.ApplySettings(s); !errors.Is(err, ErrInvalidSettings) {
		t.Errorf("database port 70000: expected ErrInvalidSettings, got %v", err)
	}
}

func TestApplySettingsEmptyPasswordClearsStored(t *testing.T) {
	cfg, _ := loadTestConfiguration(t)
	s := validSettings()
	pw := "to-be-cleared"
	s.Database.Password = &pw
	if err := cfg.ApplySettings(s); err != nil {
		t.Fatalf("ApplySettings with password: %v", err)
	}

	s2 := validSettings()
	empty := ""
	s2.Database.Password = &empty
	if err := cfg.ApplySettings(s2); err != nil {
		t.Fatalf("ApplySettings with empty password: %v", err)
	}
	if cfg.Settings().Database.PasswordSet {
		t.Error("empty string password should clear the stored password")
	}
}

func TestApplySettingsRefusesStoredSecretsForAnotherTarget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	writeConfigFile(t, path, ConfigData{
		ConfigVersion: CurrentConfigVersion,
		Server:        ServerConfig{APIKey: "key-1234567890abcdef", Port: 8443},
		Database:      DatabaseConfig{Host: "db", Port: 5432, Name: "prtg", User: "reader", Password: "pw", SSLMode: "disable"},
		PRTG:          PRTGConfig{Enabled: true, BaseURL: "https://prtg:1616", APIToken: "tok", Timeout: 30},
		Logging:       LoggingConfig{Level: "info"},
	})

	cfg, err := LoadConfiguration(newTestArgs(path), newTestLogger(dir))
	require.NoError(t, err)

	s := cfg.Settings()
	s.Database.Host = "attacker.example"
	err = cfg.ApplySettings(s)
	require.ErrorIs(t, err, ErrInvalidSettings)

	s = cfg.Settings()
	s.PRTG.BaseURL = "https://attacker.example:1616"
	err = cfg.ApplySettings(s)
	require.ErrorIs(t, err, ErrInvalidSettings)

	s = cfg.Settings()
	s.PRTG.Enabled = false
	s.PRTG.BaseURL = "https://attacker.example:1616"
	err = cfg.ApplySettings(s)
	require.ErrorIs(t, err, ErrInvalidSettings)

	s = cfg.Settings()
	s.Database.SSLMode = "require"
	require.NoError(t, cfg.ApplySettings(s))
	s = cfg.Settings()
	s.Database.SSLMode = "disable"
	err = cfg.ApplySettings(s)
	require.ErrorIs(t, err, ErrInvalidSettings)

	s = cfg.Settings()
	pw := "new"
	s.Database.Host = "other-db"
	s.Database.Password = &pw
	require.NoError(t, cfg.ApplySettings(s))
}
