package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Constants for remove tests.
const (
	removeTestID   = "0010"
	removeTestSlug = "remove-me"
	removeTestFile = "0010-remove-me.md"

	removeTestCollisionFile1 = "0011-first-collision.md"
	removeTestCollisionFile2 = "0011-second-collision.md"
	removeTestCollisionID    = "0011"

	removeTestArchiveID   = "0012"
	removeTestArchiveFile = "0012-archived-ticket.md"
)

// runRemove executes the clinban binary with "remove [args...]" in the given
// working directory and returns stdout, stderr, and the exit code.
func runRemove(t *testing.T, bin, workDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmdArgs := append([]string{"remove"}, args...)
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

// TestRemoveHappyPath tests that "clinban remove <id>" with an existing ticket
// removes the file and prints "removed: <filename>" to stdout, exiting 0.
func TestRemoveHappyPath(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, removeTestFile, validTicketContent(removeTestID))

	stdout, stderr, code := runRemove(t, bin, root, removeTestID)

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	// File must no longer exist.
	if _, err := os.Stat(filepath.Join(ticketsDir, removeTestFile)); !os.IsNotExist(err) {
		t.Error("ticket still exists on disk after remove")
	}

	// stdout must contain "removed: <filename>".
	if !strings.Contains(stdout, "removed: "+removeTestFile) {
		t.Errorf("stdout = %q, want to contain %q", stdout, "removed: "+removeTestFile)
	}
}

// TestRemoveNotFound tests that "clinban remove <id>" with an unknown ID prints
// "ticket not found" to stderr and exits 1.
func TestRemoveNotFound(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	// No tickets in dir.
	_, stderr, code := runRemove(t, bin, root, "9999")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "ticket not found") {
		t.Errorf("stderr = %q, want to contain 'ticket not found'", stderr)
	}
}

// TestRemoveCollision tests that "clinban remove <id>" with two files sharing
// the same ID prefix prints both filenames and a reference to "lint" on stderr,
// then exits 1.
func TestRemoveCollision(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, removeTestCollisionFile1, validTicketContent(removeTestCollisionID))
	writeTicket(t, ticketsDir, removeTestCollisionFile2, validTicketContent(removeTestCollisionID))

	_, stderr, code := runRemove(t, bin, root, removeTestCollisionID)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}

	// Both filenames must appear in stderr.
	if !strings.Contains(stderr, removeTestCollisionFile1) {
		t.Errorf("stderr = %q, want to contain %q", stderr, removeTestCollisionFile1)
	}
	if !strings.Contains(stderr, removeTestCollisionFile2) {
		t.Errorf("stderr = %q, want to contain %q", stderr, removeTestCollisionFile2)
	}

	// stderr must mention "lint".
	if !strings.Contains(stderr, "lint") {
		t.Errorf("stderr = %q, want to contain 'lint'", stderr)
	}

	// Neither file must be deleted.
	if _, err := os.Stat(filepath.Join(ticketsDir, removeTestCollisionFile1)); err != nil {
		t.Errorf("collision file 1 unexpectedly missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ticketsDir, removeTestCollisionFile2)); err != nil {
		t.Errorf("collision file 2 unexpectedly missing: %v", err)
	}
}

// TestRemoveFromArchive tests that "clinban remove <id>" removes a ticket that
// lives in the archive directory and exits 0.
func TestRemoveFromArchive(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, archiveDir := setupWorkDir(t)

	writeTicket(t, archiveDir, removeTestArchiveFile, doneTicketContent(removeTestArchiveID))

	stdout, stderr, code := runRemove(t, bin, root, removeTestArchiveID)

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	// File must no longer exist in archive.
	if _, err := os.Stat(filepath.Join(archiveDir, removeTestArchiveFile)); !os.IsNotExist(err) {
		t.Error("archived ticket still exists after remove")
	}
}

// TestRemovePrintsFilename tests that stdout contains just the filename
// (not the full path) after a successful remove.
func TestRemovePrintsFilename(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, removeTestFile, validTicketContent(removeTestID))

	stdout, _, code := runRemove(t, bin, root, removeTestID)

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}

	// Must contain just the filename, not the absolute path.
	if !strings.Contains(stdout, removeTestFile) {
		t.Errorf("stdout = %q, want to contain filename %q", stdout, removeTestFile)
	}
	// Must NOT contain a path separator from an absolute path component.
	if strings.Contains(stdout, ticketsDir) {
		t.Errorf("stdout = %q, must not contain full path %q", stdout, ticketsDir)
	}
}
