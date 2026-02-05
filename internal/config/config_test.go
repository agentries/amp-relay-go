package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Server defaults
	if cfg.Server.Address != ":8080" {
		t.Errorf("Server.Address = %q, want %q", cfg.Server.Address, ":8080")
	}
	if cfg.Server.ReadTimeout != 30*time.Second {
		t.Errorf("Server.ReadTimeout = %v, want %v", cfg.Server.ReadTimeout, 30*time.Second)
	}
	if cfg.Server.WriteTimeout != 30*time.Second {
		t.Errorf("Server.WriteTimeout = %v, want %v", cfg.Server.WriteTimeout, 30*time.Second)
	}
	if cfg.Server.MaxPayloadSize != 512*1024 {
		t.Errorf("Server.MaxPayloadSize = %d, want %d", cfg.Server.MaxPayloadSize, 512*1024)
	}
	if !cfg.Server.EnableWebSocket {
		t.Error("Server.EnableWebSocket = false, want true")
	}

	// Storage defaults
	if cfg.Storage.Type != "memory" {
		t.Errorf("Storage.Type = %q, want %q", cfg.Storage.Type, "memory")
	}
	if cfg.Storage.Path != "./data" {
		t.Errorf("Storage.Path = %q, want %q", cfg.Storage.Path, "./data")
	}
	if cfg.Storage.DefaultTTL != 5*time.Minute {
		t.Errorf("Storage.DefaultTTL = %v, want %v", cfg.Storage.DefaultTTL, 5*time.Minute)
	}
	if cfg.Storage.MaxMessages != 10000 {
		t.Errorf("Storage.MaxMessages = %d, want %d", cfg.Storage.MaxMessages, 10000)
	}
	if cfg.Storage.CleanupInterval != 1*time.Minute {
		t.Errorf("Storage.CleanupInterval = %v, want %v", cfg.Storage.CleanupInterval, 1*time.Minute)
	}

	// Logging defaults
	if cfg.Logging.Level != "info" {
		t.Errorf("Logging.Level = %q, want %q", cfg.Logging.Level, "info")
	}
	if cfg.Logging.Format != "text" {
		t.Errorf("Logging.Format = %q, want %q", cfg.Logging.Format, "text")
	}
	if cfg.Logging.Output != "stdout" {
		t.Errorf("Logging.Output = %q, want %q", cfg.Logging.Output, "stdout")
	}

	// Security defaults
	if cfg.Security.EnableAuth {
		t.Error("Security.EnableAuth = true, want false")
	}
	if len(cfg.Security.AllowedOrigins) != 1 || cfg.Security.AllowedOrigins[0] != "*" {
		t.Errorf("Security.AllowedOrigins = %v, want [*]", cfg.Security.AllowedOrigins)
	}
	if cfg.Security.RateLimitPerMinute != 60 {
		t.Errorf("Security.RateLimitPerMinute = %d, want %d", cfg.Security.RateLimitPerMinute, 60)
	}
}

