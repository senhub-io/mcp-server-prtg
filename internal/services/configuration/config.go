package configuration

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"

	"github.com/senhub-io/mcp-server-prtg/internal/cliargs"
	"github.com/senhub-io/mcp-server-prtg/internal/secrets"
	"github.com/senhub-io/mcp-server-prtg/internal/services/logger"
)

const (
	CurrentConfigVersion = 1
	DefaultConfigFile    = "config.yaml"
)

// Configuration represents the complete server configuration.
type Configuration struct {
	configPath string
	logger     *logger.ModuleLogger
	watcher    *fsnotify.Watcher
	args       *cliargs.ParsedArgs
	secrets    secrets.Protector

	// plain holds the decrypted in-memory copies of the secrets; data always
	// holds the encrypted values once loaded.
	plain struct {
		apiKey     string
		dbPassword string
		prtgToken  string
	}

	// Configuration data
	data ConfigData

	// Callbacks
	onChangeCallbacks []func()

	// Shutdown signal for graceful goroutine termination
	shutdownCh chan struct{}
}

// ConfigData represents the YAML configuration structure.
type ConfigData struct {
	ConfigVersion int            `yaml:"config_version"`
	Server        ServerConfig   `yaml:"server"`
	Database      DatabaseConfig `yaml:"database"`
	PRTG          PRTGConfig     `yaml:"prtg"`
	Logging       LoggingConfig  `yaml:"logging"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	APIKey             string `yaml:"api_key"`              // API Key (Bearer token)
	BindAddress        string `yaml:"bind_address"`         // Address to bind to (e.g., 0.0.0.0)
	Port               int    `yaml:"port"`                 // Port to listen on
	EnableTLS          bool   `yaml:"enable_tls"`           // Enable HTTPS
	CertFile           string `yaml:"cert_file"`            // TLS certificate file
	KeyFile            string `yaml:"key_file"`             // TLS private key file
	ReadTimeout        int    `yaml:"read_timeout"`         // Read timeout in seconds
	WriteTimeout       int    `yaml:"write_timeout"`        // Write timeout in seconds
	AllowCustomQueries bool   `yaml:"allow_custom_queries"` // Allow custom SQL queries - DISABLE in production
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"sslmode"`
}

