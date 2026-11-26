package configuration

import (
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidSettings = errors.New("invalid settings")

type DatabaseSettings struct {
	Host        string  `json:"host"`
	Port        int     `json:"port"`
	Name        string  `json:"name"`
	User        string  `json:"user"`
	SSLMode     string  `json:"sslmode"`            // disable, require, verify-ca, verify-full
	Password    *string `json:"password,omitempty"` // input only: nil keeps the stored password
	PasswordSet bool    `json:"password_set"`       // output only
}

type PRTGSettings struct {
	Enabled        bool    `json:"enabled"`
	BaseURL        string  `json:"base_url"`
	Token          *string `json:"token,omitempty"` // input only: nil keeps the stored token
	TokenSet       bool    `json:"token_set"`       // output only
	TimeoutSeconds int     `json:"timeout_seconds"`
	VerifySSL      bool    `json:"verify_ssl"`
}

type Settings struct {
	Database   DatabaseSettings `json:"database"`
	PRTG       PRTGSettings     `json:"prtg"`
	ServerPort int              `json:"server_port"`
	LogLevel   string           `json:"log_level"` // debug, info, warn, error
}

func (c *Configuration) Settings() Settings {
	return Settings{
		Database: DatabaseSettings{
			Host:        c.data.Database.Host,
			Port:        c.data.Database.Port,
			Name:        c.data.Database.Name,
			User:        c.data.Database.User,
			SSLMode:     c.data.Database.SSLMode,
			PasswordSet: c.plain.dbPassword != "",
		},
		PRTG: PRTGSettings{
			Enabled:        c.data.PRTG.Enabled,
			BaseURL:        c.data.PRTG.BaseURL,
			TokenSet:       c.plain.prtgToken != "",
			TimeoutSeconds: c.data.PRTG.Timeout,
			VerifySSL:      c.data.PRTG.VerifySSL,
		},
		ServerPort: c.data.Server.Port,
		LogLevel:   c.data.Logging.Level,
	}
}

func (c *Configuration) ApplySettings(s Settings) error {
	if err := validateSettings(s, c.plain.prtgToken != ""); err != nil {
		return err
	}

	// A stored secret is only reused for the target it was entered for:
	// pointing it at another server would hand it to that server.
	if s.Database.Password == nil && c.plain.dbPassword != "" && !c.SameDatabaseTarget(s.Database) {
		return fmt.Errorf("%w: enter the database password again when changing the host, port, user or SSL mode", ErrInvalidSettings)
	}

	if s.PRTG.Token == nil && c.plain.prtgToken != "" && !c.SamePRTGTarget(s.PRTG) {
		return fmt.Errorf("%w: enter the PRTG API token again when changing the PRTG URL or turning off certificate verification", ErrInvalidSettings)
	}

	data := c.data
	data.Server.Port = s.ServerPort
	data.Database.Host = s.Database.Host
	data.Database.Port = s.Database.Port
	data.Database.Name = s.Database.Name
	data.Database.User = s.Database.User
	data.Database.SSLMode = s.Database.SSLMode
	data.PRTG.Enabled = s.PRTG.Enabled
	data.PRTG.BaseURL = s.PRTG.BaseURL
	data.PRTG.Timeout = s.PRTG.TimeoutSeconds
	data.PRTG.VerifySSL = s.PRTG.VerifySSL
	data.Logging.Level = s.LogLevel

	dbPassword := c.plain.dbPassword
	if s.Database.Password != nil {
		dbPassword = *s.Database.Password
		enc, err := c.secrets.Encrypt(dbPassword)
		if err != nil {
			return fmt.Errorf("encrypt database password: %w", err)
		}
		data.Database.Password = enc
	}

	prtgToken := c.plain.prtgToken
	if s.PRTG.Token != nil {
		prtgToken = *s.PRTG.Token
		enc, err := c.secrets.Encrypt(prtgToken)
		if err != nil {
			return fmt.Errorf("encrypt PRTG API token: %w", err)
		}
		data.PRTG.APIToken = enc
	}

	c.data = data
	c.plain.dbPassword = dbPassword
	c.plain.prtgToken = prtgToken
	if err := c.saveConfiguration(); err != nil {
		return fmt.Errorf("save configuration: %w", err)
	}
	return nil
}

func validateSettings(s Settings, hasStoredToken bool) error {
	if s.ServerPort < 1 || s.ServerPort > 65535 {
		return fmt.Errorf("%w: server port must be between 1 and 65535, got %d", ErrInvalidSettings, s.ServerPort)
	}
	if s.Database.Port < 1 || s.Database.Port > 65535 {
		return fmt.Errorf("%w: database port must be between 1 and 65535, got %d", ErrInvalidSettings, s.Database.Port)
	}
	if strings.TrimSpace(s.Database.Host) == "" {
		return fmt.Errorf("%w: database host must not be empty", ErrInvalidSettings)
	}
	if strings.TrimSpace(s.Database.Name) == "" {
		return fmt.Errorf("%w: database name must not be empty", ErrInvalidSettings)
	}
	if strings.TrimSpace(s.Database.User) == "" {
		return fmt.Errorf("%w: database user must not be empty", ErrInvalidSettings)
	}
	switch s.Database.SSLMode {
	case "disable", "require", "verify-ca", "verify-full":
	default:
		return fmt.Errorf("%w: database sslmode must be one of disable, require, verify-ca, verify-full, got %q", ErrInvalidSettings, s.Database.SSLMode)
	}
	switch s.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("%w: log level must be one of debug, info, warn, error, got %q", ErrInvalidSettings, s.LogLevel)
	}
	if s.PRTG.TimeoutSeconds < 1 || s.PRTG.TimeoutSeconds > 300 {
		return fmt.Errorf("%w: PRTG timeout must be between 1 and 300 seconds, got %d", ErrInvalidSettings, s.PRTG.TimeoutSeconds)
	}
	if s.PRTG.Enabled {
		if !strings.HasPrefix(s.PRTG.BaseURL, "https://") && !strings.HasPrefix(s.PRTG.BaseURL, "http://") {
			return fmt.Errorf("%w: PRTG base URL must start with http:// or https://, got %q", ErrInvalidSettings, s.PRTG.BaseURL)
		}
		tokenProvided := s.PRTG.Token != nil && *s.PRTG.Token != ""
		if !tokenProvided && !hasStoredToken {
			return fmt.Errorf("%w: PRTG is enabled but no API token is configured; provide a token", ErrInvalidSettings)
		}
	}
	return nil
}

// SameDatabaseTarget reports whether s points at the stored database server and user
// with an SSL mode at least as strict, so a stored password is never sent over a weaker channel.
func (c *Configuration) SameDatabaseTarget(s DatabaseSettings) bool {
	d := c.data.Database

	return s.Host == d.Host && s.Port == d.Port && s.User == d.User && sslModeRank(s.SSLMode) >= sslModeRank(d.SSLMode)
}

func sslModeRank(mode string) int {
	switch mode {
	case "require":
		return 1
	case "verify-ca":
		return 2
	case "verify-full":
		return 3
	default:
		return 0
	}
}

// SamePRTGTarget reports whether s points at the stored PRTG API URL without
// turning off certificate verification.
func (c *Configuration) SamePRTGTarget(s PRTGSettings) bool {
	return s.BaseURL == c.data.PRTG.BaseURL && (s.VerifySSL || !c.data.PRTG.VerifySSL)
}
