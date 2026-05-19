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

// Constants for move tests.
const (
	moveTestID        = "0042"
	moveTestSlug      = "fix-login-timeout"
	moveTestFile      = "0042-fix-login-timeout.md"
	moveTestTitle     = "Fix login timeout"
	moveTestType      = "task"
	moveTestUnknownID = "9999"
)

// moveTicketContent returns a valid ticket file body with the given id and status.
func moveTicketContent(id, status string) string {
	now := time.Now().UTC().Format(time.RFC3339)
	return fmt.Sprintf(`---
id: "%s"
status: %s
type: task
title: Fix login timeout
tags: []
created: %s
updated: %s
---
`, id, status, now, now)
}

// runMove executes the clinban binary with "move [args...]" in the given
// working directory and returns stdout, stderr, and the exit code.
func runMove(t *testing.T, bin, workDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmdArgs := append([]string{"move"}, args...)
	cmd := exec.Command(bin, cmdArgs...)
	cmd.Dir = workDir
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

// TestMoveHappyPathInProgressToDone verifies that a valid transition
// (in-progress → done) updates the status and prints confirmation.
func TestMoveHappyPathInProgressToDone(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, moveTestFile, moveTicketContent(moveTestID, "in-progress"))

	stdout, stderr, code := runMove(t, bin, root, moveTestID, "done")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, moveTestID+" moved to done") {
		t.Errorf("stdout = %q, want to contain %q", stdout, moveTestID+" moved to done")
	}
}

// TestMoveHappyPathBacklogToInProgress verifies backlog → in-progress transition.
func TestMoveHappyPathBacklogToInProgress(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, moveTestFile, moveTicketContent(moveTestID, "backlog"))

	stdout, stderr, code := runMove(t, bin, root, moveTestID, "in-progress")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, moveTestID+" moved to in-progress") {
		t.Errorf("stdout = %q, want to contain %q", stdout, moveTestID+" moved to in-progress")
	}
}

// TestMoveSameStatusExitsSilently verifies that moving to the current status
// exits 0 with no output.
func TestMoveSameStatusExitsSilently(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, moveTestFile, moveTicketContent(moveTestID, "in-progress"))

	stdout, stderr, code := runMove(t, bin, root, moveTestID, "in-progress")

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	// No output expected.
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("stdout = %q, want empty (silent no-op)", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("stderr = %q, want empty (silent no-op)", stderr)
	}
}

// TestMoveUnknownID verifies that an unknown ticket ID prints "ticket not found"
// to stderr and exits 1.
func TestMoveUnknownID(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	// No tickets in the directory.
	_, stderr, code := runMove(t, bin, root, moveTestUnknownID, "done")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(strings.ToLower(stderr), "ticket not found") {
		t.Errorf("stderr = %q, want to contain 'ticket not found'", stderr)
	}
}

// TestMoveInvalidStatus verifies that an invalid target status prints the valid
// status list to stderr and exits 1.
func TestMoveInvalidStatus(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, moveTestFile, moveTicketContent(moveTestID, "backlog"))

	_, stderr, code := runMove(t, bin, root, moveTestID, "not-a-status")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	// Must mention the invalid value.
	if !strings.Contains(stderr, "not-a-status") {
		t.Errorf("stderr = %q, want to mention invalid status value", stderr)
	}
	// Must mention at least one valid status.
	if !strings.Contains(stderr, "backlog") {
		t.Errorf("stderr = %q, want to list valid statuses including 'backlog'", stderr)
	}
}

// TestMoveForbiddenTransitionBlockedToDone verifies that a forbidden FSM
// transition (blocked → done) prints an error with valid next statuses and
// exits 1.
func TestMoveForbiddenTransitionBlockedToDone(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, moveTestFile, moveTicketContent(moveTestID, "blocked"))

	_, stderr, code := runMove(t, bin, root, moveTestID, "done")

	if code != 1 {
		t.Errorf("exit code = %d, want 1; stderr=%q", code, stderr)
	}
	// Must mention the forbidden transition.
	if !strings.Contains(stderr, "blocked") {
		t.Errorf("stderr = %q, want to mention 'blocked'", stderr)
	}
	if !strings.Contains(stderr, "done") {
		t.Errorf("stderr = %q, want to mention 'done'", stderr)
	}
	// Must list valid next statuses (in-progress is the only one from blocked).
	if !strings.Contains(stderr, "in-progress") {
		t.Errorf("stderr = %q, want to list valid transition 'in-progress'", stderr)
	}
}

