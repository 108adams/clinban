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

// Constants for test ticket content.
const (
	testTicketID    = "0001"
	testTicketSlug  = "fix-login-timeout"
	testTicketTitle = "Fix login timeout"
)

// validTicketContent returns a fully-valid ticket file body with the given id
// and filename prefix. The filename must be "<id>-<slug>.md".
func validTicketContent(id string) string {
	now := time.Now().UTC().Format(time.RFC3339)
	return fmt.Sprintf(`---
id: "%s"
status: backlog
type: task
title: Fix login timeout
tags: []
created: %s
updated: %s
---
`, id, now, now)
}

// missingTypeTicketContent returns a ticket missing the 'type' field.
func missingTypeTicketContent(id string) string {
	now := time.Now().UTC().Format(time.RFC3339)
	return fmt.Sprintf(`---
id: "%s"
status: backlog
title: Fix login timeout
tags: []
created: %s
updated: %s
---
`, id, now, now)
}

// buildBinary returns the path to the clinban test binary.
//
// When TestMain has successfully built a coverage-instrumented binary
// (testBin != ""), that shared binary is returned immediately — no rebuild.
// This avoids a redundant "go build" per test and ensures all subprocess
// executions use the -cover binary so their counters land in testCoverDir.
//
// If TestMain's build failed (graceful-degradation path), a plain binary is
// compiled into a per-test temp directory as a fallback.
func buildBinary(t *testing.T) string {
	t.Helper()
	if testBin != "" {
		return testBin
	}
	// Fallback: build without -cover into a per-test temp dir.
	root, err := findModuleRoot()
	if err != nil {
		t.Fatalf("buildBinary: %v", err)
	}
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "clinban")
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/clinban/")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return binPath
}

// setupWorkDir creates a temp directory tree matching the default layout:
//
//	root/
//	  tickets/
//	    archive/
//
// It returns all three paths, all of which are guaranteed to exist on disk.
func setupWorkDir(t *testing.T) (root, ticketsDir, archiveDir string) {
	t.Helper()
	root = t.TempDir()
	ticketsDir = filepath.Join(root, "tickets")
	archiveDir = filepath.Join(ticketsDir, "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatalf("setupWorkDir: %v", err)
	}
	return root, ticketsDir, archiveDir
}

// writeTicket writes content to <dir>/<filename>.
func writeTicket(t *testing.T, dir, filename, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writeTicket: %v", err)
	}
	return path
}

// runLint executes the clinban binary with "lint [args...]" in the given
// working directory and returns stdout, stderr, and the exit code.
func runLint(t *testing.T, bin, workDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmdArgs := append([]string{"lint"}, args...)
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

// TestLintAllValid tests that "clinban lint" exits 0 and prints nothing when
// all tickets are valid.
func TestLintAllValid(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, "0001-fix-login-timeout.md", validTicketContent("0001"))
	writeTicket(t, ticketsDir, "0002-another-ticket.md", validTicketContent("0002"))

	stdout, stderr, code := runLint(t, bin, root)

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

// TestLintAllMissingType tests that "clinban lint" exits 1 and prints an error
// line when a ticket is missing the 'type' field.
func TestLintAllMissingType(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, "0001-fix-login-timeout.md", missingTypeTicketContent("0001"))

	stdout, _, code := runLint(t, bin, root)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stdout, "0001-fix-login-timeout.md") {
		t.Errorf("stdout does not name the file: %q", stdout)
	}
	if !strings.Contains(stdout, "type") {
		t.Errorf("stdout does not mention 'type' field: %q", stdout)
	}
}

// TestLintAllDuplicateID tests that "clinban lint" flags both files when two
// tickets share the same id value (one in active, one in archive).
// AllIDs is built from filename prefixes; having prefix "0001" appear in both
// active and archive means allIDs contains two "0001" entries, triggering rule 7.
func TestLintAllDuplicateID(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, archiveDir := setupWorkDir(t)

	// Put a ticket with id=0001 in the active directory.
	writeTicket(t, ticketsDir, "0001-fix-login-timeout.md", validTicketContent("0001"))

	// Put another ticket with id=0001 (and matching filename prefix) in archive.
	writeTicket(t, archiveDir, "0001-old-ticket.md", validTicketContent("0001"))

	stdout, _, code := runLint(t, bin, root)

	if code != 1 {
		t.Errorf("exit code = %d, want 1; stdout=%q", code, stdout)
	}
	// Both files should be flagged by rule 7 (non-unique id "0001").
	lines := nonEmptyLines(stdout)
	if len(lines) < 2 {
		t.Errorf("expected at least 2 error lines, got %d: %q", len(lines), stdout)
	}
}

// TestLintSingleValid tests that "clinban lint <id>" exits 0 silently for a
// valid ticket.
func TestLintSingleValid(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, "0001-fix-login-timeout.md", validTicketContent("0001"))

	stdout, stderr, code := runLint(t, bin, root, "0001")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
}

// TestLintSingleMissingType tests that "clinban lint <id>" exits 1 and prints
// errors when the named ticket has schema violations.
func TestLintSingleMissingType(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, "0001-fix-login-timeout.md", missingTypeTicketContent("0001"))

	stdout, _, code := runLint(t, bin, root, "0001")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stdout, "type") {
		t.Errorf("stdout does not mention 'type' field: %q", stdout)
	}
}

// TestLintSingleUnknownID tests that "clinban lint <id>" prints "ticket not
// found" to stderr and exits 1 when the ID does not exist.
func TestLintSingleUnknownID(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	// No ticket files in tickets dir.
	_, stderr, code := runLint(t, bin, root, "9999")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "ticket not found") {
		t.Errorf("stderr = %q, want 'ticket not found'", stderr)
	}
}

// TestLintIncludesArchive tests that "clinban lint" also validates tickets in
// the archive subdirectory.
func TestLintIncludesArchive(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, archiveDir := setupWorkDir(t)

	// Put a bad ticket in the archive dir.
	writeTicket(t, archiveDir, "0001-fix-login-timeout.md", missingTypeTicketContent("0001"))

	stdout, _, code := runLint(t, bin, root)

	if code != 1 {
		t.Errorf("exit code = %d, want 1; stdout=%q", code, stdout)
	}
	if !strings.Contains(stdout, "type") {
		t.Errorf("stdout does not mention 'type': %q", stdout)
	}
}

// TestLintNoTickets tests that "clinban lint" exits 0 when there are no tickets
// at all (empty directories).
func TestLintNoTickets(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	stdout, stderr, code := runLint(t, bin, root)

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
}

// TestLintOutputToStdout checks that lint errors go to stdout (not stderr),
// per the design spec.
func TestLintOutputToStdout(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, "0001-fix-login-timeout.md", missingTypeTicketContent("0001"))

	stdout, stderr, code := runLint(t, bin, root)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout == "" {
		t.Error("expected lint errors on stdout, got nothing")
	}
	// The lint error itself should NOT appear on stderr.
	if strings.Contains(stderr, "type") {
		t.Errorf("lint error appeared on stderr, should be on stdout: %q", stderr)
	}
}

// nonEmptyLines splits s by newline and returns non-empty lines.
func nonEmptyLines(s string) []string {
	var result []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return result
}