// PRTGConfig holds PRTG API connection settings for accessing historical metrics data.
type PRTGConfig struct {
	Enabled   bool   `yaml:"enabled"`    // Enable/disable PRTG API access
	BaseURL   string `yaml:"base_url"`   // PRTG server base URL (e.g., https://prtg.example.com)
	APIToken  string `yaml:"api_token"`  // PRTG API v2 token (Bearer authentication)
	Timeout   int    `yaml:"timeout"`    // HTTP request timeout in seconds
	VerifySSL bool   `yaml:"verify_ssl"` // Verify SSL certificates
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level      string `yaml:"level"`
	File       string `yaml:"file"`
	MaxSizeMB  int    `yaml:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAgeDays int    `yaml:"max_age_days"`
	Compress   bool   `yaml:"compress"`
}

// newConfigurationBase builds a Configuration with its secret protector, without loading or watching any file.
func newConfigurationBase(args *cliargs.ParsedArgs, baseLogger *logger.Logger) (*Configuration, error) {
	moduleLogger := logger.NewModuleLogger(baseLogger, logger.ModuleConfiguration)

	absPath, err := filepath.Abs(args.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("resolving configuration path %s: %w", args.ConfigPath, err)
	}

	protector, err := secrets.NewProtector(filepath.Dir(absPath))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize secret protector: %w", err)
	}

	return &Configuration{
		configPath:        args.ConfigPath,
		logger:            moduleLogger,
		args:              args,
		secrets:           protector,
		onChangeCallbacks: make([]func(), 0),
	}, nil
}

// NewConfiguration creates a new configuration manager.
func NewConfiguration(args *cliargs.ParsedArgs, baseLogger *logger.Logger) (*Configuration, error) {
	config, err := newConfigurationBase(args, baseLogger)
	if err != nil {
		return nil, err
	}

	// Load or create configuration
	if err := config.loadOrCreate(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize file watcher
	if err := config.initWatcher(); err != nil {
		config.logger.Warn().Err(err).Msg("Failed to initialize config file watcher")
	}

	return config, nil
}

// LoadConfiguration loads and decrypts an existing configuration file without creating it or starting the watcher.
func LoadConfiguration(args *cliargs.ParsedArgs, baseLogger *logger.Logger) (*Configuration, error) {
	config, err := newConfigurationBase(args, baseLogger)
	if err != nil {
		return nil, err
	}

	if err := config.loadConfiguration(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return config, nil
}

// InitConfiguration creates or updates the configuration file from command-line arguments without starting the watcher.
func InitConfiguration(args *cliargs.ParsedArgs, baseLogger *logger.Logger) (*Configuration, error) {
	config, err := newConfigurationBase(args, baseLogger)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(config.configPath); os.IsNotExist(err) {
		if err := config.createDefaultConfiguration(); err != nil {
			return nil, fmt.Errorf("failed to create default configuration: %w", err)
		}
	} else if err := config.loadConfiguration(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := config.applyArgs(args); err != nil {
		return nil, fmt.Errorf("failed to apply command-line settings: %w", err)
	}

	if err := config.saveConfiguration(); err != nil {
		return nil, fmt.Errorf("failed to save configuration: %w", err)
	}

	return config, nil
}

// applyArgs overrides only the settings explicitly provided on the command line.
func (c *Configuration) applyArgs(args *cliargs.ParsedArgs) error {
	if args.BindAddress != "" {
		c.data.Server.BindAddress = args.BindAddress
	}

	if args.Port != 0 {
		c.data.Server.Port = args.Port
	}

	if args.AuthKey != "" {
		c.plain.apiKey = args.AuthKey
	}

	if args.DBHost != "" {
		c.data.Database.Host = args.DBHost
	}

	if args.DBPort != 0 {
		c.data.Database.Port = args.DBPort
	}

	if args.DBName != "" {
		c.data.Database.Name = args.DBName
	}

	if args.DBUser != "" {
		c.data.Database.User = args.DBUser
	}

	if args.DBPassword != "" {
		c.plain.dbPassword = args.DBPassword
	}

	if args.DBSSLMode != "" {
		c.data.Database.SSLMode = args.DBSSLMode
	}

	if args.PRTGURL != "" {
		c.data.PRTG.BaseURL = args.PRTGURL
		c.data.PRTG.Enabled = true
	}

	if args.PRTGDisable {
		c.data.PRTG.Enabled = false
	}

	if args.PRTGToken != "" {
		c.plain.prtgToken = args.PRTGToken
	}

	if args.PRTGVerifySSL != nil {
		c.data.PRTG.VerifySSL = *args.PRTGVerifySSL
	}

	if args.PRTGTimeout != 0 {
		c.data.PRTG.Timeout = args.PRTGTimeout
	}

	if args.LogLevel != "" {
		c.data.Logging.Level = args.LogLevel
	}

	if args.LogFile != "" {
		c.data.Logging.File = args.LogFile
	}

	return c.encryptSecrets()
}

// loadOrCreate loads existing config or creates a new one.
func (c *Configuration) loadOrCreate() error {
	if _, err := os.Stat(c.configPath); os.IsNotExist(err) {
		c.logger.Info().Str("path", c.configPath).Msg("Configuration file not found, creating default")
		return c.createDefaultConfiguration()
	}

	if err := c.loadConfiguration(); err != nil {
		return err
	}

	// A deleted certificate pair is regenerated so the service can start
	// again; an existing pair, generated or provided, is never replaced.
	if c.data.Server.EnableTLS {
		if err := c.generateTLSCertificates(); err != nil {
			return fmt.Errorf("failed to generate missing TLS certificates: %w", err)
		}
	}

	return nil
}

// loadConfiguration loads configuration from YAML file.
func (c *Configuration) loadConfiguration() error {
	data, err := os.ReadFile(c.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse and resolve secrets into a copy so a bad file keeps the current
	// configuration, including the credentials in use, untouched.
	var parsed ConfigData
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	encrypted, err := c.processSecrets(&parsed)
	if err != nil {
		return err
	}

	// Configurations written by earlier versions carry doubled backslashes.
	if parsed.Server.CertFile != "" {
		parsed.Server.CertFile = filepath.Clean(parsed.Server.CertFile)
	}

	if parsed.Server.KeyFile != "" {
		parsed.Server.KeyFile = filepath.Clean(parsed.Server.KeyFile)
	}

	c.data = parsed

	if encrypted > 0 {
		if err := c.saveConfiguration(); err != nil {
			return fmt.Errorf("failed to rewrite configuration with encrypted secrets: %w", err)
		}

		c.logger.Info().Int("count", encrypted).Msg("encrypted clear-text secrets in configuration file")
	}

	c.logger.Info().
		Str("path", c.configPath).
		Int("version", c.data.ConfigVersion).
		Msg("Configuration loaded successfully")

	return nil
}

// processSecrets decrypts encrypted secrets into memory and encrypts clear-text ones in c.data.
// It returns the number of values that were clear text and therefore must be persisted encrypted.
func (c *Configuration) processSecrets(data *ConfigData) (int, error) {
	fields := []struct {
		name   string
		stored *string
		plain  *string
	}{
		{"server.api_key", &data.Server.APIKey, new(string)},
		{"database.password", &data.Database.Password, new(string)},
		{"prtg.api_token", &data.PRTG.APIToken, new(string)},
	}

	encrypted := 0

	for _, f := range fields {
		value := *f.stored

		switch {
		case secrets.IsEncrypted(value):
			plain, err := c.secrets.Decrypt(value)
			if err != nil {
				return 0, fmt.Errorf("%s: %w", f.name, err)
			}

			*f.plain = plain
		case value != "":
			enc, err := c.secrets.Encrypt(value)
			if err != nil {
				return 0, fmt.Errorf("%s: %w", f.name, err)
			}

			*f.stored = enc
			*f.plain = value
			encrypted++
		}
	}

	c.plain.apiKey = *fields[0].plain
	c.plain.dbPassword = *fields[1].plain
	c.plain.prtgToken = *fields[2].plain

	return encrypted, nil
}

// encryptSecrets writes the encrypted form of the plain in-memory secrets into c.data.
func (c *Configuration) encryptSecrets() error {
	var err error

	if c.data.Server.APIKey, err = c.secrets.Encrypt(c.plain.apiKey); err != nil {
		return fmt.Errorf("server.api_key: %w", err)
	}

	if c.data.Database.Password, err = c.secrets.Encrypt(c.plain.dbPassword); err != nil {
		return fmt.Errorf("database.password: %w", err)
	}

	if c.data.PRTG.APIToken, err = c.secrets.Encrypt(c.plain.prtgToken); err != nil {
		return fmt.Errorf("prtg.api_token: %w", err)
	}

	return nil
}

// createDefaultConfiguration creates a default configuration file.
func (c *Configuration) createDefaultConfiguration() error {
	// Generate API key if not provided
	apiKey := c.args.AuthKey

	if apiKey == "" {
		var err error

		apiKey, err = generateUUIDKey()
		if err != nil {
			return fmt.Errorf("failed to generate API key: %w", err)
		}

		c.logger.Info().Msg("Generated new API key (Bearer token)")
	}

	// Certificates live next to the configuration file, so an installer that
	// owns the program directory never finds generated files in it.
	configPath, err := filepath.Abs(c.args.ConfigPath)
	if err != nil {
		return fmt.Errorf("resolving configuration path %s: %w", c.args.ConfigPath, err)
	}

	configDir := filepath.Dir(configPath)

	defaultCertFile := filepath.Join(configDir, "certs", "server.crt")
	defaultKeyFile := filepath.Join(configDir, "certs", "server.key")

	c.logger.Debug().
		Str("config_dir", configDir).
		Str("cert_file", defaultCertFile).
		Str("key_file", defaultKeyFile).
		Msg("Using configuration directory for certificate paths")

	// Create default config
	c.data = ConfigData{
		ConfigVersion: CurrentConfigVersion,
		Server: ServerConfig{
			APIKey:             apiKey,
			BindAddress:        getOrDefault(c.args.BindAddress, "0.0.0.0"),
			Port:               getOrDefaultInt(c.args.Port, 8443),
			EnableTLS:          c.args.EnableHTTPS,
			CertFile:           getOrDefault(c.args.CertFile, defaultCertFile),
			KeyFile:            getOrDefault(c.args.KeyFile, defaultKeyFile),
			ReadTimeout:        0,     // No timeout for SSE connections
			WriteTimeout:       0,     // No timeout for SSE connections
			AllowCustomQueries: false, // SECURITY: Disable custom SQL queries by default - enable only in dev/test
		},
		Database: DatabaseConfig{
			Host:     getOrDefault(c.args.DBHost, "localhost"),
			Port:     getOrDefaultInt(c.args.DBPort, 5432),
			Name:     getOrDefault(c.args.DBName, "prtg_data_exporter_collection"),
			User:     getOrDefault(c.args.DBUser, "mcp_prtg_reader"),
			Password: c.args.DBPassword,
			SSLMode:  getOrDefault(c.args.DBSSLMode, "disable"),
		},
		PRTG: PRTGConfig{
			Enabled:   false, // Disabled by default - opt-in for PRTG API access
			BaseURL:   "",    // Example: https://prtg.example.com
			APIToken:  "",    // PRTG API v2 token
			Timeout:   30,    // 30 seconds default timeout
			VerifySSL: true,  // Verify SSL by default for security
		},
		Logging: LoggingConfig{
			Level:      getOrDefault(c.args.LogLevel, "info"),
			File:       c.args.LogFile,
			MaxSizeMB:  10,
			MaxBackups: 5,
			MaxAgeDays: 30,
			Compress:   true,
		},
	}

	c.plain.apiKey = c.data.Server.APIKey
	c.plain.dbPassword = c.data.Database.Password
	c.plain.prtgToken = c.data.PRTG.APIToken

	if err := c.encryptSecrets(); err != nil {
		return err
	}

	// Generate TLS certificates if enabled and not provided
	if c.data.Server.EnableTLS && c.args.CertFile == "" {
		if err := c.generateTLSCertificates(); err != nil {
			c.logger.Warn().Err(err).Msg("Failed to generate TLS certificates")
		}
	}

	return c.saveConfiguration()
}

// saveConfiguration saves configuration to YAML file.
func (c *Configuration) saveConfiguration() error {
	// For Windows paths, we need to generate YAML manually to ensure proper quoting
	// The yaml.Marshal doesn't add quotes around strings with backslashes
	var yamlData []byte

	var err error

	if filepath.Separator == '\\' {
		// On Windows, generate YAML manually with quoted paths
		yamlData, err = c.generateWindowsYAML()
	} else {
		// On Unix, use standard marshaller
		yamlData, err = yaml.Marshal(&c.data)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(c.configPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write with restricted permissions (0600 for security)
	if err := os.WriteFile(c.configPath, yamlData, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	c.logger.Info().
		Str("path", c.configPath).
		Msg("Configuration saved successfully")

	return nil
}

// generateWindowsYAML generates YAML with properly quoted Windows paths.
func (c *Configuration) generateWindowsYAML() ([]byte, error) {
	// Use yaml.Marshal for most fields, but manually format cert paths with quotes
	yamlData, err := yaml.Marshal(&c.data)
	if err != nil {
		return nil, err
	}

	// Replace unquoted Windows paths with quoted ones
	// This ensures YAML parsers correctly interpret backslashes
	yamlStr := string(yamlData)

	// Quote cert_file if it contains backslashes and isn't already quoted
	certFile := c.data.Server.CertFile
	if strings.Contains(certFile, "\\") && !strings.HasPrefix(certFile, "\"") {
		yamlStr = strings.ReplaceAll(yamlStr,
			fmt.Sprintf("cert_file: %s", certFile),
			fmt.Sprintf("cert_file: %q", certFile))
	}

	// Quote key_file if it contains backslashes and isn't already quoted
	keyFile := c.data.Server.KeyFile
	if strings.Contains(keyFile, "\\") && !strings.HasPrefix(keyFile, "\"") {
		yamlStr = strings.ReplaceAll(yamlStr,
			fmt.Sprintf("key_file: %s", keyFile),
			fmt.Sprintf("key_file: %q", keyFile))
	}

	return []byte(yamlStr), nil
}

// generateTLSCertificates generates self-signed TLS certificate and key.
func (c *Configuration) generateTLSCertificates() error {
	// Check if certificates already exist - don't overwrite them
	certExists := false
	keyExists := false

	if _, err := os.Stat(c.data.Server.CertFile); err == nil {
		certExists = true
	}

	if _, err := os.Stat(c.data.Server.KeyFile); err == nil {
		keyExists = true
	}

	// If both certificate files exist, don't regenerate (user may have provided their own)
	if certExists && keyExists {
		c.logger.Info().
			Str("cert", c.data.Server.CertFile).
			Str("key", c.data.Server.KeyFile).
			Msg("TLS certificates already exist, skipping generation")

		return nil
	}

	c.logger.Info().Msg("Generating self-signed TLS certificates")

	// Create certs directory
	certsDir := filepath.Dir(c.data.Server.CertFile)
	if err := os.MkdirAll(certsDir, 0750); err != nil {
		return fmt.Errorf("failed to create certs directory: %w", err)
	}

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),
		Subject: pkix.Name{
			Organization: []string{"MCP Server PRTG"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Add SANs (Subject Alternative Names)
	// MCP clients reach the server by its machine name or address, not only
	// through localhost: the certificate must cover them or TLS fails.
	template.DNSNames, template.IPAddresses = certificateNames()

	// Generate certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Write certificate file (0600 - secure permissions)
	if err := os.WriteFile(c.data.Server.CertFile, certPEM, 0600); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	// Write private key file (0600 - owner only)
	if err := os.WriteFile(c.data.Server.KeyFile, keyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	c.logger.Info().
		Str("cert", c.data.Server.CertFile).
		Str("key", c.data.Server.KeyFile).
		Msg("TLS certificates generated successfully")

	return nil
}

// generateUUIDKey generates a UUID v4 for API key.
func generateUUIDKey() (string, error) {
	uuid := make([]byte, 16)
	if _, err := rand.Read(uuid); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Set version (4) and variant bits
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant bits

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16]), nil
}

// Get methods for accessing configuration.

// Path returns the configuration file path.
func (c *Configuration) Path() string {
	return c.configPath
}

// GetAPIKey returns the API key (Bearer token).
func (c *Configuration) GetAPIKey() string {
	return c.plain.apiKey
}

// GetServerAddress returns the full server address.
func (c *Configuration) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.data.Server.BindAddress, c.data.Server.Port)
}

// GetDatabaseConnectionString returns the PostgreSQL connection string.
func (c *Configuration) GetDatabaseConnectionString() string {
	return fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		quoteConnValue(c.data.Database.Host),
		c.data.Database.Port,
		quoteConnValue(c.data.Database.Name),
		quoteConnValue(c.data.Database.User),
		quoteConnValue(c.plain.dbPassword),
		quoteConnValue(c.data.Database.SSLMode),
	)
}

// quoteConnValue quotes a libpq keyword/value: unquoted, an empty or
// space-containing value shifts the following keywords into it.
func quoteConnValue(v string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(v) + "'"
}

// GetDatabaseHost returns the database host.
func (c *Configuration) GetDatabaseHost() string {
	return c.data.Database.Host
}

// GetDatabasePort returns the database port.
func (c *Configuration) GetDatabasePort() int {
	return c.data.Database.Port
}

// GetDatabaseName returns the database name.
func (c *Configuration) GetDatabaseName() string {
	return c.data.Database.Name
}

// GetDatabaseUser returns the database user.
func (c *Configuration) GetDatabaseUser() string {
	return c.data.Database.User
}

// GetDatabaseSSLMode returns the database SSL mode.
func (c *Configuration) GetDatabaseSSLMode() string {
	return c.data.Database.SSLMode
}

// IsTLSEnabled returns whether TLS is enabled.
func (c *Configuration) IsTLSEnabled() bool {
	return c.data.Server.EnableTLS
}

// GetTLSCertFile returns the TLS certificate file path.
func (c *Configuration) GetTLSCertFile() string {
	return c.data.Server.CertFile
}

// GetTLSKeyFile returns the TLS private key file path.
func (c *Configuration) GetTLSKeyFile() string {
	return c.data.Server.KeyFile
}

// GetReadTimeout returns the server read timeout.
func (c *Configuration) GetReadTimeout() time.Duration {
	return time.Duration(c.data.Server.ReadTimeout) * time.Second
}

// GetWriteTimeout returns the server write timeout.
func (c *Configuration) GetWriteTimeout() time.Duration {
	return time.Duration(c.data.Server.WriteTimeout) * time.Second
}

// AllowCustomQueries returns whether custom SQL queries are allowed.
// SECURITY: This should be false in production environments to prevent SQL injection risks.
func (c *Configuration) AllowCustomQueries() bool {
	return c.data.Server.AllowCustomQueries
}

// IsPRTGEnabled returns whether PRTG API access is enabled.
func (c *Configuration) IsPRTGEnabled() bool {
	return c.data.PRTG.Enabled
}

// GetPRTGBaseURL returns the PRTG server base URL.
func (c *Configuration) GetPRTGBaseURL() string {
	return c.data.PRTG.BaseURL
}

// GetPRTGAPIToken returns the PRTG API token.
func (c *Configuration) GetPRTGAPIToken() string {
	return c.plain.prtgToken
}

// GetPRTGTimeout returns the PRTG API timeout duration.
func (c *Configuration) GetPRTGTimeout() time.Duration {
	return time.Duration(c.data.PRTG.Timeout) * time.Second
}

// IsPRTGSSLVerifyEnabled returns whether SSL certificate verification is enabled for PRTG API.
func (c *Configuration) IsPRTGSSLVerifyEnabled() bool {
	return c.data.PRTG.VerifySSL
}

// Helper functions.

func getOrDefault(value, defaultValue string) string {
	if value != "" {
		return value
	}

	return defaultValue
}

func getOrDefaultInt(value, defaultValue int) int {
	if value != 0 {
		return value
	}

	return defaultValue
}

// initWatcher initializes file watcher for hot-reload.
func (c *Configuration) initWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	if err := watcher.Add(c.configPath); err != nil {
		if closeErr := watcher.Close(); closeErr != nil {
			c.logger.Error().Err(closeErr).Msg("Failed to close watcher after Add error")
		}

		return fmt.Errorf("failed to watch config file: %w", err)
	}

	c.watcher = watcher
	c.shutdownCh = make(chan struct{})

	// Start watching in background
	go c.watchConfigFile()

	c.logger.Info().Str("path", c.configPath).Msg("Config file watcher initialized")

	return nil
}

// watchConfigFile watches for configuration file changes.
func (c *Configuration) watchConfigFile() {
	for {
		select {
		case <-c.shutdownCh:
			// Graceful shutdown - exit goroutine immediately
			c.logger.Debug().Msg("Config file watcher shutting down")
			return

		case event, ok := <-c.watcher.Events:
			if !ok {
				return
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				c.logger.Info().Str("path", event.Name).Msg("Configuration file changed, reloading")

				if err := c.loadConfiguration(); err != nil {
					c.logger.Error().Err(err).Msg("Failed to reload configuration")
					continue
				}

				// Notify callbacks
				for _, callback := range c.onChangeCallbacks {
					callback()
				}
			}

		case err, ok := <-c.watcher.Errors:
			if !ok {
				return
			}

			c.logger.Error().Err(err).Msg("File watcher error")
		}
	}
}

// OnConfigChanged registers a callback for configuration changes.
func (c *Configuration) OnConfigChanged(callback func()) {
	c.onChangeCallbacks = append(c.onChangeCallbacks, callback)
}

// Shutdown stops the configuration manager gracefully.
func (c *Configuration) Shutdown(_ context.Context) error {
	// Signal the watcher goroutine to exit (close only once)
	if c.shutdownCh != nil {
		select {
		case <-c.shutdownCh:
			// Already closed
		default:
			close(c.shutdownCh)
		}
	}

	// Close the file watcher
	if c.watcher != nil {
		return c.watcher.Close()
	}

	return nil
}

// certificateNames returns the host names and addresses the self-signed
// certificate is issued for: localhost, the machine name and its addresses.
func certificateNames() ([]string, []net.IP) {
	names := []string{"localhost"}
	ips := []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}

	if host, err := os.Hostname(); err == nil && host != "" {
		names = append(names, host)

		if short, _, found := strings.Cut(host, "."); found && short != "" {
			names = append(names, short)
		}
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return names, ips
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() || ipNet.IP.IsLinkLocalUnicast() {
			continue
		}

		ips = append(ips, ipNet.IP)
	}

	return names, ips
}
