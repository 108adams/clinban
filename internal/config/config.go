package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// validDefaultTypes lists the accepted values for the default_type key.
var validDefaultTypes = map[string]bool{
	"bug":     true,
	"task":    true,
	"feature": true,
	"spike":   true,
}

// ErrUnknownKey is returned by SetKey when the supplied key is not a
// recognised configuration field.
var ErrUnknownKey = errors.New("config: unknown key")

// ErrInvalidValue is returned by SetKey when the supplied value is not valid
// for the given key.
var ErrInvalidValue = errors.New("config: invalid value")

// Entry describes one configuration key with its resolved and default values.
type Entry struct {
	// Key is the TOML field name, e.g. "tickets_dir".
	Key string
	// Value is the currently active value (default when not explicitly set).
	Value string
	// Default is the built-in default value. Empty string means no default.
	Default string
	// IsSet is true when the key appears in .clinban.
	IsSet bool
}

// Entries returns all known configuration keys with their resolved values,
// defaults, and whether each was explicitly set in .clinban.
//
// If .clinban is absent, all entries report IsSet=false with defaults. If
// .clinban is present but malformed, an error wrapping ErrMalformedConfig is
// returned.
func Entries(root string) ([]Entry, error) {
	configPath := filepath.Join(root, ".clinban")

	// rawSet tracks which keys were explicitly present in .clinban.
	// SplitRawNew uses *bool to distinguish absent (nil) from explicit false.
	type rawConfig struct {
		TicketsDir  string `toml:"tickets_dir"`
		ArchiveDir  string `toml:"archive_dir"`
		DefaultType string `toml:"default_type"`
		SplitRawNew *bool  `toml:"split_raw_new"`
	}

	var raw rawConfig
	ticketsDirSet := false
	archiveDirSet := false
	defaultTypeSet := false
	splitRawNewSet := false

	data, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("config: read .clinban: %w", err)
	}

	if err == nil {
		// File exists — parse it.
		if parseErr := toml.Unmarshal(data, &raw); parseErr != nil {
			return nil, fmt.Errorf("%w: %s", ErrMalformedConfig, parseErr.Error())
		}
		ticketsDirSet = raw.TicketsDir != ""
		archiveDirSet = raw.ArchiveDir != ""
		defaultTypeSet = raw.DefaultType != ""
		splitRawNewSet = raw.SplitRawNew != nil
	}

	// Built-in defaults (relative, as displayed to the user).
	const defaultTicketsDir = "tickets"
	const defaultArchiveDir = "tickets/archive"

	// Value to display: use the raw (relative) value when set, otherwise the
	// default string. We show relative defaults, not absolute resolved paths.
	ticketsDirValue := defaultTicketsDir
	if ticketsDirSet {
		ticketsDirValue = raw.TicketsDir
	}

	archiveDirValue := defaultArchiveDir
	if archiveDirSet {
		archiveDirValue = raw.ArchiveDir
	} else if ticketsDirSet {
		// When tickets_dir was overridden without archive_dir, the effective
		// displayed default is tickets_dir/archive.
		archiveDirValue = raw.TicketsDir + "/archive"
	}

	defaultTypeValue := ""
	if defaultTypeSet {
		defaultTypeValue = raw.DefaultType
	}

	splitRawNewValue := "true" // built-in default
	if splitRawNewSet {
		splitRawNewValue = fmt.Sprintf("%v", *raw.SplitRawNew)
	}

	return []Entry{
		{
			Key:     "tickets_dir",
			Value:   ticketsDirValue,
			Default: defaultTicketsDir,
			IsSet:   ticketsDirSet,
		},
		{
			Key:     "archive_dir",
			Value:   archiveDirValue,
			Default: defaultArchiveDir,
			IsSet:   archiveDirSet,
		},
		{
			Key:     "default_type",
			Value:   defaultTypeValue,
			Default: "", // no built-in default
			IsSet:   defaultTypeSet,
		},
		{
			Key:     "split_raw_new",
			Value:   splitRawNewValue,
			Default: "true",
			IsSet:   splitRawNewSet,
		},
	}, nil
}

