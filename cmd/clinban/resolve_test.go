package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func resolveTicketContent(created, title string) string {
	return fmt.Sprintf(`---
title: "%s"
status: backlog
type: task
tags: []
created: "%s"
updated: "%s"
---

Body
`, title, created, created)
}

func invalidResolveTicketContent() string {
	return `---
title: [
---
`
}

func runResolve(t *testing.T, bin, workDir string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(bin, "resolve")
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

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be missing, stat error: %v", path, err)
	}
}

func TestResolveNoConflicts(t *testing.T) {
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)
	writeTicket(t, ticketsDir, "0001-first.md", resolveTicketContent("2026-06-10T10:00:00Z", "First"))
	writeTicket(t, ticketsDir, "0002-second.md", resolveTicketContent("2026-06-10T11:00:00Z", "Second"))

	stdout, stderr, code := runResolve(t, bin, root)
	if code != 0 {
		t.Fatalf("resolve exit = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if strings.TrimSpace(stdout) != "no conflicts found" {
		t.Errorf("stdout = %q, want no conflicts found", stdout)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

func TestResolveActiveConflictKeepsOldest(t *testing.T) {
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)
	oldPath := writeTicket(t, ticketsDir, "0001-old.md", resolveTicketContent("2026-06-10T10:00:00Z", "Old"))
	youngPath := writeTicket(t, ticketsDir, "0001-young.md", resolveTicketContent("2026-06-10T11:00:00Z", "Young"))

	stdout, stderr, code := runResolve(t, bin, root)
	if code != 0 {
		t.Fatalf("resolve exit = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	assertExists(t, oldPath)
	assertMissing(t, youngPath)
	assertExists(t, filepath.Join(ticketsDir, "0002-young.md"))
	if !strings.Contains(stdout, "renamed: tickets/0001-young.md -> tickets/0002-young.md") {
		t.Errorf("stdout = %q, want rename line", stdout)
	}
}

func TestResolveArchiveConflictKeepsArchived(t *testing.T) {
	bin := buildBinary(t)
	root, _, archiveDir := setupWorkDir(t)
	oldPath := writeTicket(t, archiveDir, "0001-old.md", resolveTicketContent("2026-06-10T10:00:00Z", "Old"))
	youngPath := writeTicket(t, archiveDir, "0001-young.md", resolveTicketContent("2026-06-10T11:00:00Z", "Young"))

	stdout, stderr, code := runResolve(t, bin, root)
	if code != 0 {
		t.Fatalf("resolve exit = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	assertExists(t, oldPath)
	assertMissing(t, youngPath)
	assertExists(t, filepath.Join(archiveDir, "0002-young.md"))
}

func TestResolveActiveArchiveConflict(t *testing.T) {
	bin := buildBinary(t)
	root, ticketsDir, archiveDir := setupWorkDir(t)
	activePath := writeTicket(t, ticketsDir, "0001-active.md", resolveTicketContent("2026-06-10T10:00:00Z", "Active"))
	archivePath := writeTicket(t, archiveDir, "0001-archived.md", resolveTicketContent("2026-06-10T11:00:00Z", "Archived"))

	stdout, stderr, code := runResolve(t, bin, root)
	if code != 0 {
		t.Fatalf("resolve exit = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	assertExists(t, activePath)
	assertMissing(t, archivePath)
	assertExists(t, filepath.Join(archiveDir, "0002-archived.md"))
}

func TestResolveThreeTicketConflict(t *testing.T) {
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)
	writeTicket(t, ticketsDir, "0001-old.md", resolveTicketContent("2026-06-10T10:00:00Z", "Old"))
	writeTicket(t, ticketsDir, "0001-middle.md", resolveTicketContent("2026-06-10T11:00:00Z", "Middle"))
	writeTicket(t, ticketsDir, "0001-young.md", resolveTicketContent("2026-06-10T12:00:00Z", "Young"))

	stdout, stderr, code := runResolve(t, bin, root)
	if code != 0 {
		t.Fatalf("resolve exit = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	assertExists(t, filepath.Join(ticketsDir, "0001-old.md"))
	assertExists(t, filepath.Join(ticketsDir, "0002-middle.md"))
	assertExists(t, filepath.Join(ticketsDir, "0003-young.md"))
}

func TestResolveCreatedTieUsesPathOrder(t *testing.T) {
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)
	created := "2026-06-10T10:00:00Z"
	firstPath := writeTicket(t, ticketsDir, "0001-a-first.md", resolveTicketContent(created, "First"))
	secondPath := writeTicket(t, ticketsDir, "0001-b-second.md", resolveTicketContent(created, "Second"))

	stdout, stderr, code := runResolve(t, bin, root)
	if code != 0 {
		t.Fatalf("resolve exit = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	assertExists(t, firstPath)
	assertMissing(t, secondPath)
	assertExists(t, filepath.Join(ticketsDir, "0002-b-second.md"))
}

func TestResolveParseFailureLeavesFilesUnchanged(t *testing.T) {
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)
	oldPath := writeTicket(t, ticketsDir, "0001-old.md", resolveTicketContent("2026-06-10T10:00:00Z", "Old"))
	badPath := writeTicket(t, ticketsDir, "0001-bad.md", invalidResolveTicketContent())

	stdout, stderr, code := runResolve(t, bin, root)
	if code != 1 {
		t.Fatalf("resolve exit = %d, want 1; stdout=%q stderr=%q", code, stdout, stderr)
	}
	assertExists(t, oldPath)
	assertExists(t, badPath)
	if !strings.Contains(stderr, "resolve: read") {
		t.Errorf("stderr = %q, want parse error context", stderr)
	}
}

func TestResolvePlannedDestinationCollisionLeavesFilesUnchanged(t *testing.T) {
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)
	oldPath := writeTicket(t, ticketsDir, "0001-old.md", resolveTicketContent("2026-06-10T10:00:00Z", "Old"))
	youngPath := writeTicket(t, ticketsDir, "0001-young.md", resolveTicketContent("2026-06-10T11:00:00Z", "Young"))
	if err := os.Mkdir(filepath.Join(ticketsDir, "0002-young.md"), 0o755); err != nil {
		t.Fatalf("mkdir destination collision: %v", err)
	}

	stdout, stderr, code := runResolve(t, bin, root)
	if code != 1 {
		t.Fatalf("resolve exit = %d, want 1; stdout=%q stderr=%q", code, stdout, stderr)
	}
	assertExists(t, oldPath)
	assertExists(t, youngPath)
	if !strings.Contains(stderr, "destination already exists") {
		t.Errorf("stderr = %q, want destination collision", stderr)
	}
}

func TestResolveUnrelatedMalformedTicketDoesNotBlock(t *testing.T) {
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)
	writeTicket(t, ticketsDir, "0001-old.md", resolveTicketContent("2026-06-10T10:00:00Z", "Old"))
	writeTicket(t, ticketsDir, "0001-young.md", resolveTicketContent("2026-06-10T11:00:00Z", "Young"))
	writeTicket(t, ticketsDir, "0009-bad.md", invalidResolveTicketContent())

	stdout, stderr, code := runResolve(t, bin, root)
	if code != 0 {
		t.Fatalf("resolve exit = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	assertExists(t, filepath.Join(ticketsDir, "0010-young.md"))
}
