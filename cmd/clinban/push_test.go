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

// Constants for push tests.
const (
	pushTestID        = "0007"
	pushTestSlug      = "add-dashboard-widget"
	pushTestFile      = "0007-add-dashboard-widget.md"
	pushTestUnknownID = "8888"
)

// pushTicketContent returns a valid ticket file body with the given id and status.
func pushTicketContent(id, status string) string {
	now := time.Now().UTC().Format(time.RFC3339)
	return fmt.Sprintf(`---
id: "%s"
status: %s
type: task
title: Add dashboard widget
tags: []
created: %s
updated: %s
---
`, id, status, now, now)
}

// runPush executes the clinban binary with "push [args...]" in the given
// working directory and returns stdout, stderr, and the exit code.
func runPush(t *testing.T, bin, workDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmdArgs := append([]string{"push"}, args...)
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

// TestPushFromBacklog verifies that a backlog ticket is advanced to in-progress,
// exits 0, stdout contains "moved to in-progress", and the file has status: in-progress.
func TestPushFromBacklog(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	ticketPath := writeTicket(t, ticketsDir, pushTestFile, pushTicketContent(pushTestID, "backlog"))

	stdout, stderr, code := runPush(t, bin, root, pushTestID)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "moved to in-progress") {
		t.Errorf("stdout = %q, want to contain %q", stdout, "moved to in-progress")
	}

	content, err := os.ReadFile(ticketPath)
	if err != nil {
		t.Fatalf("read ticket file: %v", err)
	}
	if !strings.Contains(string(content), "status: in-progress") {
		t.Errorf("ticket file does not contain 'status: in-progress':\n%s", content)
	}
}

// TestPushFromInProgress verifies that an in-progress ticket is advanced to done,
// exits 0, and stdout contains "moved to done".
func TestPushFromInProgress(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, pushTestFile, pushTicketContent(pushTestID, "in-progress"))

	stdout, stderr, code := runPush(t, bin, root, pushTestID)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "moved to done") {
		t.Errorf("stdout = %q, want to contain %q", stdout, "moved to done")
	}
}

// TestPushFromBlocked verifies that a blocked ticket is advanced to in-progress,
// exits 0, and stdout contains "moved to in-progress".
func TestPushFromBlocked(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, pushTestFile, pushTicketContent(pushTestID, "blocked"))

	stdout, stderr, code := runPush(t, bin, root, pushTestID)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "moved to in-progress") {
		t.Errorf("stdout = %q, want to contain %q", stdout, "moved to in-progress")
	}
}

// TestPushFromDone verifies that a done ticket exits 0 and stdout contains
// "already done".
func TestPushFromDone(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, pushTestFile, pushTicketContent(pushTestID, "done"))

	stdout, stderr, code := runPush(t, bin, root, pushTestID)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "already done") {
		t.Errorf("stdout = %q, want to contain %q", stdout, "already done")
	}
}

// TestPushUnknownStatus verifies that a ticket with an unrecognised status exits 1
// and reports schema corruption rather than silently treating it as final.
func TestPushUnknownStatus(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, pushTestFile, pushTicketContent(pushTestID, "triaged"))

	_, stderr, code := runPush(t, bin, root, pushTestID)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "unrecognised status") {
		t.Errorf("stderr = %q, want to contain %q", stderr, "unrecognised status")
	}
}

// TestPushTicketNotFound verifies that pushing an unknown ID exits 1 and stderr
// contains "not found".
func TestPushTicketNotFound(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	// No tickets in the directory.
	_, stderr, code := runPush(t, bin, root, pushTestUnknownID)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(strings.ToLower(stderr), "not found") {
		t.Errorf("stderr = %q, want to contain 'not found'", stderr)
	}
}

