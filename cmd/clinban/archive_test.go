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

// Constants for archive tests.
const (
	archiveTestDoneID   = "0001"
	archiveTestDoneSlug = "fix-login-timeout"
	archiveTestDoneFile = "0001-fix-login-timeout.md"

	archiveTestActiveID   = "0002"
	archiveTestActiveSlug = "another-ticket"
	archiveTestActiveFile = "0002-another-ticket.md"

	archiveTestDoneID2   = "0003"
	archiveTestDoneSlug2 = "third-ticket"
	archiveTestDoneFile2 = "0003-third-ticket.md"
)

// doneTicketContent returns a fully-valid ticket with status=done.
func doneTicketContent(id string) string {
	now := time.Now().UTC().Format(time.RFC3339)
	return fmt.Sprintf(`---
id: "%s"
status: done
type: task
title: Fix login timeout
tags: []
created: %s
updated: %s
---
`, id, now, now)
}

// inProgressTicketContent returns a valid ticket with status=in-progress.
func inProgressTicketContent(id string) string {
	now := time.Now().UTC().Format(time.RFC3339)
	return fmt.Sprintf(`---
id: "%s"
status: in-progress
type: task
title: Another ticket
tags: []
created: %s
updated: %s
---
`, id, now, now)
}

// runArchive executes the clinban binary with "archive [args...]" in the given
// working directory and returns stdout, stderr, and the exit code.
// If stdinInput is non-empty it is piped to the process.
func runArchive(t *testing.T, bin, workDir string, stdinInput string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmdArgs := append([]string{"archive"}, args...)
	cmd := exec.Command(bin, cmdArgs...)
	cmd.Dir = workDir
	if stdinInput != "" {
		cmd.Stdin = strings.NewReader(stdinInput)
	}
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

// TestArchiveSingleHappyPath tests that "clinban archive <id>" with a done
// ticket moves the file to the archive directory and prints a confirmation.
func TestArchiveSingleHappyPath(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, archiveDir := setupWorkDir(t)

	writeTicket(t, ticketsDir, archiveTestDoneFile, doneTicketContent(archiveTestDoneID))

	stdout, stderr, code := runArchive(t, bin, root, "", archiveTestDoneID)

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "archived: "+archiveTestDoneFile) {
		t.Errorf("stdout = %q, want to contain %q", stdout, "archived: "+archiveTestDoneFile)
	}

	// The ticket must no longer be in the active directory.
	if _, err := os.Stat(filepath.Join(ticketsDir, archiveTestDoneFile)); !os.IsNotExist(err) {
		t.Error("ticket still exists in active directory after archiving")
	}

	// The ticket must now be in the archive directory.
	archivePath := filepath.Join(archiveDir, archiveTestDoneFile)
	if _, err := os.Stat(archivePath); err != nil {
		t.Errorf("ticket not found in archive directory: %v", err)
	}
}

// TestArchiveSingleNotFound tests that "clinban archive <id>" with an unknown
// ID prints "ticket not found" to stderr and exits 1.
func TestArchiveSingleNotFound(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	// No tickets in dir.
	_, stderr, code := runArchive(t, bin, root, "", "9999")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "ticket not found") {
		t.Errorf("stderr = %q, want to contain 'ticket not found'", stderr)
	}
}

// TestArchiveSingleNotDone tests that "clinban archive <id>" with a ticket
// that is not done prints an error to stderr and exits 1.
func TestArchiveSingleNotDone(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, archiveTestActiveFile, inProgressTicketContent(archiveTestActiveID))

	_, stderr, code := runArchive(t, bin, root, "", archiveTestActiveID)

	if code != 1 {
		t.Errorf("exit code = %d, want 1; stderr=%q", code, stderr)
	}
	if !strings.Contains(stderr, "done") {
		t.Errorf("stderr = %q, want to mention 'done' status requirement", stderr)
	}

	// The ticket must still be in the active directory.
	if _, err := os.Stat(filepath.Join(ticketsDir, archiveTestActiveFile)); err != nil {
		t.Errorf("ticket unexpectedly missing from active directory: %v", err)
	}
}

// TestArchiveSingleCreatesArchiveDir tests that "clinban archive <id>"
// creates the archive directory if it does not exist.
func TestArchiveSingleCreatesArchiveDir(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)

	// Manually create only root/tickets/ so that archive/ does NOT exist yet —
	// the command itself must create it.
	root := t.TempDir()
	ticketsDir := filepath.Join(root, "tickets")
	if err := os.MkdirAll(ticketsDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	writeTicket(t, ticketsDir, archiveTestDoneFile, doneTicketContent(archiveTestDoneID))

	// Confirm archive dir does NOT exist yet.
	archiveDir := filepath.Join(ticketsDir, "archive")
	if _, err := os.Stat(archiveDir); !os.IsNotExist(err) {
		t.Skip("archive dir already exists, test precondition not met")
	}

	_, _, code := runArchive(t, bin, root, "", archiveTestDoneID)

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if _, err := os.Stat(archiveDir); err != nil {
		t.Errorf("archive directory was not created: %v", err)
	}
}

// TestArchiveBulkNoDoneTickets tests that "clinban archive" with no done
// tickets prints "No done tickets to archive" and exits 0.
func TestArchiveBulkNoDoneTickets(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, archiveTestActiveFile, inProgressTicketContent(archiveTestActiveID))

	stdout, stderr, code := runArchive(t, bin, root, "")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "No done tickets to archive") {
		t.Errorf("stdout = %q, want to contain 'No done tickets to archive'", stdout)
	}
}

