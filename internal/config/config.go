package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Config holds CLI configuration: session, API credentials, and output settings.
type Config struct {
	// API credentials.
	APIID   int    `json:"api_id"`
	APIHash string `json:"api_hash"`
	// Session string (Telethon/Pyrogram/gotg format).
	Session string `json:"session,omitempty"`
	// Bot token (alternative to session).
	BotToken string `json:"bot_token,omitempty"`
	// Phone number (alternative to session for user auth).
	Phone string `json:"phone,omitempty"`
	// DatabasePath is the path to a SQLite database file for persistent
	// session and peer storage. When set, takes priority over session strings.
	DatabasePath string `json:"database_path,omitempty"`
	// DatabaseName is the session name within the database (default: "default").
	DatabaseName string `json:"database_name,omitempty"`
	// SocketPath is the Unix domain socket for IPC between listen and invoke.
	SocketPath string `json:"socket_path,omitempty"`
	// Output format: "text" (default), "json".
	Format string `json:"format,omitempty"`
	// Enable verbose/debug output.
	Debug bool `json:"debug,omitempty"`
	// Disable color output.
	NoColor bool `json:"no_color,omitempty"`
	// ConfigPath is the loaded config file path. It is runtime metadata, not persisted.
	ConfigPath string `json:"-"`
}

// DefaultPath returns the default config file path.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".tgdev.json"
	}
	return filepath.Join(home, ".tgdev.json")
}

// DefaultDBPath returns the default SQLite database path.
func DefaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "tgdev.db"
	}
	return filepath.Join(home, ".tgdev", "tgdev.db")
}

// Load reads config from a JSON file.
func Load(path string) (*Config, error) {
	// Warn and fix overly permissive config files.
	if fi, err := os.Stat(path); err == nil {
		if fi.Mode().Perm()&0077 != 0 {
			fmt.Fprintf(os.Stderr, "warning: %s has overly permissive permissions (%04o), fixing to 0600\n", path, fi.Mode().Perm())
			os.Chmod(path, 0600)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// Save writes config to a JSON file.
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// FromFlags creates a Config from CLI flags, falling back to the config file
// for any missing values.
func FromFlags(apiID int, apiHash, session, botToken, phone, dbPath, dbName, socketPath, configPath string) (*Config, error) {
	cfg := &Config{}

	// Try loading from config file — tolerate missing file.
	if configPath != "" {
		if loaded, err := Load(configPath); err == nil {
			cfg = loaded
		}
	}

	// Environment variables fill in missing config values.
	// These avoid exposing secrets in process args (ps aux).
	if cfg.APIID == 0 {
		if v := os.Getenv("TGDEV_API_ID"); v != "" {
			if id, err := ParseAPIID(v); err == nil {
				cfg.APIID = id
			}
		}
	}
	if cfg.APIHash == "" {
		if v := os.Getenv("TGDEV_API_HASH"); v != "" {
			cfg.APIHash = v
		}
	}
	if cfg.BotToken == "" {
		if v := os.Getenv("TGDEV_BOT_TOKEN"); v != "" {
			cfg.BotToken = v
		}
	}
	if cfg.Session == "" {
		if v := os.Getenv("TGDEV_SESSION"); v != "" {
			cfg.Session = v
		}
	}
	if cfg.Phone == "" {
		if v := os.Getenv("TGDEV_PHONE"); v != "" {
			cfg.Phone = v
		}
	}

	// CLI flags override config file and env var values.
	if apiID != 0 {
		cfg.APIID = apiID
	}
	if apiHash != "" {
		cfg.APIHash = apiHash
	}
	if session != "" {
		cfg.Session = session
	}
	if botToken != "" {
		cfg.BotToken = botToken
	}
	if phone != "" {
		cfg.Phone = phone
	}
	if dbPath != "" {
		cfg.DatabasePath = dbPath
	}
	if dbName != "" {
		cfg.DatabaseName = dbName
	}
	if socketPath != "" {
		cfg.SocketPath = socketPath
	}

	return cfg, nil
}

// Validate checks that required fields are present.
func (c *Config) Validate() error {
	if c.APIID == 0 {
		return fmt.Errorf("api_id is required (use --api-id or set in config)")
	}
	if c.APIHash == "" {
		return fmt.Errorf("api_hash is required (use --api-hash or set in config)")
	}
	// At least one auth method is needed.
	if c.DatabasePath == "" && c.Session == "" && c.BotToken == "" && c.Phone == "" {
		return fmt.Errorf("one of --db, --session, --bot-token, or --phone is required")
	}
	return nil
}

// ParseAPIID parses a string API ID into an integer.
func ParseAPIID(s string) (int, error) {
	id, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid api_id: %w", err)
	}
	return id, nil
}

// UseColor returns whether color output should be used.
func (c *Config) UseColor() bool {
	return !c.NoColor
}

// HasDatabase returns true if SQLite database path is configured.
func (c *Config) HasDatabase() bool {
	return c.DatabasePath != ""
}

// DBName returns the database session name, defaulting to "default".
func (c *Config) DBName() string {
	if c.DatabaseName == "" {
		return "default"
	}
	return c.DatabaseName
}

// GetSocketPath returns the IPC socket path, defaulting to DefaultSocketPath.
func (c *Config) GetSocketPath() string {
	if c.SocketPath != "" {
		return c.SocketPath
	}
	return DefaultSocketPath()
}

// DefaultSocketPath returns the default Unix domain socket path for IPC.
func DefaultSocketPath() string {
	// Prefer XDG_RUNTIME_DIR (per-user, restricted permissions).
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "tgdev.sock")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to per-user /run directory, not /tmp.
		return filepath.Join(fmt.Sprintf("/run/user/%d", os.Getuid()), "tgdev.sock")
	}
	return filepath.Join(home, ".tgdev", "tgdev.sock")
}
