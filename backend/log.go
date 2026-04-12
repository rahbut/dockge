package main

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// initLogger configures the global zerolog logger with console output
// (colourised in development, JSON in production).
func initLogger(cfg *Config) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if cfg.IsDev {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// DOCKGE_HIDE_LOG support is best-effort at the application level since
	// zerolog uses field filtering rather than per-module suppression.
	_ = cfg.HideLog
}

// logger returns a zerolog logger scoped to a module name.
// This mirrors the original log.ts pattern of log.info("moduleName", msg).
func logger(module string) zerolog.Logger {
	return log.With().Str("module", module).Logger()
}

// suppressedModules checks whether the given level+module combo is hidden.
func suppressedModules(level, module string, hideLog []string) bool {
	key := strings.ToLower(level + "_" + module)
	for _, h := range hideLog {
		if strings.ToLower(strings.TrimSpace(h)) == key {
			return true
		}
	}
	return false
}