// TestArchiveBulkEmpty tests that "clinban archive" in a directory with no
// tickets at all prints "No done tickets to archive" and exits 0.
func TestArchiveBulkEmpty(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	stdout, stderr, code := runArchive(t, bin, root, "")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "No done tickets to archive") {
		t.Errorf("stdout = %q, want 'No done tickets to archive'", stdout)
	}
}

// TestArchiveBulkConfirmYes tests that "clinban archive" confirmed with 'y'
// moves all done tickets to archive and prints a count.
func TestArchiveBulkConfirmYes(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, archiveDir := setupWorkDir(t)

	writeTicket(t, ticketsDir, archiveTestDoneFile, doneTicketContent(archiveTestDoneID))
	writeTicket(t, ticketsDir, archiveTestDoneFile2, doneTicketContent(archiveTestDoneID2))
	// Also add one non-done ticket that should be untouched.
	writeTicket(t, ticketsDir, archiveTestActiveFile, inProgressTicketContent(archiveTestActiveID))

	stdout, stderr, code := runArchive(t, bin, root, "y")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	// Prompt must mention both done tickets.
	if !strings.Contains(stdout, archiveTestDoneFile) {
		t.Errorf("stdout = %q, want to list %q", stdout, archiveTestDoneFile)
	}
	if !strings.Contains(stdout, archiveTestDoneFile2) {
		t.Errorf("stdout = %q, want to list %q", stdout, archiveTestDoneFile2)
	}

	// The count in the output must be 2.
	if !strings.Contains(stdout, "2") {
		t.Errorf("stdout = %q, want to contain count '2'", stdout)
	}

	// Both done tickets must now be in archive.
	for _, f := range []string{archiveTestDoneFile, archiveTestDoneFile2} {
		if _, err := os.Stat(filepath.Join(archiveDir, f)); err != nil {
			t.Errorf("ticket %q not found in archive: %v", f, err)
		}
		// Must be gone from active dir.
		if _, err := os.Stat(filepath.Join(ticketsDir, f)); !os.IsNotExist(err) {
			t.Errorf("ticket %q still present in active dir after archiving", f)
		}
	}

	// The non-done ticket must remain in active dir.
	if _, err := os.Stat(filepath.Join(ticketsDir, archiveTestActiveFile)); err != nil {
		t.Errorf("non-done ticket unexpectedly missing from active dir: %v", err)
	}
}

// TestArchiveBulkConfirmUpperY tests that 'Y' (uppercase) is also accepted as confirmation.
func TestArchiveBulkConfirmUpperY(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, archiveDir := setupWorkDir(t)

	writeTicket(t, ticketsDir, archiveTestDoneFile, doneTicketContent(archiveTestDoneID))

	stdout, stderr, code := runArchive(t, bin, root, "Y")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	archivePath := filepath.Join(archiveDir, archiveTestDoneFile)
	if _, err := os.Stat(archivePath); err != nil {
		t.Errorf("ticket not found in archive directory: %v", err)
	}
}

// TestArchiveBulkConfirmNo tests that "clinban archive" confirmed with 'n'
// (or just Enter) leaves tickets untouched and exits 0.
func TestArchiveBulkConfirmNo(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)

	// Use manual setup so we can verify archive dir was not created by the command.
	// setupWorkDir pre-creates tickets/archive/, which would defeat the assertion.
	root := t.TempDir()
	ticketsDir := filepath.Join(root, "tickets")
	if err := os.MkdirAll(ticketsDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	writeTicket(t, ticketsDir, archiveTestDoneFile, doneTicketContent(archiveTestDoneID))

	stdout, _, code := runArchive(t, bin, root, "n")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q", code, stdout)
	}

	// Ticket must remain in active dir.
	if _, err := os.Stat(filepath.Join(ticketsDir, archiveTestDoneFile)); err != nil {
		t.Errorf("ticket unexpectedly moved despite 'n' answer: %v", err)
	}
	// No archive dir should have been created.
	if _, err := os.Stat(filepath.Join(ticketsDir, "archive")); !os.IsNotExist(err) {
		t.Error("archive directory was created despite 'n' answer")
	}
}

// TestArchiveBulkListsFilenames tests that the bulk prompt lists the filename
// of each done ticket before prompting.
func TestArchiveBulkListsFilenames(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, archiveTestDoneFile, doneTicketContent(archiveTestDoneID))

	// Use 'n' so we don't actually move anything; we just check the listing.
	stdout, _, code := runArchive(t, bin, root, "n")

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, archiveTestDoneFile) {
		t.Errorf("stdout = %q, want to list filename %q", stdout, archiveTestDoneFile)
	}
}

// TestArchiveBulkPromptFormat tests that the confirmation prompt contains the
// ticket count and the [y/N] marker.
func TestArchiveBulkPromptFormat(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, archiveTestDoneFile, doneTicketContent(archiveTestDoneID))

	stdout, _, _ := runArchive(t, bin, root, "n")

	if !strings.Contains(stdout, "1") {
		t.Errorf("prompt = %q, want to contain count '1'", stdout)
	}
	if !strings.Contains(stdout, "[y/N]") {
		t.Errorf("prompt = %q, want to contain '[y/N]'", stdout)
	}
}

// TestArchiveSingleErrorToStderr tests that single-archive errors go to stderr.
func TestArchiveSingleErrorToStderr(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, archiveTestActiveFile, inProgressTicketContent(archiveTestActiveID))

	stdout, stderr, code := runArchive(t, bin, root, "", archiveTestActiveID)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	// Error must be on stderr, not stdout.
	if strings.Contains(stdout, "done") {
		t.Errorf("error appeared on stdout, should be on stderr: stdout=%q", stdout)
	}
	if !strings.Contains(stderr, "done") {
		t.Errorf("stderr = %q, want to mention 'done'", stderr)
	}
}
