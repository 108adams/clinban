package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Constants for config command tests.
const (
	configDefaultTicketsDir  = "tickets"
	configDefaultArchiveDir  = "tickets/archive"
	configCustomTicketsDir   = "mytickets"
	configCustomArchiveDir   = "myarchive"
	configValidDefaultType   = "feature"
	configInvalidDefaultType = "notavalidtype"
)

// runConfig executes "clinban config [args...]" in workDir and returns stdout,
// stderr, and the exit code.
func runConfig(t *testing.T, bin, workDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmdArgs := append([]string{"config"}, args...)
	cmd := exec.Command(bin, cmdArgs...)
	cmd.Dir = workDir
	cmd.Env = coverEnv()
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return stdout, stderr, exitCode
}

// writeClibanConfig writes content to .clinban in dir.
func writeClibanConfig(t *testing.T, dir, content string) {
	t.Helper()
	path := filepath.Join(dir, ".clinban")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writeClibanConfig: %v", err)
	}
}

// TestConfigNoArgs_NoConfig verifies that with no .clinban all three keys are
// shown with default notes and exit code is 0.
func TestConfigNoArgs_NoConfig(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	stdout, stderr, code := runConfig(t, bin, root)

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr=%q", code, stderr)
	}

	// All three keys must appear.
	for _, key := range []string{"tickets_dir", "archive_dir", "default_type"} {
		if !strings.Contains(stdout, key) {
			t.Errorf("stdout missing key %q: %q", key, stdout)
		}
	}

	// Unset keys must show the not-set note.
	if !strings.Contains(stdout, "not set in .clinban") {
		t.Errorf("stdout should contain 'not set in .clinban' note: %q", stdout)
	}

	// default_type has no default so should say "no default".
	if !strings.Contains(stdout, "no default") {
		t.Errorf("stdout should contain 'no default' for default_type: %q", stdout)
	}

	// tickets_dir and archive_dir have defaults.
	if !strings.Contains(stdout, configDefaultTicketsDir) {
		t.Errorf("stdout missing default tickets_dir value %q: %q", configDefaultTicketsDir, stdout)
	}
	if !strings.Contains(stdout, configDefaultArchiveDir) {
		t.Errorf("stdout missing default archive_dir value %q: %q", configDefaultArchiveDir, stdout)
	}
}

// TestConfigNoArgs_WithOneSetKey verifies that an explicitly set key shows no
// parenthetical note, while unset keys still show their notes.
func TestConfigNoArgs_WithOneSetKey(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)
	writeClibanConfig(t, root, `tickets_dir = "`+configCustomTicketsDir+`"`+"\n")

	stdout, stderr, code := runConfig(t, bin, root)

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr=%q", code, stderr)
	}

	// tickets_dir is set; it must appear without the note.
	lines := nonEmptyLines(stdout)
	var ticketsDirLine string
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "tickets_dir") {
			ticketsDirLine = l
			break
		}
	}
	if ticketsDirLine == "" {
		t.Fatalf("tickets_dir line not found in output: %q", stdout)
	}
	if strings.Contains(ticketsDirLine, "not set in .clinban") {
		t.Errorf("tickets_dir line should not contain 'not set in .clinban': %q", ticketsDirLine)
	}
	if !strings.Contains(ticketsDirLine, configCustomTicketsDir) {
		t.Errorf("tickets_dir line should contain value %q: %q", configCustomTicketsDir, ticketsDirLine)
	}

	// archive_dir is not explicitly set so it should show the note.
	if !strings.Contains(stdout, "not set in .clinban") {
		t.Errorf("stdout should still contain 'not set in .clinban' for unset keys: %q", stdout)
	}
}