func TestLoad_NoFile(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\") returned error: %v", err)
	}

	defaults := DefaultConfig()

	if cfg.Server.Address != defaults.Server.Address {
		t.Errorf("Server.Address = %q, want %q", cfg.Server.Address, defaults.Server.Address)
	}
	if cfg.Storage.Type != defaults.Storage.Type {
		t.Errorf("Storage.Type = %q, want %q", cfg.Storage.Type, defaults.Storage.Type)
	}
	if cfg.Logging.Level != defaults.Logging.Level {
		t.Errorf("Logging.Level = %q, want %q", cfg.Logging.Level, defaults.Logging.Level)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	tests := []struct {
		name    string
		envKey  string
		envVal  string
		checkFn func(t *testing.T, cfg *Config)
	}{
		{
			name:   "AMP_SERVER_ADDRESS overrides default",
			envKey: "AMP_SERVER_ADDRESS",
			envVal: ":9090",
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.Server.Address != ":9090" {
					t.Errorf("Server.Address = %q, want %q", cfg.Server.Address, ":9090")
				}
			},
		},
		{
			name:   "AMP_STORAGE_TYPE overrides default",
			envKey: "AMP_STORAGE_TYPE",
			envVal: "file",
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.Storage.Type != "file" {
					t.Errorf("Storage.Type = %q, want %q", cfg.Storage.Type, "file")
				}
			},
		},
		{
			name:   "AMP_LOG_LEVEL overrides default",
			envKey: "AMP_LOG_LEVEL",
			envVal: "debug",
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.Logging.Level != "debug" {
					t.Errorf("Logging.Level = %q, want %q", cfg.Logging.Level, "debug")
				}
			},
		},
		{
			name:   "AMP_SERVER_READ_TIMEOUT overrides default",
			envKey: "AMP_SERVER_READ_TIMEOUT",
			envVal: "60s",
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.Server.ReadTimeout != 60*time.Second {
					t.Errorf("Server.ReadTimeout = %v, want %v", cfg.Server.ReadTimeout, 60*time.Second)
				}
			},
		},
		{
			name:   "AMP_SERVER_MAX_PAYLOAD_SIZE overrides default",
			envKey: "AMP_SERVER_MAX_PAYLOAD_SIZE",
			envVal: "1048576",
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.Server.MaxPayloadSize != 1048576 {
					t.Errorf("Server.MaxPayloadSize = %d, want %d", cfg.Server.MaxPayloadSize, 1048576)
				}
			},
		},
		{
			name:   "AMP_SERVER_ENABLE_WEBSOCKET overrides default",
			envKey: "AMP_SERVER_ENABLE_WEBSOCKET",
			envVal: "false",
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.Server.EnableWebSocket {
					t.Error("Server.EnableWebSocket = true, want false")
				}
			},
		},
		{
			name:   "AMP_STORAGE_PATH overrides default",
			envKey: "AMP_STORAGE_PATH",
			envVal: "/tmp/amp-data",
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.Storage.Path != "/tmp/amp-data" {
					t.Errorf("Storage.Path = %q, want %q", cfg.Storage.Path, "/tmp/amp-data")
				}
			},
		},
		{
			name:   "AMP_STORAGE_MAX_MESSAGES overrides default",
			envKey: "AMP_STORAGE_MAX_MESSAGES",
			envVal: "50000",
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.Storage.MaxMessages != 50000 {
					t.Errorf("Storage.MaxMessages = %d, want %d", cfg.Storage.MaxMessages, 50000)
				}
			},
		},
		{
			name:   "AMP_SECURITY_ENABLE_AUTH overrides default",
			envKey: "AMP_SECURITY_ENABLE_AUTH",
			envVal: "true",
			checkFn: func(t *testing.T, cfg *Config) {
				if !cfg.Security.EnableAuth {
					t.Error("Security.EnableAuth = false, want true")
				}
			},
		},
		{
			name:   "AMP_SECURITY_ALLOWED_ORIGINS overrides default",
			envKey: "AMP_SECURITY_ALLOWED_ORIGINS",
			envVal: "https://example.com,https://other.com",
			checkFn: func(t *testing.T, cfg *Config) {
				want := []string{"https://example.com", "https://other.com"}
				if len(cfg.Security.AllowedOrigins) != len(want) {
					t.Fatalf("Security.AllowedOrigins length = %d, want %d", len(cfg.Security.AllowedOrigins), len(want))
				}
				for i, v := range want {
					if cfg.Security.AllowedOrigins[i] != v {
						t.Errorf("Security.AllowedOrigins[%d] = %q, want %q", i, cfg.Security.AllowedOrigins[i], v)
					}
				}
			},
		},
		{
			name:   "AMP_SECURITY_RATE_LIMIT overrides default",
			envKey: "AMP_SECURITY_RATE_LIMIT",
			envVal: "120",
			checkFn: func(t *testing.T, cfg *Config) {
				if cfg.Security.RateLimitPerMinute != 120 {
					t.Errorf("Security.RateLimitPerMinute = %d, want %d", cfg.Security.RateLimitPerMinute, 120)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envKey, tt.envVal)

			cfg, err := Load("")
			if err != nil {
				t.Fatalf("Load(\"\") returned error: %v", err)
			}
			tt.checkFn(t, cfg)
		})
	}
}