// TestMoveForbiddenTransitionDoneToInProgress verifies done → in-progress is rejected.
func TestMoveForbiddenTransitionDoneToInProgress(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, moveTestFile, moveTicketContent(moveTestID, "done"))

	_, stderr, code := runMove(t, bin, root, moveTestID, "in-progress")

	if code != 1 {
		t.Errorf("exit code = %d, want 1; stderr=%q", code, stderr)
	}
	if !strings.Contains(stderr, "done") {
		t.Errorf("stderr = %q, want to mention 'done'", stderr)
	}
	// Valid next from done is backlog only.
	if !strings.Contains(stderr, "backlog") {
		t.Errorf("stderr = %q, want to list valid transition 'backlog'", stderr)
	}
}

// TestMoveUpdatesStatusOnDisk verifies that the ticket file on disk actually
// reflects the new status after a successful move.
func TestMoveUpdatesStatusOnDisk(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	ticketPath := writeTicket(t, ticketsDir, moveTestFile, moveTicketContent(moveTestID, "backlog"))

	_, _, code := runMove(t, bin, root, moveTestID, "in-progress")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	// Read the file back and check the status field.
	content, err := os.ReadFile(ticketPath)
	if err != nil {
		t.Fatalf("read ticket file: %v", err)
	}
	if !strings.Contains(string(content), "status: in-progress") {
		t.Errorf("ticket file does not contain 'status: in-progress':\n%s", content)
	}
}

// TestMoveUpdatedTimestampChanges verifies that the updated timestamp is refreshed
// after a successful move.
func TestMoveUpdatedTimestampChanges(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	// Create ticket with a known old timestamp.
	oldTime := "2000-01-01T00:00:00Z"
	now := time.Now().UTC().Format(time.RFC3339)
	content := fmt.Sprintf(`---
id: "%s"
status: backlog
type: task
title: Fix login timeout
tags: []
created: %s
updated: %s
---
`, moveTestID, now, oldTime)
	ticketPath := writeTicket(t, ticketsDir, moveTestFile, content)

	_, _, code := runMove(t, bin, root, moveTestID, "in-progress")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	updated, err := os.ReadFile(ticketPath)
	if err != nil {
		t.Fatalf("read ticket file: %v", err)
	}
	// The updated timestamp must differ from the old one.
	if strings.Contains(string(updated), oldTime) {
		t.Errorf("updated timestamp not changed; file still contains %q:\n%s", oldTime, updated)
	}
}

// TestMoveArchivedTicketDoneToBacklog verifies the special case: a done ticket in
// the archive is moved back to the active directory when its status becomes backlog.
func TestMoveArchivedTicketDoneToBacklog(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, archiveDir := setupWorkDir(t)

	// Write a done ticket in the archive directory.
	archivePath := writeTicket(t, archiveDir, moveTestFile, moveTicketContent(moveTestID, "done"))

	stdout, stderr, code := runMove(t, bin, root, moveTestID, "backlog")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, moveTestID+" moved to backlog") {
		t.Errorf("stdout = %q, want to contain %q", stdout, moveTestID+" moved to backlog")
	}

	// The ticket must no longer be in the archive.
	if _, err := os.Stat(archivePath); !os.IsNotExist(err) {
		t.Error("ticket still exists in archive directory after reopen")
	}

	// The ticket must now be in the active directory with status backlog.
	activePath := filepath.Join(ticketsDir, moveTestFile)
	if _, err := os.Stat(activePath); err != nil {
		t.Errorf("ticket not found in active directory after reopen: %v", err)
	}
	content, err := os.ReadFile(activePath)
	if err != nil {
		t.Fatalf("read active ticket: %v", err)
	}
	if !strings.Contains(string(content), "status: backlog") {
		t.Errorf("ticket does not have 'status: backlog' after reopen:\n%s", content)
	}
}

// TestMoveArchivedTicketInActiveDir verifies that a ticket in the archive
// (non-done status edge case is not applicable per FSM) can still be read
// and updated without moving it to active (i.e. it stays in archive if transition
// is not done→backlog).
//
// Note: The only valid transition from done is → backlog. This test instead
// exercises an archived ticket that has a status other than done (unusual edge
// case — possible via direct file edit). We treat it as a regular transition
// and do NOT move the file.
func TestMoveArchivedTicketNonReopenTransition(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, archiveDir := setupWorkDir(t)

	// Write a ticket in in-progress status in the archive (unusual but possible
	// via direct file edit).
	archiveFile := filepath.Join(archiveDir, moveTestFile)
	writeTicket(t, archiveDir, moveTestFile, moveTicketContent(moveTestID, "in-progress"))

	stdout, stderr, code := runMove(t, bin, root, moveTestID, "done")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, moveTestID+" moved to done") {
		t.Errorf("stdout = %q, want to contain %q", stdout, moveTestID+" moved to done")
	}

	// The ticket must remain in the archive (not moved to active).
	if _, err := os.Stat(archiveFile); err != nil {
		t.Errorf("ticket unexpectedly missing from archive: %v", err)
	}
	// Must not exist in active dir.
	if _, err := os.Stat(filepath.Join(ticketsDir, moveTestFile)); !os.IsNotExist(err) {
		t.Error("ticket unexpectedly appeared in active directory")
	}
}

