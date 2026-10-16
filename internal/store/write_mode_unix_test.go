//go:build unix

package store_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// withUmask sets the process umask for the duration of t and restores it on
// cleanup. Callers must not use t.Parallel: the umask is process-wide, and
// top-level parallel tests only resume after every sequential test finished.
func withUmask(t *testing.T, mask int) {
	t.Helper()
	old := syscall.Umask(mask)
	t.Cleanup(func() { syscall.Umask(old) })
}

func fileMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return info.Mode().Perm()
}

// TestWriteTicketPreservesExistingMode verifies that rewriting an existing
// ticket keeps its permission bits, independent of the umask.
func TestWriteTicketPreservesExistingMode(t *testing.T) {
	withUmask(t, 0o077)
	for _, mode := range []os.FileMode{0o644, 0o600, 0o640, 0o664} {
		s := newStore(t)
		path := writeTicketFile(t, s.TicketsDir, "0001-fix-login-timeout-on-staging.md", makeTicket("0001", testTitle))
		if err := os.Chmod(path, mode); err != nil {
			t.Fatalf("chmod: %v", err)
		}

		if err := s.WriteTicket(makeTicket("0001", testTitle2), path); err != nil {
			t.Fatalf("WriteTicket: %v", err)
		}

		if got := fileMode(t, path); got != mode {
			t.Errorf("mode after rewrite = %o, want %o", got, mode)
		}
	}
}

// TestWriteTicketNewFileHonoursUmask verifies that a ticket written to a new
// path gets 0666 minus the umask, not a hardcoded 0600.
func TestWriteTicketNewFileHonoursUmask(t *testing.T) {
	for _, tc := range []struct {
		umask int
		want  os.FileMode
	}{
		{0o022, 0o644},
		{0o027, 0o640},
		{0o077, 0o600},
	} {
		withUmask(t, tc.umask)
		s := newStore(t)
		path := filepath.Join(s.TicketsDir, "0001-fix-login-timeout-on-staging.md")

		if err := s.WriteTicket(makeTicket("0001", testTitle), path); err != nil {
			t.Fatalf("WriteTicket: %v", err)
		}

		if got := fileMode(t, path); got != tc.want {
			t.Errorf("umask %03o: mode = %o, want %o", tc.umask, got, tc.want)
		}
	}
}
