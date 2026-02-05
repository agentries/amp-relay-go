// Package config provides configuration management for the AMP Relay Server
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the AMP Relay Server
type Config struct {
	// Server configuration
	Server ServerConfig `yaml:"server" json:"server"`

	// Storage configuration
	Storage StorageConfig `yaml:"storage" json:"storage"`

	// Logging configuration
	Logging LoggingConfig `yaml:"logging" json:"logging"`

	// Security configuration
	Security SecurityConfig `yaml:"security" json:"security"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	// Address to listen on (e.g., ":8080" or "0.0.0.0:8080")
	Address string `yaml:"address" json:"address"`

	// ReadTimeout is the maximum duration for reading the entire request
	ReadTimeout time.Duration `yaml:"read_timeout" json:"read_timeout"`

	// WriteTimeout is the maximum duration before timing out writes of the response
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`

	// MaxPayloadSize is the maximum allowed message payload size in bytes
	MaxPayloadSize int64 `yaml:"max_payload_size" json:"max_payload_size"`

	// EnableWebSocket enables WebSocket transport
	EnableWebSocket bool `yaml:"enable_websocket" json:"enable_websocket"`
}

// StorageConfig holds storage-specific configuration
type StorageConfig struct {
	// Type of storage backend (memory, file, redis)
	Type string `yaml:"type" json:"type"`

	// Path to storage directory (for file-based storage)
	Path string `yaml:"path" json:"path"`

	// DefaultTTL is the default message TTL
	DefaultTTL time.Duration `yaml:"default_ttl" json:"default_ttl"`

	// MaxMessages is the maximum number of messages to store (0 = unlimited)
	MaxMessages int `yaml:"max_messages" json:"max_messages"`

	// CleanupInterval is the interval between cleanup runs
	CleanupInterval time.Duration `yaml:"cleanup_interval" json:"cleanup_interval"`
}

// LoggingConfig holds logging-specific configuration
type LoggingConfig struct {
	// Level is the log level (debug, info, warn, error)
	Level string `yaml:"level" json:"level"`

	// Format is the log format (text, json)
	Format string `yaml:"format" json:"format"`

	// Output is the log output (stdout, stderr, or file path)
	Output string `yaml:"output" json:"output"`
}