// TestMoveAllValidTransitions exercises every valid FSM transition to confirm
// they all exit 0.
func TestMoveAllValidTransitions(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)

	transitions := []struct {
		from string
		to   string
	}{
		{"backlog", "in-progress"},
		{"backlog", "blocked"},
		{"in-progress", "blocked"},
		{"in-progress", "done"},
		{"blocked", "in-progress"},
		{"done", "backlog"},
	}

	for _, tc := range transitions {
		tc := tc
		t.Run(tc.from+"->"+tc.to, func(t *testing.T) {
			t.Parallel()
			root, ticketsDir, _ := setupWorkDir(t)

			writeTicket(t, ticketsDir, moveTestFile, moveTicketContent(moveTestID, tc.from))

			_, stderr, code := runMove(t, bin, root, moveTestID, tc.to)
			if code != 0 {
				t.Errorf("exit code = %d, want 0 for %s→%s; stderr=%q", code, tc.from, tc.to, stderr)
			}
		})
	}
}

// TestMoveAllInvalidTransitions exercises known forbidden transitions to confirm
// they all exit 1 with an error.
func TestMoveAllInvalidTransitions(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)

	// These are the transitions NOT in the valid table.
	// Source: requirements §4.
	forbiddenTransitions := []struct {
		from string
		to   string
	}{
		{"blocked", "done"},
		{"done", "in-progress"},
		{"backlog", "done"},
		{"backlog", "backlog"},
		{"in-progress", "backlog"},
		{"in-progress", "in-progress"},
		{"blocked", "backlog"},
		{"blocked", "blocked"},
		{"done", "done"},
		{"done", "blocked"},
	}

	for _, tc := range forbiddenTransitions {
		tc := tc
		t.Run(tc.from+"->"+tc.to, func(t *testing.T) {
			t.Parallel()

			// Same-status transitions exit 0 silently, so skip those.
			if tc.from == tc.to {
				root, ticketsDir, _ := setupWorkDir(t)
				writeTicket(t, ticketsDir, moveTestFile, moveTicketContent(moveTestID, tc.from))
				_, _, code := runMove(t, bin, root, moveTestID, tc.to)
				if code != 0 {
					t.Errorf("self-transition %s→%s: exit code = %d, want 0 (no-op)", tc.from, tc.to, code)
				}
				return
			}

			root, ticketsDir, _ := setupWorkDir(t)
			writeTicket(t, ticketsDir, moveTestFile, moveTicketContent(moveTestID, tc.from))

			_, stderr, code := runMove(t, bin, root, moveTestID, tc.to)
			if code != 1 {
				t.Errorf("exit code = %d, want 1 for forbidden %s→%s; stderr=%q", code, tc.from, tc.to, stderr)
			}
			if stderr == "" {
				t.Errorf("stderr is empty for forbidden %s→%s, want error message", tc.from, tc.to)
			}
		})
	}
}

// TestMoveErrorToStderr verifies that move errors go to stderr and nothing
// error-related appears on stdout.
func TestMoveErrorToStderr(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, moveTestFile, moveTicketContent(moveTestID, "blocked"))

	stdout, stderr, code := runMove(t, bin, root, moveTestID, "done")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	// Error must be on stderr, not stdout.
	if stdout != "" && strings.Contains(stdout, "cannot") {
		t.Errorf("error appeared on stdout, should be on stderr: stdout=%q", stdout)
	}
	if !strings.Contains(stderr, "cannot") {
		t.Errorf("stderr = %q, want FSM error message", stderr)
	}
}

// TestMoveConfirmationToStdout verifies that the success message goes to stdout.
func TestMoveConfirmationToStdout(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, moveTestFile, moveTicketContent(moveTestID, "in-progress"))

	stdout, _, code := runMove(t, bin, root, moveTestID, "done")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	// Confirmation must be on stdout.
	if !strings.Contains(stdout, moveTestID) {
		t.Errorf("stdout = %q, does not contain ID %q", stdout, moveTestID)
	}
	if !strings.Contains(stdout, "moved to") {
		t.Errorf("stdout = %q, does not contain 'moved to'", stdout)
	}
}

// TestMoveNoArgs verifies that the move command fails with no arguments.
func TestMoveNoArgs(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	_, _, code := runMove(t, bin, root)

	if code == 0 {
		t.Error("exit code = 0, want non-zero for missing arguments")
	}
}

// TestMoveOneArg verifies that the move command fails with only one argument.
func TestMoveOneArg(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	_, _, code := runMove(t, bin, root, moveTestID)

	if code == 0 {
		t.Error("exit code = 0, want non-zero for missing status argument")
	}
}
