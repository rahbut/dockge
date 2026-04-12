package main

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration for the Dockge server, resolved from
// environment variables with CLI flag overrides applied on top.
type Config struct {
	Port      int
	Hostname  string
	DataDir   string
	StacksDir string

	SSLKey           string
	SSLCert          string
	SSLKeyPassphrase string

	EnableConsole bool
	IsContainer   bool

	// Development mode — enables CORS, verbose logging, bypasses origin check.
	IsDev bool

	// SQL query logging.
	SQLLog bool

	// Comma-separated "level_module" pairs to suppress, e.g. "debug_monitor".
	HideLog []string

	// Set to "bypass" to skip WebSocket origin validation.
	WSOriginCheck string
}

// LoadConfig reads configuration from environment variables.
// CLI flag overrides are applied separately via applyFlags().
func LoadConfig() *Config {
	c := &Config{
		Port:             envInt("DOCKGE_PORT", 5001),
		Hostname:         os.Getenv("DOCKGE_HOSTNAME"),
		DataDir:          envStr("DOCKGE_DATA_DIR", "./data/"),
		StacksDir:        envStr("DOCKGE_STACKS_DIR", defaultStacksDir()),
		SSLKey:           os.Getenv("DOCKGE_SSL_KEY"),
		SSLCert:          os.Getenv("DOCKGE_SSL_CERT"),
		SSLKeyPassphrase: os.Getenv("DOCKGE_SSL_KEY_PASSPHRASE"),
		EnableConsole:    os.Getenv("DOCKGE_ENABLE_CONSOLE") == "1" || strings.ToLower(os.Getenv("DOCKGE_ENABLE_CONSOLE")) == "true",
		IsContainer:      os.Getenv("DOCKGE_IS_CONTAINER") == "1",
		IsDev:            strings.ToLower(os.Getenv("NODE_ENV")) == "development",
		SQLLog:           os.Getenv("SQL_LOG") == "1",
		WSOriginCheck:    os.Getenv("UPTIME_KUMA_WS_ORIGIN_CHECK"),
	}
	if raw := os.Getenv("DOCKGE_HIDE_LOG"); raw != "" {
		c.HideLog = strings.Split(raw, ",")
	}
	return c
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func defaultStacksDir() string {
	// On Windows use a local ./stacks directory; everywhere else /opt/stacks.
	if os.PathSeparator == '\\' {
		return "./stacks"
	}
	return "/opt/stacks"
}
