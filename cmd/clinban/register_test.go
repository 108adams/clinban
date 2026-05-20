package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Constants for register tests.
const (
	registerTitle = "Fix login timeout"
	registerType  = "task"
)

// validRegisterContent returns a ticket file body that is valid for
// registration. It intentionally uses a placeholder ID so we can verify that
// clinban register overwrites it with a system-assigned value.
func validRegisterContent() string {
	now := time.Now().UTC().Format(time.RFC3339)
	return fmt.Sprintf(`---
id: "9999"
status: backlog
type: task
title: Fix login timeout
tags: []
created: %s
updated: %s
---
`, now, now)
}

// missingTitleRegisterContent returns a ticket that will fail lint because
// title is empty (missing required field).
func missingTitleRegisterContent() string {
	now := time.Now().UTC().Format(time.RFC3339)
	return fmt.Sprintf(`---
id: "9999"
status: backlog
type: task
title: ""
tags: []
created: %s
updated: %s
---
`, now, now)
}

// invalidYAMLContent returns content that cannot be parsed as YAML frontmatter.
func invalidYAMLContent() string {
	return "this is not a ticket file at all — no frontmatter\n"
}

// runRegister executes the clinban binary with "register <path>" in the given
// working directory and returns stdout, stderr, and the exit code.
func runRegister(t *testing.T, bin, workDir, path string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(bin, "register", path)
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

// TestRegisterHappyPath verifies that a valid external file is adopted:
// id/created/updated are overwritten, the file is moved to TicketsDir,
// the source is removed, and stdout prints "registered: <filename>".
func TestRegisterHappyPath(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	// Write the external file outside the tickets dir (a sibling temp dir).
	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "external-ticket.md")
	if err := os.WriteFile(srcPath, []byte(validRegisterContent()), 0o600); err != nil {
		t.Fatalf("setup: write source file: %v", err)
	}

	stdout, stderr, code := runRegister(t, bin, root, srcPath)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "registered:") {
		t.Errorf("stdout = %q, want 'registered: ...'", stdout)
	}

	// Source file must be deleted.
	if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
		t.Errorf("source file still exists at %s after register", srcPath)
	}

	// A ticket file must have appeared in ticketsDir.
	entries, err := os.ReadDir(ticketsDir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	var ticketFiles []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			ticketFiles = append(ticketFiles, e.Name())
		}
	}
	if len(ticketFiles) != 1 {
		t.Fatalf("expected 1 ticket file in tickets dir, got %d: %v", len(ticketFiles), ticketFiles)
	}

	// The filename printed in stdout must match the actual file on disk.
	registeredFile := strings.TrimPrefix(strings.TrimSpace(stdout), "registered: ")
	if registeredFile != ticketFiles[0] {
		t.Errorf("stdout says %q but file on disk is %q", registeredFile, ticketFiles[0])
	}
}

// TestRegisterOverwritesSystemFields verifies that id, created, updated in the
// registered file are system-assigned values (not the placeholder "9999").
func TestRegisterOverwritesSystemFields(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "external-ticket.md")
	if err := os.WriteFile(srcPath, []byte(validRegisterContent()), 0o600); err != nil {
		t.Fatalf("setup: write source file: %v", err)
	}

	_, _, code := runRegister(t, bin, root, srcPath)
	if code != 0 {
		t.Fatalf("register failed with exit code %d", code)
	}

	// Read the written ticket and verify system fields were overwritten.
	entries, _ := os.ReadDir(ticketsDir)
	var registeredPath string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			registeredPath = filepath.Join(ticketsDir, e.Name())
		}
	}
	if registeredPath == "" {
		t.Fatal("no ticket file found in tickets dir")
	}
	content, err := os.ReadFile(registeredPath)
	if err != nil {
		t.Fatalf("read registered ticket: %v", err)
	}
	// id should be 0001 (first ticket), not the placeholder 9999.
	if strings.Contains(string(content), `id: "9999"`) {
		t.Errorf("registered ticket still contains placeholder id '9999'")
	}
	if !strings.Contains(string(content), `id: "0001"`) {
		t.Errorf("registered ticket does not have system-assigned id '0001'; content:\n%s", content)
	}
}

// TestRegisterFileNotFound verifies that "file not found" is printed to stderr
// and exit code is 1 when the path does not exist.
func TestRegisterFileNotFound(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	_, stderr, code := runRegister(t, bin, root, "/nonexistent/path/ticket.md")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "file not found") {
		t.Errorf("stderr = %q, want 'file not found'", stderr)
	}
}

// TestRegisterInvalidYAML verifies that a parse error causes exit 1 and the
// error is printed to stderr; the source file is not deleted.
func TestRegisterInvalidYAML(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "bad-ticket.md")
	if err := os.WriteFile(srcPath, []byte(invalidYAMLContent()), 0o600); err != nil {
		t.Fatalf("setup: write source file: %v", err)
	}

	_, stderr, code := runRegister(t, bin, root, srcPath)

	if code != 1 {
		t.Errorf("exit code = %d, want 1; stderr=%q", code, stderr)
	}
	if stderr == "" {
		t.Errorf("expected parse error on stderr, got nothing")
	}

	// Source file must NOT be deleted.
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		t.Errorf("source file was deleted despite parse failure")
	}
}

// TestRegisterLintFailure verifies that when lint fails after system fields are
// filled, errors go to stderr, exit code is 1, and the file is not moved.
func TestRegisterLintFailure(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "bad-ticket.md")
	if err := os.WriteFile(srcPath, []byte(missingTitleRegisterContent()), 0o600); err != nil {
		t.Fatalf("setup: write source file: %v", err)
	}

	_, stderr, code := runRegister(t, bin, root, srcPath)

	if code != 1 {
		t.Errorf("exit code = %d, want 1; stderr=%q", code, stderr)
	}
	if stderr == "" {
		t.Errorf("expected lint errors on stderr, got nothing")
	}

	// Source file must NOT be deleted.
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		t.Errorf("source file was deleted despite lint failure")
	}

	// No ticket must appear in the tickets dir.
	entries, _ := os.ReadDir(ticketsDir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			t.Errorf("ticket file %q appeared in tickets dir despite lint failure", e.Name())
		}
	}
}

// TestRegisterNoArgument verifies that "clinban register" without a path
// argument exits 1 (Cobra validates argument count).
func TestRegisterNoArgument(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	cmd := exec.Command(bin, "register")
	cmd.Dir = root
	cmd.Env = coverEnv()
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected exit code 1, got 0")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() == 0 {
		t.Errorf("expected non-zero exit code, got %v", err)
	}
}

// TestRegisterIdempotentNextID verifies that sequential registrations assign
// incrementing IDs (0001, 0002, …).
func TestRegisterIdempotentNextID(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	for i, id := range []string{"0001", "0002"} {
		srcDir := t.TempDir()
		srcPath := filepath.Join(srcDir, fmt.Sprintf("ticket-%d.md", i))
		if err := os.WriteFile(srcPath, []byte(validRegisterContent()), 0o600); err != nil {
			t.Fatalf("setup: write source file: %v", err)
		}

		stdout, stderr, code := runRegister(t, bin, root, srcPath)
		if code != 0 {
			t.Fatalf("register %d: exit code = %d, want 0; stderr=%q", i, code, stderr)
		}

		// Verify the file with the expected id prefix exists in ticketsDir.
		entries, _ := os.ReadDir(ticketsDir)
		found := false
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), id) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("register %d: expected file with prefix %q in tickets dir; stdout=%q; files=%v",
				i, id, stdout, entries)
		}
	}
}
