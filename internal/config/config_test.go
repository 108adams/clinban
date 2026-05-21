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

// TestLoad_DefaultType_Set checks that default_type is read from the config file.
func TestLoad_DefaultType_Set(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfigFile(t, dir, `default_type = "feature"
`)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const want = "feature"
	if cfg.DefaultType != want {
		t.Errorf("DefaultType: got %q, want %q", cfg.DefaultType, want)
	}
}

// TestLoad_DefaultType_Absent checks that DefaultType is empty string when not set.
func TestLoad_DefaultType_Absent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfigFile(t, dir, `tickets_dir = "tickets"
`)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DefaultType != "" {
		t.Errorf("DefaultType: got %q, want empty string", cfg.DefaultType)
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

// --------------------------------------------------------------------------
// Entries tests
// --------------------------------------------------------------------------

// TestEntries_NoConfig verifies that all three keys are returned with IsSet=false
// and sensible defaults when .clinban is absent.
func TestEntries_NoConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	entries, err := config.Entries(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	byKey := make(map[string]config.Entry, len(entries))
	for _, e := range entries {
		byKey[e.Key] = e
	}

	// tickets_dir
	td, ok := byKey["tickets_dir"]
	if !ok {
		t.Fatal("missing key tickets_dir")
	}
	if td.IsSet {
		t.Error("tickets_dir: expected IsSet=false")
	}
	if td.Default == "" {
		t.Error("tickets_dir: expected non-empty Default")
	}

	// archive_dir
	ad, ok := byKey["archive_dir"]
	if !ok {
		t.Fatal("missing key archive_dir")
	}
	if ad.IsSet {
		t.Error("archive_dir: expected IsSet=false")
	}
	if ad.Default == "" {
		t.Error("archive_dir: expected non-empty Default")
	}

	// default_type
	dt, ok := byKey["default_type"]
	if !ok {
		t.Fatal("missing key default_type")
	}
	if dt.IsSet {
		t.Error("default_type: expected IsSet=false")
	}
	if dt.Default != "" {
		t.Errorf("default_type: expected empty Default, got %q", dt.Default)
	}
}

// TestEntries_PartialConfig verifies that explicitly set keys have IsSet=true
// and unset keys have IsSet=false.
func TestEntries_PartialConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfigFile(t, dir, `tickets_dir = "mytickets"`+"\n")

	entries, err := config.Entries(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	byKey := make(map[string]config.Entry, len(entries))
	for _, e := range entries {
		byKey[e.Key] = e
	}

	if !byKey["tickets_dir"].IsSet {
		t.Error("tickets_dir: expected IsSet=true")
	}
	if byKey["tickets_dir"].Value == "" {
		t.Error("tickets_dir: expected non-empty Value")
	}
	if byKey["archive_dir"].IsSet {
		t.Error("archive_dir: expected IsSet=false")
	}
	if byKey["default_type"].IsSet {
		t.Error("default_type: expected IsSet=false")
	}
}

// TestEntries_FullConfig verifies all keys report IsSet=true when all are set.
func TestEntries_FullConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfigFile(t, dir, "tickets_dir = \"work\"\narchive_dir = \"work/done\"\ndefault_type = \"bug\"\n")

	entries, err := config.Entries(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	byKey := make(map[string]config.Entry, len(entries))
	for _, e := range entries {
		byKey[e.Key] = e
	}

	for _, key := range []string{"tickets_dir", "archive_dir", "default_type"} {
		if !byKey[key].IsSet {
			t.Errorf("%s: expected IsSet=true", key)
		}
	}
	if byKey["default_type"].Value != "bug" {
		t.Errorf("default_type value: got %q, want %q", byKey["default_type"].Value, "bug")
	}
}

// TestEntries_MalformedConfig verifies that a malformed .clinban returns an error.
func TestEntries_MalformedConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfigFile(t, dir, "[[broken")

	_, err := config.Entries(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, config.ErrMalformedConfig) {
		t.Errorf("expected ErrMalformedConfig, got: %v", err)
	}
}

// --------------------------------------------------------------------------
// SetKey tests
// --------------------------------------------------------------------------

// TestSetKey_CreatesMissingFile verifies SetKey creates .clinban when absent.
func TestSetKey_CreatesMissingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := config.SetKey(dir, "tickets_dir", "work"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := filepath.Join(dir, ".clinban")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf(".clinban not created: %v", err)
	}

	// Reload to confirm.
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("load after set: %v", err)
	}
	want := filepath.Join(dir, "work")
	if cfg.TicketsDir != want {
		t.Errorf("tickets_dir: got %q, want %q", cfg.TicketsDir, want)
	}
}

// TestSetKey_UpdatesExistingFile verifies SetKey updates an existing .clinban
// and preserves other fields.
func TestSetKey_UpdatesExistingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfigFile(t, dir, `tickets_dir = "original"`+"\n")

	if err := config.SetKey(dir, "archive_dir", "custom/archive"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("load after set: %v", err)
	}

	// Both fields should now be set.
	if cfg.ArchiveDir != filepath.Join(dir, "custom/archive") {
		t.Errorf("archive_dir: got %q", cfg.ArchiveDir)
	}
	// Original field preserved.
	if cfg.TicketsDir != filepath.Join(dir, "original") {
		t.Errorf("tickets_dir: got %q, want original", cfg.TicketsDir)
	}
}

// TestSetKey_DefaultType verifies setting default_type to a valid value.
func TestSetKey_DefaultType(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := config.SetKey(dir, "default_type", "feature"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("load after set: %v", err)
	}
	if cfg.DefaultType != "feature" {
		t.Errorf("default_type: got %q, want %q", cfg.DefaultType, "feature")
	}
}

// TestSetKey_UnknownKey verifies SetKey returns ErrUnknownKey for unknown keys.
func TestSetKey_UnknownKey(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	err := config.SetKey(dir, "unknown_field", "value")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, config.ErrUnknownKey) {
		t.Errorf("expected ErrUnknownKey, got: %v", err)
	}
}

// TestSetKey_InvalidDefaultType verifies SetKey returns ErrInvalidValue for
// a bad default_type value.
func TestSetKey_InvalidDefaultType(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	err := config.SetKey(dir, "default_type", "notatype")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, config.ErrInvalidValue) {
		t.Errorf("expected ErrInvalidValue, got: %v", err)
	}
}

// TestSetKey_EmptyTicketsDir verifies SetKey returns ErrInvalidValue for empty
// path values.
func TestSetKey_EmptyTicketsDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	err := config.SetKey(dir, "tickets_dir", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, config.ErrInvalidValue) {
		t.Errorf("expected ErrInvalidValue, got: %v", err)
	}
}

// TestSetKey_EmptyDefaultTypeAllowed verifies that setting default_type to ""
// is valid (it unsets the value).
func TestSetKey_EmptyDefaultTypeAllowed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeConfigFile(t, dir, `default_type = "task"`+"\n")

	if err := config.SetKey(dir, "default_type", ""); err != nil {
		t.Fatalf("unexpected error unsetting default_type: %v", err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("load after unset: %v", err)
	}
	if cfg.DefaultType != "" {
		t.Errorf("default_type: expected empty after unset, got %q", cfg.DefaultType)
	}
}

// TestSetKey_FileMode verifies that .clinban is created with 0600 permissions.
func TestSetKey_FileMode(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := config.SetKey(dir, "tickets_dir", "work"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, ".clinban"))
	if err != nil {
		t.Fatalf("stat .clinban: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("file mode: got %o, want 0600", mode)
	}
}
