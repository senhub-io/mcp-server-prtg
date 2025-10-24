package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration.
type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Log      LogConfig      `yaml:"log"`
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

// LogConfig holds logging settings.
type LogConfig struct {
	Level string `yaml:"level"`
}

// Load reads configuration from environment variables and optional YAML file.
func Load(configPath string) (*Config, error) {
	cfg := &Config{
		Database: DatabaseConfig{
			Host:     getEnv("PRTG_DB_HOST", "localhost"),
			Port:     getEnvInt("PRTG_DB_PORT", 5432),
			Name:     getEnv("PRTG_DB_NAME", "prtg_data_exporter"),
			User:     getEnv("PRTG_DB_USER", "prtg_reader"),
			Password: getEnv("PRTG_DB_PASSWORD", ""),
			SSLMode:  getEnv("PRTG_DB_SSLMODE", "disable"),
		},
		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
	}

	// Override with YAML file if provided
	if configPath != "" {
		if err := loadYAMLConfig(configPath, cfg); err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	}

	// Validate required fields
	if cfg.Database.Password == "" {
		return nil, fmt.Errorf("database password is required (set PRTG_DB_PASSWORD)")
	}

	return cfg, nil
}

// loadYAMLConfig loads configuration from a YAML file.
func loadYAMLConfig(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, cfg)
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}

// getEnvInt retrieves an integer environment variable or returns a default value.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}

	return defaultValue
}

// ConnectionString builds a PostgreSQL connection string.
func (d *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		d.Host, d.Port, d.Name, d.User, d.Password, d.SSLMode,
	)
}