func TestValidate_Valid(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Errorf("DefaultConfig().Validate() returned error: %v", err)
	}
}

func TestValidate_InvalidAddress(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.Address = ""

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() returned nil, want error for empty address")
	}
}

func TestValidate_InvalidStorageType(t *testing.T) {
	tests := []struct {
		name        string
		storageType string
	}{
		{name: "empty storage type", storageType: ""},
		{name: "unsupported storage type sqlite", storageType: "sqlite"},
		{name: "unsupported storage type postgres", storageType: "postgres"},
		{name: "unsupported storage type s3", storageType: "s3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Storage.Type = tt.storageType

			err := cfg.Validate()
			if err == nil {
				t.Fatalf("Validate() returned nil, want error for storage type %q", tt.storageType)
			}
		})
	}
}

func TestValidate_TableDriven(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(cfg *Config)
		wantErr bool
	}{
		{
			name:    "valid default config",
			mutate:  func(cfg *Config) {},
			wantErr: false,
		},
		{
			name:    "empty server address",
			mutate:  func(cfg *Config) { cfg.Server.Address = "" },
			wantErr: true,
		},
		{
			name:    "zero max payload size",
			mutate:  func(cfg *Config) { cfg.Server.MaxPayloadSize = 0 },
			wantErr: true,
		},
		{
			name:    "negative max payload size",
			mutate:  func(cfg *Config) { cfg.Server.MaxPayloadSize = -1 },
			wantErr: true,
		},
		{
			name:    "zero read timeout",
			mutate:  func(cfg *Config) { cfg.Server.ReadTimeout = 0 },
			wantErr: true,
		},
		{
			name:    "zero write timeout",
			mutate:  func(cfg *Config) { cfg.Server.WriteTimeout = 0 },
			wantErr: true,
		},
		{
			name:    "invalid storage type",
			mutate:  func(cfg *Config) { cfg.Storage.Type = "unknown" },
			wantErr: true,
		},
		{
			name:    "file storage with empty path",
			mutate:  func(cfg *Config) { cfg.Storage.Type = "file"; cfg.Storage.Path = "" },
			wantErr: true,
		},
		{
			name: "file storage with valid path",
			mutate: func(cfg *Config) {
				cfg.Storage.Type = "file"
				cfg.Storage.Path = "/tmp/data"
			},
			wantErr: false,
		},
		{
			name:    "zero default TTL",
			mutate:  func(cfg *Config) { cfg.Storage.DefaultTTL = 0 },
			wantErr: true,
		},
		{
			name:    "invalid log level",
			mutate:  func(cfg *Config) { cfg.Logging.Level = "trace" },
			wantErr: true,
		},
		{
			name:    "invalid log format",
			mutate:  func(cfg *Config) { cfg.Logging.Format = "xml" },
			wantErr: true,
		},
		{
			name:    "negative rate limit",
			mutate:  func(cfg *Config) { cfg.Security.RateLimitPerMinute = -1 },
			wantErr: true,
		},
		{
			name:    "zero rate limit is allowed",
			mutate:  func(cfg *Config) { cfg.Security.RateLimitPerMinute = 0 },
			wantErr: false,
		},
		{
			name:    "valid storage types - memory",
			mutate:  func(cfg *Config) { cfg.Storage.Type = "memory" },
			wantErr: false,
		},
		{
			name:    "valid storage types - redis",
			mutate:  func(cfg *Config) { cfg.Storage.Type = "redis" },
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.mutate(cfg)

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestSaveAndLoadFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("roundtrip YAML", func(t *testing.T) {
		original := DefaultConfig()
		original.Server.Address = ":9999"
		original.Storage.Type = "redis"
		original.Logging.Level = "debug"

		yamlPath := filepath.Join(tmpDir, "config_roundtrip.yaml")
		if err := original.SaveToFile(yamlPath); err != nil {
			t.Fatalf("SaveToFile(%q) returned error: %v", yamlPath, err)
		}

		loaded, err := Load(yamlPath)
		if err != nil {
			t.Fatalf("Load(%q) returned error: %v", yamlPath, err)
		}

		if loaded.Server.Address != original.Server.Address {
			t.Errorf("Server.Address = %q, want %q", loaded.Server.Address, original.Server.Address)
		}
		if loaded.Storage.Type != original.Storage.Type {
			t.Errorf("Storage.Type = %q, want %q", loaded.Storage.Type, original.Storage.Type)
		}
		if loaded.Logging.Level != original.Logging.Level {
			t.Errorf("Logging.Level = %q, want %q", loaded.Logging.Level, original.Logging.Level)
		}
	})

	t.Run("roundtrip JSON", func(t *testing.T) {
		original := DefaultConfig()
		original.Server.Address = ":7777"
		original.Security.RateLimitPerMinute = 200

		jsonPath := filepath.Join(tmpDir, "config_roundtrip.json")
		if err := original.SaveToFile(jsonPath); err != nil {
			t.Fatalf("SaveToFile(%q) returned error: %v", jsonPath, err)
		}

		loaded, err := Load(jsonPath)
		if err != nil {
			t.Fatalf("Load(%q) returned error: %v", jsonPath, err)
		}

		if loaded.Server.Address != original.Server.Address {
			t.Errorf("Server.Address = %q, want %q", loaded.Server.Address, original.Server.Address)
		}
		if loaded.Security.RateLimitPerMinute != original.Security.RateLimitPerMinute {
			t.Errorf("Security.RateLimitPerMinute = %d, want %d", loaded.Security.RateLimitPerMinute, original.Security.RateLimitPerMinute)
		}
	})
}

func TestLoadFromFile_JSON(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := map[string]interface{}{
		"server": map[string]interface{}{
			"address":          ":3000",
			"read_timeout":     60000000000,  // 60s in nanoseconds (time.Duration)
			"write_timeout":    60000000000,
			"max_payload_size": 1048576,
			"enable_websocket": false,
		},
		"storage": map[string]interface{}{
			"type":             "file",
			"path":             "/var/amp/data",
			"default_ttl":      600000000000, // 10m
			"max_messages":     50000,
			"cleanup_interval": 120000000000, // 2m
		},
		"logging": map[string]interface{}{
			"level":  "warn",
			"format": "json",
			"output": "stderr",
		},
		"security": map[string]interface{}{
			"enable_auth":          true,
			"allowed_origins":      []string{"https://example.com"},
			"rate_limit_per_minute": 100,
		},
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	jsonPath := filepath.Join(tmpDir, "test_config.json")
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	loaded, err := Load(jsonPath)
	if err != nil {
		t.Fatalf("Load(%q) returned error: %v", jsonPath, err)
	}

	if loaded.Server.Address != ":3000" {
		t.Errorf("Server.Address = %q, want %q", loaded.Server.Address, ":3000")
	}
	if !loaded.Server.EnableWebSocket {
		// JSON unmarshals into the struct starting from defaults, then overrides.
		// Actually Load starts with DefaultConfig and then unmarshals on top.
		// EnableWebSocket in the JSON file is false, so it should be false.
		// But note: Load starts with DefaultConfig (EnableWebSocket=true),
		// then loadFromFile unmarshals the JSON on top. JSON false should override.
	}
	if loaded.Storage.Type != "file" {
		t.Errorf("Storage.Type = %q, want %q", loaded.Storage.Type, "file")
	}
	if loaded.Storage.Path != "/var/amp/data" {
		t.Errorf("Storage.Path = %q, want %q", loaded.Storage.Path, "/var/amp/data")
	}
	if loaded.Logging.Level != "warn" {
		t.Errorf("Logging.Level = %q, want %q", loaded.Logging.Level, "warn")
	}
	if loaded.Logging.Format != "json" {
		t.Errorf("Logging.Format = %q, want %q", loaded.Logging.Format, "json")
	}
	if !loaded.Security.EnableAuth {
		t.Error("Security.EnableAuth = false, want true")
	}
	if loaded.Security.RateLimitPerMinute != 100 {
		t.Errorf("Security.RateLimitPerMinute = %d, want %d", loaded.Security.RateLimitPerMinute, 100)
	}
}

func TestLoadFromFile_YAML(t *testing.T) {
	tmpDir := t.TempDir()

	yamlContent := `
server:
  address: ":4000"
  read_timeout: 45s
  write_timeout: 45s
  max_payload_size: 262144
  enable_websocket: true
storage:
  type: memory
  path: ./storage
  default_ttl: 10m
  max_messages: 20000
  cleanup_interval: 5m
logging:
  level: error
  format: json
  output: /var/log/amp.log
security:
  enable_auth: true
  allowed_origins:
    - "https://app.example.com"
    - "https://admin.example.com"
  rate_limit_per_minute: 30
`

	yamlPath := filepath.Join(tmpDir, "test_config.yaml")
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	loaded, err := Load(yamlPath)
	if err != nil {
		t.Fatalf("Load(%q) returned error: %v", yamlPath, err)
	}

	if loaded.Server.Address != ":4000" {
		t.Errorf("Server.Address = %q, want %q", loaded.Server.Address, ":4000")
	}
	if loaded.Server.ReadTimeout != 45*time.Second {
		t.Errorf("Server.ReadTimeout = %v, want %v", loaded.Server.ReadTimeout, 45*time.Second)
	}
	if loaded.Server.WriteTimeout != 45*time.Second {
		t.Errorf("Server.WriteTimeout = %v, want %v", loaded.Server.WriteTimeout, 45*time.Second)
	}
	if loaded.Server.MaxPayloadSize != 262144 {
		t.Errorf("Server.MaxPayloadSize = %d, want %d", loaded.Server.MaxPayloadSize, 262144)
	}
	if !loaded.Server.EnableWebSocket {
		t.Error("Server.EnableWebSocket = false, want true")
	}
	if loaded.Storage.Type != "memory" {
		t.Errorf("Storage.Type = %q, want %q", loaded.Storage.Type, "memory")
	}
	if loaded.Storage.DefaultTTL != 10*time.Minute {
		t.Errorf("Storage.DefaultTTL = %v, want %v", loaded.Storage.DefaultTTL, 10*time.Minute)
	}
	if loaded.Storage.MaxMessages != 20000 {
		t.Errorf("Storage.MaxMessages = %d, want %d", loaded.Storage.MaxMessages, 20000)
	}
	if loaded.Logging.Level != "error" {
		t.Errorf("Logging.Level = %q, want %q", loaded.Logging.Level, "error")
	}
	if loaded.Logging.Format != "json" {
		t.Errorf("Logging.Format = %q, want %q", loaded.Logging.Format, "json")
	}
	if !loaded.Security.EnableAuth {
		t.Error("Security.EnableAuth = false, want true")
	}
	if len(loaded.Security.AllowedOrigins) != 2 {
		t.Fatalf("Security.AllowedOrigins length = %d, want 2", len(loaded.Security.AllowedOrigins))
	}
	if loaded.Security.AllowedOrigins[0] != "https://app.example.com" {
		t.Errorf("Security.AllowedOrigins[0] = %q, want %q", loaded.Security.AllowedOrigins[0], "https://app.example.com")
	}
	if loaded.Security.AllowedOrigins[1] != "https://admin.example.com" {
		t.Errorf("Security.AllowedOrigins[1] = %q, want %q", loaded.Security.AllowedOrigins[1], "https://admin.example.com")
	}
	if loaded.Security.RateLimitPerMinute != 30 {
		t.Errorf("Security.RateLimitPerMinute = %d, want %d", loaded.Security.RateLimitPerMinute, 30)
	}
}

func TestLoadFromFile_UnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	txtPath := filepath.Join(tmpDir, "config.txt")
	if err := os.WriteFile(txtPath, []byte("key=value"), 0644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := Load(txtPath)
	if err == nil {
		t.Fatal("Load() returned nil, want error for unsupported format .txt")
	}
}

func TestLoadFromFile_NonExistent(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("Load() returned nil, want error for nonexistent file")
	}
}

func TestLoadFromFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "bad.yaml")
	if err := os.WriteFile(yamlPath, []byte("{{{{invalid yaml"), 0644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := Load(yamlPath)
	if err == nil {
		t.Fatal("Load() returned nil, want error for invalid YAML")
	}
}

