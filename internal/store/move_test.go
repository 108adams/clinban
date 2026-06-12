package store

// White-box tests for BatchRenameWithinDir. Package store (not store_test) so
// tests can override the linkFile / removeFile package-level seams.

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/108adams/clinban/internal/config"
)

// ---- constants ----

const (
	batchSrc1 = "0001-alpha.md"
	batchSrc2 = "0002-beta.md"
	batchSrc3 = "0003-gamma.md"
	batchDst1 = "0001-alpha-new.md"
	batchDst2 = "0002-beta-new.md"
	batchDst3 = "0003-gamma-new.md"
)

// ---- helpers ----

// newBatchStore creates a Store backed by a fresh temp directory.
// TicketsDir is the temp dir itself; ArchiveDir is a subdirectory "archive".
// The archive subdirectory is NOT created — callers create it when needed.
func newBatchStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{
		TicketsDir: dir,
		ArchiveDir: filepath.Join(dir, "archive"),
	}
	return New(cfg)
}

// writeSrc writes a minimal source file into dir with the given basename and
// returns the full path.
func writeSrc(t *testing.T, dir, base string) string {
	t.Helper()
	path := filepath.Join(dir, base)
	if err := os.WriteFile(path, []byte("# "+base), 0o600); err != nil {
		t.Fatalf("writeSrc: write %s: %v", path, err)
	}
	return path
}

// assertExists fails t if path does not exist on the filesystem.
func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected %q to exist, got: %v", path, err)
	}
}

// assertAbsent fails t if path exists on the filesystem or if an unexpected
// error occurs (e.g. permission denied). Only fs.ErrNotExist is the expected
// outcome.
func assertAbsent(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	if err == nil {
		t.Errorf("expected %q to be absent, but it exists", path)
		return
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected %q to be absent, got unexpected stat error: %v", path, err)
	}
}

// ---- Tests ----

// TestBatchRenameWithinDir_SuccessSingleDir exercises the happy path for ops
// all within a single directory (s.TicketsDir).
func TestBatchRenameWithinDir_SuccessSingleDir(t *testing.T) {
	t.Parallel()

	s := newBatchStore(t)

	src1 := writeSrc(t, s.TicketsDir, batchSrc1)
	src2 := writeSrc(t, s.TicketsDir, batchSrc2)
	src3 := writeSrc(t, s.TicketsDir, batchSrc3)

	ops := []RenameOp{
		{OldPath: src1, NewBase: batchDst1},
		{OldPath: src2, NewBase: batchDst2},
		{OldPath: src3, NewBase: batchDst3},
	}

	got, err := s.BatchRenameWithinDir(ops)
	if err != nil {
		t.Fatalf("BatchRenameWithinDir returned unexpected error: %v", err)
	}

	// Returned slice must have exactly one path per op.
	if len(got) != len(ops) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(ops))
	}

	wantDests := []string{
		filepath.Join(s.TicketsDir, batchDst1),
		filepath.Join(s.TicketsDir, batchDst2),
		filepath.Join(s.TicketsDir, batchDst3),
	}

	for i, want := range wantDests {
		if got[i] != want {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want)
		}
		assertExists(t, got[i])
	}

	// All source paths must be gone.
	assertAbsent(t, src1)
	assertAbsent(t, src2)
	assertAbsent(t, src3)
}

