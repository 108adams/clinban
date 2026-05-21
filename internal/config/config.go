package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ErrMalformedConfig is wrapped by Load when .clinban exists but cannot be
// parsed as TOML.
var ErrMalformedConfig = errors.New("config: malformed .clinban file")

// Config is the resolved filesystem layout for a Clinban project.
//
// Both paths are absolute or relative exactly as returned by Load after
// resolving .clinban values against the project root.
type Config struct {
	// TicketsDir is the directory containing active ticket files.
	TicketsDir string `toml:"tickets_dir"`
	// ArchiveDir is the directory containing archived ticket files.
	ArchiveDir string `toml:"archive_dir"`
	// DefaultType is the ticket type used when --type is not supplied.
	// Empty string means "not set"; validation is the caller's responsibility.
	DefaultType string `toml:"default_type"`
}

// Load reads .clinban from projectRoot and returns the resolved configuration.
//
// If .clinban is absent, Load returns defaults without error. If .clinban is
// present but malformed, Load returns an error wrapping ErrMalformedConfig.
// Partial configs are valid: omitted fields fall back to defaults. Relative
// paths are resolved against projectRoot.
//
// The default layout uses projectRoot/tickets for active tickets and
// projectRoot/tickets/archive for archived tickets.
func Load(projectRoot string) (*Config, error) {
	configPath := filepath.Join(projectRoot, ".clinban")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return defaults(projectRoot), nil
		}
		return nil, fmt.Errorf("config: read .clinban: %w", err)
	}

	var raw struct {
		TicketsDir  string `toml:"tickets_dir"`
		ArchiveDir  string `toml:"archive_dir"`
		DefaultType string `toml:"default_type"`
	}

	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrMalformedConfig, err.Error())
	}

	cfg := defaults(projectRoot)

	if raw.TicketsDir != "" {
		cfg.TicketsDir = absPath(projectRoot, raw.TicketsDir)
		// When tickets_dir is overridden, reset archive_dir to tickets_dir/archive
		// so that a partial config (tickets_dir only) gets a consistent default.
		cfg.ArchiveDir = filepath.Join(cfg.TicketsDir, "archive")
	}

	if raw.ArchiveDir != "" {
		cfg.ArchiveDir = absPath(projectRoot, raw.ArchiveDir)
	}

	cfg.DefaultType = raw.DefaultType

	return cfg, nil
}

// defaults returns a Config with all fields set to their default values.
func defaults(projectRoot string) *Config {
	return &Config{
		TicketsDir: filepath.Join(projectRoot, "tickets"),
		ArchiveDir: filepath.Join(projectRoot, "tickets", "archive"),
	}
}

// absPath resolves p relative to base if p is not already absolute.
func absPath(base, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(base, p)
}
