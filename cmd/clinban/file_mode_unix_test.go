//go:build unix

package main_test

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// withUmask sets the process umask (inherited by the clinban subprocess) for
// the duration of t. Callers must not use t.Parallel: the umask is
// process-wide, and top-level parallel tests only resume after every
// sequential test finished.
func withUmask(t *testing.T, mask int) {
	t.Helper()
	old := syscall.Umask(mask)
	t.Cleanup(func() { syscall.Umask(old) })
}

func filePerm(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return info.Mode().Perm()
}

// TestNewNoInteractiveHonoursUmask verifies that a ticket created from flags
// gets 0666 minus the umask.
func TestNewNoInteractiveHonoursUmask(t *testing.T) {
	withUmask(t, 0o027)
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	stdout, stderr, code := runNew(t, bin, root, "--title", newTestTitle, "--type", newTestType)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	path := filepath.Join(ticketsDir, fmt.Sprintf("%s-%s.md", newExpectedID, newExpectedSlug))
	if got := filePerm(t, path); got != 0o640 {
		t.Errorf("mode = %o, want 640", got)
	}
}

// TestNewInteractiveHonoursUmask verifies that a ticket created through the
// editor temp file gets 0666 minus the umask, not the temp file's 0600.
func TestNewInteractiveHonoursUmask(t *testing.T) {
	withUmask(t, 0o022)
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)
	editor := makeEditorScript(t, t.TempDir(), interactiveTestTitle, interactiveTestType)

	stdout, stderr, code := runNewInteractive(t, bin, root, editor, "")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	path := filepath.Join(ticketsDir, fmt.Sprintf("%s-%s.md", interactiveTestID, interactiveTestSlug))
	if got := filePerm(t, path); got != 0o644 {
		t.Errorf("mode = %o, want 644", got)
	}
}

// TestMovePreservesTicketMode verifies that a status change rewrites the
// ticket without changing its permission bits.
func TestMovePreservesTicketMode(t *testing.T) {
	withUmask(t, 0o077)
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)
	path := writeTicket(t, ticketsDir, moveTestFile, moveTicketContent(moveTestID, "backlog"))
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	stdout, stderr, code := runMove(t, bin, root, moveTestID, "in-progress")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	if got := filePerm(t, path); got != 0o644 {
		t.Errorf("mode = %o, want 644", got)
	}
}