// TestBatchRenameWithinDir_SuccessTwoDirs exercises the happy path where ops
// span two separate directories (TicketsDir and ArchiveDir). Each dest is
// placed in the same directory as its source.
func TestBatchRenameWithinDir_SuccessTwoDirs(t *testing.T) {
	t.Parallel()

	s := newBatchStore(t)
	if err := os.MkdirAll(s.ArchiveDir, 0o755); err != nil {
		t.Fatalf("mkdir archive: %v", err)
	}

	srcActive := writeSrc(t, s.TicketsDir, batchSrc1)
	srcArchive := writeSrc(t, s.ArchiveDir, batchSrc2)

	ops := []RenameOp{
		{OldPath: srcActive, NewBase: batchDst1},
		{OldPath: srcArchive, NewBase: batchDst2},
	}

	got, err := s.BatchRenameWithinDir(ops)
	if err != nil {
		t.Fatalf("BatchRenameWithinDir returned unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}

	wantActive := filepath.Join(s.TicketsDir, batchDst1)
	wantArchive := filepath.Join(s.ArchiveDir, batchDst2)

	if got[0] != wantActive {
		t.Errorf("got[0] = %q, want %q", got[0], wantActive)
	}
	if got[1] != wantArchive {
		t.Errorf("got[1] = %q, want %q", got[1], wantArchive)
	}

	assertExists(t, wantActive)
	assertExists(t, wantArchive)
	assertAbsent(t, srcActive)
	assertAbsent(t, srcArchive)
}

// TestBatchRenameWithinDir_PreflightInvalidBasename verifies that an op with a
// NewBase containing "/" returns a plain (non-BatchError) validation error and
// makes zero filesystem changes.
func TestBatchRenameWithinDir_PreflightInvalidBasename(t *testing.T) {
	t.Parallel()

	s := newBatchStore(t)

	src1 := writeSrc(t, s.TicketsDir, batchSrc1)
	src2 := writeSrc(t, s.TicketsDir, batchSrc2)

	ops := []RenameOp{
		{OldPath: src1, NewBase: batchDst1},
		{OldPath: src2, NewBase: "subdir/" + batchDst2}, // invalid: contains "/"
	}

	_, err := s.BatchRenameWithinDir(ops)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	// Must NOT be a *BatchError — it is a plain validation error.
	var be *BatchError
	if errors.As(err, &be) {
		t.Errorf("error is *BatchError; want plain validation error: %v", err)
	}

	// Filesystem must be unchanged: sources still exist, no dests created.
	assertExists(t, src1)
	assertExists(t, src2)
	assertAbsent(t, filepath.Join(s.TicketsDir, batchDst1))
	assertAbsent(t, filepath.Join(s.TicketsDir, batchDst2))
}

// TestBatchRenameWithinDir_Phase1LinkFailure verifies that when Phase-1 link
// fails at op[1] (because its dest already exists), the already-created link
// for op[0] is cleaned up and all source files remain intact.
func TestBatchRenameWithinDir_Phase1LinkFailure(t *testing.T) {
	t.Parallel()

	s := newBatchStore(t)

	src1 := writeSrc(t, s.TicketsDir, batchSrc1)
	src2 := writeSrc(t, s.TicketsDir, batchSrc2)

	// Pre-create the dest for op[1] so os.Link will return ErrExist.
	dest2 := filepath.Join(s.TicketsDir, batchDst2)
	if err := os.WriteFile(dest2, []byte("blocker"), 0o600); err != nil {
		t.Fatalf("setup: pre-create dest2: %v", err)
	}

	ops := []RenameOp{
		{OldPath: src1, NewBase: batchDst1}, // op[0]: will succeed link
		{OldPath: src2, NewBase: batchDst2}, // op[1]: dest pre-exists → ErrExist
	}

	_, err := s.BatchRenameWithinDir(ops)
	if err == nil {
		t.Fatal("expected error from Phase-1 link failure, got nil")
	}

	// Error must be a *BatchError.
	var be *BatchError
	if !errors.As(err, &be) {
		t.Fatalf("error is not *BatchError: %T %v", err, err)
	}

	// Failed op must be a link failure.
	if be.Failed.Kind != OpLink {
		t.Errorf("Failed.Kind = %v, want OpLink", be.Failed.Kind)
	}

	// The dest created by op[0] during Phase-1 must have been cleaned up.
	assertAbsent(t, filepath.Join(s.TicketsDir, batchDst1))

	// Both original source files must still exist.
	assertExists(t, src1)
	assertExists(t, src2)
}
