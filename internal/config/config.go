package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ErrMalformedConfig is returned when the .clinban file exists but cannot be parsed as valid TOML.
var ErrMalformedConfig = errors.New("config: malformed .clinban file")

// Config holds the resolved configuration for a Clinban project.
type Config struct {
	TicketsDir string `toml:"tickets_dir"`
	ArchiveDir string `toml:"archive_dir"`
}

// Load reads .clinban from projectRoot.
// If the file is absent, returns defaults silently (no error).
// If the file exists but is malformed TOML, returns a non-nil error.
// Partial configs are valid; unset fields fall back to defaults.
// Defaults: TicketsDir = projectRoot, ArchiveDir = projectRoot/archive.
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
		TicketsDir string `toml:"tickets_dir"`
		ArchiveDir string `toml:"archive_dir"`
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

	return cfg, nil
}

// defaults returns a Config with all fields set to their default values.
func defaults(projectRoot string) *Config {
	return &Config{
		TicketsDir: projectRoot,
		ArchiveDir: filepath.Join(projectRoot, "archive"),
	}
}

// absPath resolves p relative to base if p is not already absolute.
func absPath(base, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(base, p)
}
