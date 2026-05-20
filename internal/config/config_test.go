package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/108adams/clinban/internal/config"
)

const (
	testCustomTicketsDir = "tasks"
	testCustomArchiveDir = "tasks/archive"
)

// writeConfigFile writes content to a .clinban file inside dir.
func writeConfigFile(t *testing.T, dir, content string) {
	t.Helper()

	path := filepath.Join(dir, ".clinban")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writeConfigFile: %v", err)
	}
}

// TestLoad_AbsentFile checks that Load returns correct defaults when .clinban is absent.
func TestLoad_AbsentFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("expected nil error for absent config file, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil Config, got nil")
	}
	wantTicketsDir := filepath.Join(dir, "tickets")
	if cfg.TicketsDir != wantTicketsDir {
		t.Errorf("TicketsDir: got %q, want %q", cfg.TicketsDir, wantTicketsDir)
	}

	wantArchive := filepath.Join(dir, "tickets", "archive")
	if cfg.ArchiveDir != wantArchive {
		t.Errorf("ArchiveDir: got %q, want %q", cfg.ArchiveDir, wantArchive)
	}
}

// TestLoad_MalformedTOML checks that Load returns an error for a malformed .clinban file.
func TestLoad_MalformedTOML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfigFile(t, dir, "this is not valid toml = [ unclosed bracket")

	cfg, err := config.Load(dir)
	if err == nil {
		t.Fatal("expected non-nil error for malformed TOML, got nil")
	}
	if cfg != nil {
		t.Errorf("expected nil Config on error, got %+v", cfg)
	}
}

// TestLoad_ValidTOML_FullConfig checks that both fields are set when both are specified.
func TestLoad_ValidTOML_FullConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfigFile(t, dir, `tickets_dir = "tasks"
archive_dir = "tasks/archive"
`)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantTicketsDir := filepath.Join(dir, testCustomTicketsDir)
	wantArchiveDir := filepath.Join(dir, testCustomArchiveDir)

	if cfg.TicketsDir != wantTicketsDir {
		t.Errorf("TicketsDir: got %q, want %q", cfg.TicketsDir, wantTicketsDir)
	}
	if cfg.ArchiveDir != wantArchiveDir {
		t.Errorf("ArchiveDir: got %q, want %q", cfg.ArchiveDir, wantArchiveDir)
	}
}

// TestLoad_PartialConfig_TicketsDirOnly checks that archive_dir defaults to tickets_dir/archive
// when only tickets_dir is specified.
func TestLoad_PartialConfig_TicketsDirOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfigFile(t, dir, `tickets_dir = "tasks"
`)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantTicketsDir := filepath.Join(dir, testCustomTicketsDir)
	wantArchiveDir := filepath.Join(wantTicketsDir, "archive")

	if cfg.TicketsDir != wantTicketsDir {
		t.Errorf("TicketsDir: got %q, want %q", cfg.TicketsDir, wantTicketsDir)
	}
	if cfg.ArchiveDir != wantArchiveDir {
		t.Errorf("ArchiveDir: got %q, want %q", cfg.ArchiveDir, wantArchiveDir)
	}
}

// TestLoad_PartialConfig_ArchiveDirOnly checks that tickets_dir defaults to projectRoot
// when only archive_dir is specified.
func TestLoad_PartialConfig_ArchiveDirOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfigFile(t, dir, `archive_dir = "custom/archive"
`)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantTicketsDir := filepath.Join(dir, "tickets")
	wantArchiveDir := filepath.Join(dir, "custom/archive")

	if cfg.TicketsDir != wantTicketsDir {
		t.Errorf("TicketsDir: got %q, want %q", cfg.TicketsDir, wantTicketsDir)
	}
	if cfg.ArchiveDir != wantArchiveDir {
		t.Errorf("ArchiveDir: got %q, want %q", cfg.ArchiveDir, wantArchiveDir)
	}
}

// TestLoad_EmptyTOML checks that an empty config file returns all defaults.
func TestLoad_EmptyTOML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfigFile(t, dir, "")

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error for empty TOML: %v", err)
	}

	wantTicketsDir := filepath.Join(dir, "tickets")
	if cfg.TicketsDir != wantTicketsDir {
		t.Errorf("TicketsDir: got %q, want %q", cfg.TicketsDir, wantTicketsDir)
	}

	wantArchive := filepath.Join(dir, "tickets", "archive")
	if cfg.ArchiveDir != wantArchive {
		t.Errorf("ArchiveDir: got %q, want %q", cfg.ArchiveDir, wantArchive)
	}
}

// TestLoad_MalformedTOML_ErrorWrapping checks the error wraps config sentinel.
func TestLoad_MalformedTOML_ErrorWrapping(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfigFile(t, dir, "[[broken")

	_, err := config.Load(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, config.ErrMalformedConfig) {
		t.Errorf("error %v does not wrap ErrMalformedConfig", err)
	}
}

// TestLoad_AbsoluteTicketsDir checks that absolute paths in config are used directly.
func TestLoad_AbsoluteTicketsDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	absTarget := filepath.Join(dir, "custom-abs")

	content := "tickets_dir = " + `"` + absTarget + `"` + "\n"
	writeConfigFile(t, dir, content)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// An absolute path in config should be used as-is (it is already absolute).
	if cfg.TicketsDir != absTarget {
		t.Errorf("TicketsDir: got %q, want %q", cfg.TicketsDir, absTarget)
	}
}