// SetKey validates key and value, then writes or updates the key in .clinban.
//
// If .clinban is absent it is created. The write is atomic: a temporary file
// is written and renamed into place. Returns ErrUnknownKey for unknown keys
// and ErrInvalidValue for invalid values.
func SetKey(root, key, value string) error {
	// Validate key.
	switch key {
	case "tickets_dir", "archive_dir":
		if value == "" {
			return fmt.Errorf("%w: %s cannot be empty", ErrInvalidValue, key)
		}
	case "default_type":
		// Empty string is allowed (unsets the value).
		if value != "" && !validDefaultTypes[value] {
			return fmt.Errorf("%w: default_type must be one of bug, task, feature, spike; got %q", ErrInvalidValue, value)
		}
	case "split_raw_new":
		if value != "true" && value != "false" {
			return fmt.Errorf("%w: split_raw_new must be \"true\" or \"false\"; got %q", ErrInvalidValue, value)
		}
	default:
		return fmt.Errorf("%w: %q", ErrUnknownKey, key)
	}

	configPath := filepath.Join(root, ".clinban")

	// Read existing raw config, or start with zero value.
	// SplitRawNew uses *bool to distinguish absent (nil) from explicit false.
	type rawConfig struct {
		TicketsDir  string `toml:"tickets_dir"`
		ArchiveDir  string `toml:"archive_dir"`
		DefaultType string `toml:"default_type"`
		SplitRawNew *bool  `toml:"split_raw_new"`
	}

	var raw rawConfig

	data, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("config: read .clinban: %w", err)
	}
	if err == nil {
		if parseErr := toml.Unmarshal(data, &raw); parseErr != nil {
			return fmt.Errorf("%w: %s", ErrMalformedConfig, parseErr.Error())
		}
	}

	// Update the relevant field.
	switch key {
	case "tickets_dir":
		raw.TicketsDir = value
	case "archive_dir":
		raw.ArchiveDir = value
	case "default_type":
		raw.DefaultType = value
	case "split_raw_new":
		b := value == "true"
		raw.SplitRawNew = &b
	}

	// Marshal back to TOML.
	var buf []byte
	if raw.TicketsDir != "" {
		buf = append(buf, []byte(fmt.Sprintf("tickets_dir = %q\n", raw.TicketsDir))...)
	}
	if raw.ArchiveDir != "" {
		buf = append(buf, []byte(fmt.Sprintf("archive_dir = %q\n", raw.ArchiveDir))...)
	}
	if raw.DefaultType != "" {
		buf = append(buf, []byte(fmt.Sprintf("default_type = %q\n", raw.DefaultType))...)
	}
	if raw.SplitRawNew != nil {
		buf = append(buf, []byte(fmt.Sprintf("split_raw_new = %v\n", *raw.SplitRawNew))...)
	}

	// Atomic write: temp file in same directory then rename.
	tmp, err := os.CreateTemp(root, ".clinban-*.tmp")
	if err != nil {
		return fmt.Errorf("config: set key: create temp: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(buf); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("config: set key: write temp: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("config: set key: chmod temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("config: set key: sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("config: set key: close temp: %w", err)
	}
	if err := os.Rename(tmpPath, configPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("config: set key: rename: %w", err)
	}

	return nil
}

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
	// SplitRawNew controls whether the `new` command splits the joined
	// positional-args string on the first `#` to pre-fill the ticket title.
	// Default is true when the key is absent from .clinban.
	SplitRawNew bool `toml:"split_raw_new"`
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

	// SplitRawNew uses *bool so that nil (absent) is distinguishable from
	// explicit false — TOML bool zero-value is false, which would conflict
	// with the default-true requirement.
	var raw struct {
		TicketsDir  string `toml:"tickets_dir"`
		ArchiveDir  string `toml:"archive_dir"`
		DefaultType string `toml:"default_type"`
		SplitRawNew *bool  `toml:"split_raw_new"`
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

	if raw.SplitRawNew != nil {
		cfg.SplitRawNew = *raw.SplitRawNew
	}
	// When nil (absent), cfg.SplitRawNew retains the default set by defaults().

	return cfg, nil
}

// defaults returns a Config with all fields set to their default values.
func defaults(projectRoot string) *Config {
	return &Config{
		TicketsDir:  filepath.Join(projectRoot, "tickets"),
		ArchiveDir:  filepath.Join(projectRoot, "tickets", "archive"),
		SplitRawNew: true,
	}
}

// absPath resolves p relative to base if p is not already absolute.
func absPath(base, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(base, p)
}
