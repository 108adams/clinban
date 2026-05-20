package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runInit executes the clinban binary with "init [args...]" in the given
// working directory and returns stdout, stderr, and the exit code.
func runInit(t *testing.T, bin, workDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmdArgs := append([]string{"init"}, args...)
	cmd := exec.Command(bin, cmdArgs...)
	cmd.Dir = workDir
	cmd.Env = coverEnv()
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

// TestInitFreshDirectory verifies that "clinban init" in a clean directory
// creates all three artifacts and exits 0.
func TestInitFreshDirectory(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	stdout, stderr, exitCode := runInit(t, bin, dir)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if !strings.Contains(stdout, "created: tickets/") {
		t.Errorf("stdout does not contain %q: %q", "created: tickets/", stdout)
	}
	if !strings.Contains(stdout, "created: tickets/archive/") {
		t.Errorf("stdout does not contain %q: %q", "created: tickets/archive/", stdout)
	}
	if !strings.Contains(stdout, "created: .clinban") {
		t.Errorf("stdout does not contain %q: %q", "created: .clinban", stdout)
	}

	// Verify artifacts exist on disk.
	if _, err := os.Stat(filepath.Join(dir, "tickets")); err != nil {
		t.Errorf("tickets/ not found on disk: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "tickets", "archive")); err != nil {
		t.Errorf("tickets/archive/ not found on disk: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".clinban")); err != nil {
		t.Errorf(".clinban not found on disk: %v", err)
	}
}

// TestInitAlreadyExists_NoForce verifies that "clinban init" without --force
// exits 1 and reports each existing artifact on stderr when all three already exist.
func TestInitAlreadyExists_NoForce(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	// Pre-create all three artifacts.
	if err := os.MkdirAll(filepath.Join(dir, "tickets", "archive"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".clinban"), []byte(""), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	stdout, stderr, exitCode := runInit(t, bin, dir)

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1; stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if !strings.Contains(stderr, "tickets/") {
		t.Errorf("stderr does not mention tickets/: %q", stderr)
	}
	if !strings.Contains(stderr, "tickets/archive/") {
		t.Errorf("stderr does not mention tickets/archive/: %q", stderr)
	}
	if !strings.Contains(stderr, ".clinban") {
		t.Errorf("stderr does not mention .clinban: %q", stderr)
	}
}

// TestInitAlreadyExists_WithForce verifies that "clinban init --force" exits 1
// with "already fully initialized" on stderr when all three artifacts already exist.
func TestInitAlreadyExists_WithForce(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	// Pre-create all three artifacts.
	if err := os.MkdirAll(filepath.Join(dir, "tickets", "archive"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".clinban"), []byte(""), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	stdout, stderr, exitCode := runInit(t, bin, dir, "--force")

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1; stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if !strings.Contains(stderr, "already fully initialized") {
		t.Errorf("stderr does not contain %q: %q", "already fully initialized", stderr)
	}
}

// TestInitPartial_DirsExist_NoConfig_Force verifies that "clinban init --force"
// creates only the missing .clinban when tickets/ and tickets/archive/ already exist.
func TestInitPartial_DirsExist_NoConfig_Force(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	// Pre-create directories only, no .clinban.
	if err := os.MkdirAll(filepath.Join(dir, "tickets", "archive"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	stdout, stderr, exitCode := runInit(t, bin, dir, "--force")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if !strings.Contains(stdout, "created: .clinban") {
		t.Errorf("stdout does not contain %q: %q", "created: .clinban", stdout)
	}
	if strings.Contains(stdout, "created: tickets/") {
		t.Errorf("stdout unexpectedly contains %q: %q", "created: tickets/", stdout)
	}
	if strings.Contains(stdout, "created: tickets/archive/") {
		t.Errorf("stdout unexpectedly contains %q: %q", "created: tickets/archive/", stdout)
	}
}

// TestInitPartial_ConfigExists_NoDirs_Force verifies that "clinban init --force"
// creates tickets/ and tickets/archive/ when only .clinban already exists.
func TestInitPartial_ConfigExists_NoDirs_Force(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	// Pre-create .clinban only, no directories.
	if err := os.WriteFile(filepath.Join(dir, ".clinban"), []byte(""), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	stdout, stderr, exitCode := runInit(t, bin, dir, "--force")

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if !strings.Contains(stdout, "created: tickets/") {
		t.Errorf("stdout does not contain %q: %q", "created: tickets/", stdout)
	}
	if !strings.Contains(stdout, "created: tickets/archive/") {
		t.Errorf("stdout does not contain %q: %q", "created: tickets/archive/", stdout)
	}
	if strings.Contains(stdout, "created: .clinban") {
		t.Errorf("stdout unexpectedly contains %q: %q", "created: .clinban", stdout)
	}
}