// TestPushUpdatesTimestamp verifies that the updated timestamp is refreshed
// after a successful push.
func TestPushUpdatesTimestamp(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	oldTime := "2000-01-01T00:00:00Z"
	now := time.Now().UTC().Format(time.RFC3339)
	content := fmt.Sprintf(`---
id: "%s"
status: backlog
type: task
title: Add dashboard widget
tags: []
created: %s
updated: %s
---
`, pushTestID, now, oldTime)
	ticketPath := writeTicket(t, ticketsDir, pushTestFile, content)

	_, _, code := runPush(t, bin, root, pushTestID)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	updated, err := os.ReadFile(ticketPath)
	if err != nil {
		t.Fatalf("read ticket file: %v", err)
	}
	if strings.Contains(string(updated), oldTime) {
		t.Errorf("updated timestamp not changed; file still contains %q:\n%s", oldTime, updated)
	}
}

// TestPushNoArgs verifies that the push command fails with no arguments.
func TestPushNoArgs(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	_, _, code := runPush(t, bin, root)

	if code == 0 {
		t.Error("exit code = 0, want non-zero for missing argument")
	}
}

// TestPushOutputToStdout verifies that successful push output goes to stdout.
func TestPushOutputToStdout(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, pushTestFile, pushTicketContent(pushTestID, "backlog"))

	stdout, _, code := runPush(t, bin, root, pushTestID)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, pushTestID) {
		t.Errorf("stdout = %q, does not contain ticket ID %q", stdout, pushTestID)
	}
}

// TestPushFinalStatusToStdout verifies that the "final status" message goes to
// stdout (not stderr) and exits 0.
func TestPushFinalStatusToStdout(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	writeTicket(t, ticketsDir, pushTestFile, pushTicketContent(pushTestID, "done"))

	stdout, stderr, code := runPush(t, bin, root, pushTestID)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, pushTestID) {
		t.Errorf("stdout = %q, does not contain ticket ID %q", stdout, pushTestID)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("stderr = %q, want empty for 'already done' case", stderr)
	}
}

// TestPushFromInProgressUpdatesFileToDone verifies the file on disk has
// status: done after pushing an in-progress ticket.
func TestPushFromInProgressUpdatesFileToDone(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	ticketPath := writeTicket(t, ticketsDir, pushTestFile, pushTicketContent(pushTestID, "in-progress"))

	_, _, code := runPush(t, bin, root, pushTestID)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	content, err := os.ReadFile(ticketPath)
	if err != nil {
		t.Fatalf("read ticket file: %v", err)
	}
	if !strings.Contains(string(content), "status: done") {
		t.Errorf("ticket file does not contain 'status: done':\n%s", content)
	}
}

// TestPushTwoArgs verifies that the push command fails with two arguments
// (only one ID expected).
func TestPushTwoArgs(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	_, _, code := runPush(t, bin, root, pushTestID, "extra-arg")

	if code == 0 {
		t.Error("exit code = 0, want non-zero for too many arguments")
	}
}

// TestPushErrorToStderr verifies that "ticket not found" goes to stderr.
func TestPushErrorToStderr(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	stdout, stderr, code := runPush(t, bin, root, pushTestUnknownID)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" && strings.Contains(stdout, "not found") {
		t.Errorf("error appeared on stdout, should be on stderr: stdout=%q", stdout)
	}
	if !strings.Contains(stderr, "not found") {
		t.Errorf("stderr = %q, want 'ticket not found' message", stderr)
	}
}

// setupWorkDirWithConfigAndTickets creates a work dir that also has a .clinban
// config file — required when tests need the binary to find the project root.
func setupPushWorkDir(t *testing.T) (root, ticketsDir, archiveDir string) {
	t.Helper()
	root, ticketsDir, archiveDir = setupWorkDir(t)
	// Write a minimal .clinban config so findProjectRoot stops here.
	configPath := filepath.Join(root, ".clinban")
	if err := os.WriteFile(configPath, []byte(""), 0o600); err != nil {
		t.Fatalf("setupPushWorkDir: write .clinban: %v", err)
	}
	return root, ticketsDir, archiveDir
}