// SecurityConfig holds security-specific configuration
type SecurityConfig struct {
	// EnableAuth enables DID-based authentication
	EnableAuth bool `yaml:"enable_auth" json:"enable_auth"`

	// AllowedOrigins is a list of allowed CORS origins
	AllowedOrigins []string `yaml:"allowed_origins" json:"allowed_origins"`

	// RateLimitPerMinute is the number of requests allowed per minute per client
	RateLimitPerMinute int `yaml:"rate_limit_per_minute" json:"rate_limit_per_minute"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Address:         ":8080",
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			MaxPayloadSize:  512 * 1024, // 512KB
			EnableWebSocket: true,
		},
		Storage: StorageConfig{
			Type:            "memory",
			Path:            "./data",
			DefaultTTL:      5 * time.Minute,
			MaxMessages:     10000,
			CleanupInterval: 1 * time.Minute,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
		Security: SecurityConfig{
			EnableAuth:         false,
			AllowedOrigins:     []string{"*"},
			RateLimitPerMinute: 60,
		},
	}
}

// Load loads configuration from file and environment variables
// Environment variables take precedence over file configuration
func Load(configPath string) (*Config, error) {
	// Start with defaults
	config := DefaultConfig()

	// Load from file if provided
	if configPath != "" {
		if err := loadFromFile(config, configPath); err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	}

	// Override with environment variables
	if err := loadFromEnv(config); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// loadFromFile loads configuration from a YAML or JSON file
func loadFromFile(config *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, config); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, config); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config file format: %s (use .yaml, .yml, or .json)", ext)
	}

	return nil
}

// loadFromEnv overrides configuration with environment variables
// Environment variables use the prefix "AMP_" and follow the pattern:
// AMP_SERVER_ADDRESS, AMP_STORAGE_PATH, etc.
func loadFromEnv(config *Config) error {
	// Server configuration
	if v := os.Getenv("AMP_SERVER_ADDRESS"); v != "" {
		config.Server.Address = v
	}
	if v := os.Getenv("AMP_SERVER_READ_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			config.Server.ReadTimeout = d
		}
	}
	if v := os.Getenv("AMP_SERVER_WRITE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			config.Server.WriteTimeout = d
		}
	}
	if v := os.Getenv("AMP_SERVER_MAX_PAYLOAD_SIZE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			config.Server.MaxPayloadSize = n
		}
	}
	if v := os.Getenv("AMP_SERVER_ENABLE_WEBSOCKET"); v != "" {
		config.Server.EnableWebSocket = parseBool(v)
	}

	// Storage configuration
	if v := os.Getenv("AMP_STORAGE_TYPE"); v != "" {
		config.Storage.Type = v
	}
	if v := os.Getenv("AMP_STORAGE_PATH"); v != "" {
		config.Storage.Path = v
	}
	if v := os.Getenv("AMP_STORAGE_DEFAULT_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			config.Storage.DefaultTTL = d
		}
	}
	if v := os.Getenv("AMP_STORAGE_MAX_MESSAGES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			config.Storage.MaxMessages = n
		}
	}
	if v := os.Getenv("AMP_STORAGE_CLEANUP_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			config.Storage.CleanupInterval = d
		}
	}

	// Logging configuration
	if v := os.Getenv("AMP_LOG_LEVEL"); v != "" {
		config.Logging.Level = v
	}
	if v := os.Getenv("AMP_LOG_FORMAT"); v != "" {
		config.Logging.Format = v
	}
	if v := os.Getenv("AMP_LOG_OUTPUT"); v != "" {
		config.Logging.Output = v
	}

	// Security configuration
	if v := os.Getenv("AMP_SECURITY_ENABLE_AUTH"); v != "" {
		config.Security.EnableAuth = parseBool(v)
	}
	if v := os.Getenv("AMP_SECURITY_ALLOWED_ORIGINS"); v != "" {
		config.Security.AllowedOrigins = strings.Split(v, ",")
	}
	if v := os.Getenv("AMP_SECURITY_RATE_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			config.Security.RateLimitPerMinute = n
		}
	}

	return nil
}

// parseBool parses a string as a boolean value
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "on"
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server configuration
	if c.Server.Address == "" {
		return fmt.Errorf("server address cannot be empty")
	}
	if c.Server.MaxPayloadSize <= 0 {
		return fmt.Errorf("max payload size must be positive")
	}
	if c.Server.ReadTimeout <= 0 {
		return fmt.Errorf("read timeout must be positive")
	}
	if c.Server.WriteTimeout <= 0 {
		return fmt.Errorf("write timeout must be positive")
	}

	// Validate storage configuration
	if c.Storage.Type == "" {
		return fmt.Errorf("storage type cannot be empty")
	}
	validStorageTypes := []string{"memory", "file", "redis"}
	if !contains(validStorageTypes, c.Storage.Type) {
		return fmt.Errorf("invalid storage type: %s (must be one of: %v)", c.Storage.Type, validStorageTypes)
	}
	if c.Storage.Type == "file" && c.Storage.Path == "" {
		return fmt.Errorf("storage path cannot be empty when using file storage")
	}
	if c.Storage.DefaultTTL <= 0 {
		return fmt.Errorf("default TTL must be positive")
	}

	// Validate logging configuration
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLogLevels, strings.ToLower(c.Logging.Level)) {
		return fmt.Errorf("invalid log level: %s (must be one of: %v)", c.Logging.Level, validLogLevels)
	}
	validLogFormats := []string{"text", "json"}
	if !contains(validLogFormats, strings.ToLower(c.Logging.Format)) {
		return fmt.Errorf("invalid log format: %s (must be one of: %v)", c.Logging.Format, validLogFormats)
	}

	// Validate security configuration
	if c.Security.RateLimitPerMinute < 0 {
		return fmt.Errorf("rate limit cannot be negative")
	}

	return nil
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	item = strings.ToLower(item)
	for _, s := range slice {
		if strings.ToLower(s) == item {
			return true
		}
	}
	return false
}

// SaveToFile saves the current configuration to a file
func (c *Config) SaveToFile(path string) error {
	ext := strings.ToLower(filepath.Ext(path))

	var data []byte
	var err error

	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(c)
	case ".json":
		data, err = json.MarshalIndent(c, "", "  ")
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetStoragePath returns the absolute storage path
func (c *Config) GetStoragePath() string {
	if filepath.IsAbs(c.Storage.Path) {
		return c.Storage.Path
	}
	// Convert to absolute path
	abs, _ := filepath.Abs(c.Storage.Path)
	return abs
}

// IsDebug returns true if log level is debug
func (c *Config) IsDebug() bool {
	return strings.ToLower(c.Logging.Level) == "debug"
}