// TestConfigNoArgs_AllKeysSet verifies that when all three keys are set no
// parenthetical notes appear.
func TestConfigNoArgs_AllKeysSet(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)
	writeClibanConfig(t, root,
		`tickets_dir = "`+configCustomTicketsDir+`"`+"\n"+
			`archive_dir = "`+configCustomArchiveDir+`"`+"\n"+
			`default_type = "`+configValidDefaultType+`"`+"\n")

	stdout, stderr, code := runConfig(t, bin, root)

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr=%q", code, stderr)
	}
	if strings.Contains(stdout, "not set in .clinban") {
		t.Errorf("no 'not set' notes expected when all keys are set: %q", stdout)
	}
	if strings.Contains(stdout, "no default") {
		t.Errorf("no 'no default' notes expected when all keys are set: %q", stdout)
	}
}

// TestConfigSet_TicketsDir verifies that setting tickets_dir writes the value
// to .clinban and exits 0.
func TestConfigSet_TicketsDir(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	_, stderr, code := runConfig(t, bin, root, "tickets_dir="+configCustomTicketsDir)

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr=%q", code, stderr)
	}

	// Re-read .clinban and verify.
	data, err := os.ReadFile(filepath.Join(root, ".clinban"))
	if err != nil {
		t.Fatalf("read .clinban: %v", err)
	}
	if !strings.Contains(string(data), configCustomTicketsDir) {
		t.Errorf(".clinban missing value %q: %q", configCustomTicketsDir, string(data))
	}
}

// TestConfigSet_DefaultType verifies that setting default_type to a valid value
// writes it to .clinban.
func TestConfigSet_DefaultType(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	_, stderr, code := runConfig(t, bin, root, "default_type="+configValidDefaultType)

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr=%q", code, stderr)
	}

	data, err := os.ReadFile(filepath.Join(root, ".clinban"))
	if err != nil {
		t.Fatalf("read .clinban: %v", err)
	}
	if !strings.Contains(string(data), configValidDefaultType) {
		t.Errorf(".clinban missing default_type %q: %q", configValidDefaultType, string(data))
	}
}

// TestConfigSet_UpdateExistingFile verifies that setting a key updates the
// file without destroying existing keys.
func TestConfigSet_UpdateExistingFile(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)
	writeClibanConfig(t, root, `tickets_dir = "`+configCustomTicketsDir+`"`+"\n")

	_, stderr, code := runConfig(t, bin, root, "default_type=task")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr=%q", code, stderr)
	}

	data, err := os.ReadFile(filepath.Join(root, ".clinban"))
	if err != nil {
		t.Fatalf("read .clinban: %v", err)
	}
	content := string(data)
	// Both values must now be present.
	if !strings.Contains(content, configCustomTicketsDir) {
		t.Errorf(".clinban missing tickets_dir value: %q", content)
	}
	if !strings.Contains(content, "task") {
		t.Errorf(".clinban missing default_type value: %q", content)
	}
}

// TestConfigSet_UnknownKey verifies that an unknown key exits 1 with an error
// on stderr.
func TestConfigSet_UnknownKey(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	_, stderr, code := runConfig(t, bin, root, "unknown_field=value")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "unknown") {
		t.Errorf("stderr = %q, expected unknown key error", stderr)
	}
}

// TestConfigSet_InvalidDefaultType verifies that an invalid default_type value
// exits 1 with an error on stderr.
func TestConfigSet_InvalidDefaultType(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	_, stderr, code := runConfig(t, bin, root, "default_type="+configInvalidDefaultType)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stderr == "" {
		t.Error("expected error on stderr, got nothing")
	}
}

// TestConfigSet_MissingEquals verifies that a single arg without '=' exits 1.
func TestConfigSet_MissingEquals(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	_, stderr, code := runConfig(t, bin, root, "tickets_dir")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stderr == "" {
		t.Error("expected error on stderr, got nothing")
	}
}

// TestConfigSet_EmptyPathValue verifies that setting a path key to "" exits 1.
func TestConfigSet_EmptyPathValue(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	_, stderr, code := runConfig(t, bin, root, "tickets_dir=")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stderr == "" {
		t.Error("expected error on stderr, got nothing")
	}
}

// TestConfigHelp verifies that --help exits 0 and mentions "config".
func TestConfigHelp(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	stdout, _, code := runConfig(t, bin, root, "--help")

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "config") {
		t.Errorf("stdout = %q, want usage containing 'config'", stdout)
	}
}