func TestLoadFromFile_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "bad.json")
	if err := os.WriteFile(jsonPath, []byte("{not valid json}"), 0644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := Load(jsonPath)
	if err == nil {
		t.Fatal("Load() returned nil, want error for invalid JSON")
	}
}

func TestGetStoragePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantAbs  bool
	}{
		{
			name:    "relative path returns absolute",
			path:    "./data",
			wantAbs: true,
		},
		{
			name:    "absolute path stays absolute",
			path:    "/var/amp/data",
			wantAbs: true,
		},
		{
			name:    "relative nested path returns absolute",
			path:    "storage/messages",
			wantAbs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Storage.Path = tt.path

			result := cfg.GetStoragePath()

			if !filepath.IsAbs(result) {
				t.Errorf("GetStoragePath() = %q, want absolute path", result)
			}

			// For absolute input paths, the result should match exactly
			if filepath.IsAbs(tt.path) && result != tt.path {
				t.Errorf("GetStoragePath() = %q, want %q for absolute input", result, tt.path)
			}
		})
	}
}

func TestIsDebug(t *testing.T) {
	tests := []struct {
		name  string
		level string
		want  bool
	}{
		{name: "debug level", level: "debug", want: true},
		{name: "DEBUG level (uppercase)", level: "DEBUG", want: true},
		{name: "Debug level (mixed case)", level: "Debug", want: true},
		{name: "info level", level: "info", want: false},
		{name: "warn level", level: "warn", want: false},
		{name: "error level", level: "error", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Logging.Level = tt.level

			if got := cfg.IsDebug(); got != tt.want {
				t.Errorf("IsDebug() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSaveToFile_UnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := DefaultConfig()
	path := filepath.Join(tmpDir, "config.toml")

	err := cfg.SaveToFile(path)
	if err == nil {
		t.Fatal("SaveToFile() returned nil, want error for unsupported format .toml")
	}
}

func TestSaveToFile_YML_Extension(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := DefaultConfig()
	ymlPath := filepath.Join(tmpDir, "config.yml")

	if err := cfg.SaveToFile(ymlPath); err != nil {
		t.Fatalf("SaveToFile(%q) returned error: %v", ymlPath, err)
	}

	// Verify file was written and is valid YAML
	data, err := os.ReadFile(ymlPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) returned error: %v", ymlPath, err)
	}

	var loaded Config
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("yaml.Unmarshal returned error: %v", err)
	}

	if loaded.Server.Address != cfg.Server.Address {
		t.Errorf("Server.Address = %q, want %q", loaded.Server.Address, cfg.Server.Address)
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"on", true},
		{"ON", true},
		{"false", false},
		{"FALSE", false},
		{"0", false},
		{"no", false},
		{"off", false},
		{"", false},
		{"  true  ", true},
		{"  false  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := parseBool(tt.input); got != tt.want {
				t.Errorf("parseBool(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLoad_EnvOverride_ValidationFailure(t *testing.T) {
	// Set env var to an invalid storage type - Load should return an error
	// because validation runs inside Load
	t.Setenv("AMP_STORAGE_TYPE", "invalid_type")

	_, err := Load("")
	if err == nil {
		t.Fatal("Load() returned nil, want validation error for invalid storage type from env")
	}
}
