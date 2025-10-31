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

	"github.com/matthieu/mcp-server-prtg/internal/cliargs"
	"github.com/matthieu/mcp-server-prtg/internal/services/logger"
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
	Logging       LoggingConfig  `yaml:"logging"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	APIKey             string `yaml:"api_key"`              // API Key (Bearer token)
	BindAddress        string `yaml:"bind_address"`         // Address to bind to (e.g., 0.0.0.0)
	Port               int    `yaml:"port"`                 // Port to listen on
	PublicURL          string `yaml:"public_url"`           // Public URL for SSE endpoint (optional)
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

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level      string `yaml:"level"`
	File       string `yaml:"file"`
	MaxSizeMB  int    `yaml:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAgeDays int    `yaml:"max_age_days"`
	Compress   bool   `yaml:"compress"`
}

// NewConfiguration creates a new configuration manager.
func NewConfiguration(args *cliargs.ParsedArgs, baseLogger *logger.Logger) (*Configuration, error) {
	logger := logger.NewModuleLogger(baseLogger, logger.ModuleConfiguration)

	config := &Configuration{
		configPath:        args.ConfigPath,
		logger:            logger,
		args:              args,
		onChangeCallbacks: make([]func(), 0),
	}

	// Load or create configuration
	if err := config.loadOrCreate(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize file watcher
	if err := config.initWatcher(); err != nil {
		logger.Warn().Err(err).Msg("Failed to initialize config file watcher")
	}

	return config, nil
}

// loadOrCreate loads existing config or creates a new one.
func (c *Configuration) loadOrCreate() error {
	if _, err := os.Stat(c.configPath); os.IsNotExist(err) {
		c.logger.Info().Str("path", c.configPath).Msg("Configuration file not found, creating default")
		return c.createDefaultConfiguration()
	}

	return c.loadConfiguration()
}

// loadConfiguration loads configuration from YAML file.
func (c *Configuration) loadConfiguration() error {
	data, err := os.ReadFile(c.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &c.data); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	c.logger.Info().
		Str("path", c.configPath).
		Int("version", c.data.ConfigVersion).
		Msg("Configuration loaded successfully")

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

	// Get absolute paths for certificates (relative to executable directory)
	// This ensures paths work correctly when running as a Windows service
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	exeDir := filepath.Dir(exePath)

	defaultCertFile := filepath.Join(exeDir, "certs", "server.crt")
	defaultKeyFile := filepath.Join(exeDir, "certs", "server.key")

	// On Windows, double backslashes for YAML compatibility
	// Windows accepts both C:\\path and C:/path but YAML needs escaped backslashes
	if filepath.Separator == '\\' {
		defaultCertFile = strings.ReplaceAll(defaultCertFile, "\\", "\\\\")
		defaultKeyFile = strings.ReplaceAll(defaultKeyFile, "\\", "\\\\")
	}

	c.logger.Debug().
		Str("exe_dir", exeDir).
		Str("cert_file", defaultCertFile).
		Str("key_file", defaultKeyFile).
		Msg("Using executable directory for certificate paths")

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
			Name:     getOrDefault(c.args.DBName, "prtg_data_exporter"),
			User:     getOrDefault(c.args.DBUser, "prtg_reader"),
			Password: c.args.DBPassword,
			SSLMode:  getOrDefault(c.args.DBSSLMode, "disable"),
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
	template.DNSNames = []string{"localhost"}
	template.IPAddresses = []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}

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

// GetAPIKey returns the API key (Bearer token).
func (c *Configuration) GetAPIKey() string {
	return c.data.Server.APIKey
}

// GetServerAddress returns the full server address.
func (c *Configuration) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.data.Server.BindAddress, c.data.Server.Port)
}

// GetPublicURL returns the public URL (for SSE endpoint URLs).
// If not configured, falls back to bind_address:port.
func (c *Configuration) GetPublicURL() string {
	if c.data.Server.PublicURL != "" {
		return c.data.Server.PublicURL
	}

	// Fallback to constructed URL from bind address
	protocol := "http"
	if c.data.Server.EnableTLS {
		protocol = "https"
	}

	return fmt.Sprintf("%s://%s:%d", protocol, c.data.Server.BindAddress, c.data.Server.Port)
}

// GetDatabaseConnectionString returns the PostgreSQL connection string.
func (c *Configuration) GetDatabaseConnectionString() string {
	return fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		c.data.Database.Host,
		c.data.Database.Port,
		c.data.Database.Name,
		c.data.Database.User,
		c.data.Database.Password,
		c.data.Database.SSLMode,
	)
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
