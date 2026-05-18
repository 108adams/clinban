package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Constants for list-command tests.
const (
	listTestID1 = "0001"
	listTestID2 = "0002"
	listTestID3 = "0003"
	listTestID4 = "0004"

	listNoActiveMsg = "No active tickets"
)

// makeTicket returns a valid ticket file body for the given parameters.
func makeTicket(id, status, typ, title string, tags []string) string {
	now := time.Now().UTC().Format(time.RFC3339)
	tagLine := "tags: []"
	if len(tags) > 0 {
		quoted := make([]string, len(tags))
		for i, t := range tags {
			quoted[i] = fmt.Sprintf("%q", t)
		}
		tagLine = "tags: [" + strings.Join(quoted, ", ") + "]"
	}
	return fmt.Sprintf(`---
id: %q
status: %s
type: %s
title: %s
%s
created: %s
updated: %s
---
`, id, status, typ, title, tagLine, now, now)
}

// writeListTicket writes a ticket file into dir with a derived filename.
func writeListTicket(t *testing.T, dir, id, status, typ, title string, tags []string) string {
	t.Helper()
	filename := id + "-" + strings.ReplaceAll(strings.ToLower(title), " ", "-") + ".md"
	content := makeTicket(id, status, typ, title, tags)
	return writeTicket(t, dir, filename, content)
}

// runList executes "clinban list [args...]" in workDir and returns stdout,
// stderr, and the exit code.
func runList(t *testing.T, bin, workDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmdArgs := append([]string{"list"}, args...)
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

// TestListNoTickets verifies "No active tickets" is printed and exit code is 0
// when there are no ticket files.
func TestListNoTickets(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	stdout, stderr, code := runList(t, bin, dir)

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, listNoActiveMsg) {
		t.Errorf("stdout = %q, want %q", stdout, listNoActiveMsg)
	}
}

// TestListNoMatchingFilter verifies "No active tickets" when filters exclude all
// tickets.
func TestListNoMatchingFilter(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	writeListTicket(t, dir, listTestID1, "backlog", "task", "Fix login timeout", nil)

	stdout, _, code := runList(t, bin, dir, "--status", "in-progress")

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, listNoActiveMsg) {
		t.Errorf("stdout = %q, want %q", stdout, listNoActiveMsg)
	}
}

// TestListAllTicketsUnfiltered verifies that all active tickets appear in output
// when no filters are applied.
func TestListAllTicketsUnfiltered(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	writeListTicket(t, dir, listTestID1, "backlog", "task", "Fix login timeout", nil)
	writeListTicket(t, dir, listTestID2, "in-progress", "bug", "Auth loop crash", nil)

	stdout, stderr, code := runList(t, bin, dir)

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, listTestID1) {
		t.Errorf("stdout missing ticket %s: %q", listTestID1, stdout)
	}
	if !strings.Contains(stdout, listTestID2) {
		t.Errorf("stdout missing ticket %s: %q", listTestID2, stdout)
	}
}

// TestListSortOrder verifies the priority sort: in-progress → blocked →
// backlog → done.
func TestListSortOrder(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	writeListTicket(t, dir, listTestID1, "done", "task", "Done ticket", nil)
	writeListTicket(t, dir, listTestID2, "backlog", "task", "Backlog ticket", nil)
	writeListTicket(t, dir, listTestID3, "blocked", "task", "Blocked ticket", nil)
	writeListTicket(t, dir, listTestID4, "in-progress", "task", "In progress ticket", nil)

	stdout, _, code := runList(t, bin, dir)

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}

	lines := nonEmptyLines(stdout)
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d: %q", len(lines), stdout)
	}

	// in-progress must appear before blocked, blocked before backlog, backlog
	// before done.
	posInProgress := indexContaining(lines, "in-progress")
	posBlocked := indexContaining(lines, "blocked")
	posBacklog := indexContaining(lines, "backlog")
	posDone := indexContaining(lines, "done")

	if posInProgress == -1 {
		t.Fatalf("in-progress line not found: %q", lines)
	}
	if posBlocked == -1 {
		t.Fatalf("blocked line not found: %q", lines)
	}
	if posBacklog == -1 {
		t.Fatalf("backlog line not found: %q", lines)
	}
	if posDone == -1 {
		t.Fatalf("done line not found: %q", lines)
	}

	if !(posInProgress < posBlocked && posBlocked < posBacklog && posBacklog < posDone) {
		t.Errorf("sort order wrong: in-progress=%d blocked=%d backlog=%d done=%d",
			posInProgress, posBlocked, posBacklog, posDone)
	}
}

// TestListSortOrderWithinGroup verifies ascending ID order within a status group.
func TestListSortOrderWithinGroup(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	// Write in reverse ID order so we can confirm sorting puts them right.
	writeListTicket(t, dir, "0003", "backlog", "task", "Third ticket", nil)
	writeListTicket(t, dir, "0001", "backlog", "task", "First ticket", nil)
	writeListTicket(t, dir, "0002", "backlog", "task", "Second ticket", nil)

	stdout, _, code := runList(t, bin, dir)

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}

	lines := nonEmptyLines(stdout)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), stdout)
	}

	// Each line must lead with the ID. Verify ascending order.
	for i, want := range []string{"0001", "0002", "0003"} {
		if !strings.HasPrefix(lines[i], want) {
			t.Errorf("line[%d] = %q, want prefix %q", i, lines[i], want)
		}
	}
}

// TestListFilterByStatus verifies --status filters to matching tickets only.
func TestListFilterByStatus(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	writeListTicket(t, dir, listTestID1, "backlog", "task", "Backlog ticket", nil)
	writeListTicket(t, dir, listTestID2, "in-progress", "task", "In progress ticket", nil)

	stdout, _, code := runList(t, bin, dir, "--status", "in-progress")

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	lines := nonEmptyLines(stdout)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %q", len(lines), stdout)
	}
	if !strings.Contains(lines[0], "in-progress") {
		t.Errorf("line does not contain in-progress: %q", lines[0])
	}
	if strings.Contains(stdout, "backlog") {
		t.Errorf("backlog ticket should be filtered out: %q", stdout)
	}
}

// TestListFilterByType verifies --type filters to matching tickets only.
func TestListFilterByType(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	writeListTicket(t, dir, listTestID1, "backlog", "bug", "Bug ticket", nil)
	writeListTicket(t, dir, listTestID2, "backlog", "task", "Task ticket", nil)

	stdout, _, code := runList(t, bin, dir, "--type", "bug")

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	lines := nonEmptyLines(stdout)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %q", len(lines), stdout)
	}
	if !strings.Contains(lines[0], "bug") {
		t.Errorf("line does not contain bug: %q", lines[0])
	}
}

// TestListFilterByTag verifies --tag filters to matching tickets only.
func TestListFilterByTag(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	writeListTicket(t, dir, listTestID1, "backlog", "task", "Tagged ticket", []string{"auth", "urgent"})
	writeListTicket(t, dir, listTestID2, "backlog", "task", "Untagged ticket", nil)

	stdout, _, code := runList(t, bin, dir, "--tag", "auth")

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	lines := nonEmptyLines(stdout)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %q", len(lines), stdout)
	}
	if !strings.Contains(lines[0], listTestID1) {
		t.Errorf("line does not contain %s: %q", listTestID1, lines[0])
	}
}

// TestListFilterCombined verifies AND logic when multiple filters are applied.
func TestListFilterCombined(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	// Only ticket 2 should match: status=backlog AND type=bug.
	writeListTicket(t, dir, listTestID1, "backlog", "task", "Backlog task", nil)
	writeListTicket(t, dir, listTestID2, "backlog", "bug", "Backlog bug", nil)
	writeListTicket(t, dir, listTestID3, "in-progress", "bug", "In-progress bug", nil)

	stdout, _, code := runList(t, bin, dir, "--status", "backlog", "--type", "bug")

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	lines := nonEmptyLines(stdout)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %q", len(lines), stdout)
	}
	if !strings.Contains(lines[0], listTestID2) {
		t.Errorf("expected ticket %s: %q", listTestID2, lines[0])
	}
}

// TestListOutputColumns verifies the output includes ID, status, type, and title
// on each line.
func TestListOutputColumns(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	writeListTicket(t, dir, listTestID1, "backlog", "task", "Fix login timeout", nil)

	stdout, _, code := runList(t, bin, dir)

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	lines := nonEmptyLines(stdout)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %q", len(lines), stdout)
	}
	line := lines[0]
	for _, want := range []string{listTestID1, "backlog", "task", "Fix login timeout"} {
		if !strings.Contains(line, want) {
			t.Errorf("line missing %q: %q", want, line)
		}
	}
}

// TestListInvalidStatus verifies that an invalid --status value exits 1.
func TestListInvalidStatus(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	_, stderr, code := runList(t, bin, dir, "--status", "invalid-status")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	// The error about invalid status should appear on stderr.
	if !strings.Contains(stderr, "invalid") && !strings.Contains(stderr, "status") {
		t.Errorf("stderr = %q, expected invalid status message", stderr)
	}
}

// TestListInvalidType verifies that an invalid --type value exits 1.
func TestListInvalidType(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	_, stderr, code := runList(t, bin, dir, "--type", "invalid-type")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "invalid") && !strings.Contains(stderr, "type") {
		t.Errorf("stderr = %q, expected invalid type message", stderr)
	}
}

// TestListDoesNotIncludeArchive verifies that tickets in the archive directory
// are not shown by "clinban list".
func TestListDoesNotIncludeArchive(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	// Write a ticket in the archive directory.
	archiveDir := fmt.Sprintf("%s/archive", dir)
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeListTicket(t, archiveDir, listTestID1, "done", "task", "Archived ticket", nil)

	stdout, _, code := runList(t, bin, dir)

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, listNoActiveMsg) {
		t.Errorf("expected %q but got: %q", listNoActiveMsg, stdout)
	}
}

// TestListHelpFlag verifies that --help exits 0 and produces usage output.
func TestListHelpFlag(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	stdout, _, code := runList(t, bin, dir, "--help")

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "list") {
		t.Errorf("stdout = %q, want usage with 'list'", stdout)
	}
}

// indexContaining returns the index of the first element in lines that contains
// substr, or -1 if not found.
func indexContaining(lines []string, substr string) int {
	for i, l := range lines {
		if strings.Contains(l, substr) {
			return i
		}
	}
	return -1
}
