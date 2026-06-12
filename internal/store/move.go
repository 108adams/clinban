package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// RenameOp describes a single within-directory rename for BatchRenameWithinDir.
// OldPath is the full path to the existing source file. NewBase is the
// destination basename only (no path separators); the destination directory is
// the same as OldPath's directory.
type RenameOp struct {
	OldPath string
	NewBase string
}

// OpKind discriminates the kind of filesystem operation in an error report.
type OpKind int

const (
	// OpLink represents an os.Link call.
	OpLink OpKind = iota
	// OpRemove represents an os.Remove call.
	OpRemove
)

// String returns "link" or "remove".
func (k OpKind) String() string {
	switch k {
	case OpLink:
		return "link"
	case OpRemove:
		return "remove"
	default:
		return "unknown"
	}
}

// OpError records a single failed filesystem operation within a batch rename.
type OpError struct {
	Kind OpKind
	Base string // basename of the file involved
	Err  error
}

// BatchError is the error type returned by BatchRenameWithinDir on failure.
// Failed is the primary operation that triggered the failure. Rollback holds
// any errors from the best-effort cleanup or restore steps; it is non-empty
// only when the filesystem may be in an inconsistent state.
type BatchError struct {
	Failed   OpError
	Rollback []OpError
}

// Error implements the error interface. It returns a store-scoped summary
// string. The CLI layer owns the "resolve:" prefix and reformats for display.
func (e *BatchError) Error() string {
	return fmt.Sprintf("store: batch rename: %s %s: %s", e.Failed.Kind, e.Failed.Base, e.Failed.Err)
}

// Inconsistent reports whether the rollback encountered errors, meaning the
// filesystem may be in an inconsistent state.
func (e *BatchError) Inconsistent() bool {
	return len(e.Rollback) > 0
}

// linkFile is the package-level seam for os.Link, used exclusively by
// BatchRenameWithinDir. Override in tests to inject deterministic failures.
var linkFile = os.Link

// removeFile is the package-level seam for os.Remove, used exclusively by
// BatchRenameWithinDir. Override in tests to inject deterministic failures.
var removeFile = os.Remove

// BatchRenameWithinDir renames all ops atomically within their respective
// directories using a two-phase link+remove protocol with rollback.
//
// Pre-flight: validates every NewBase is a bare basename. On failure returns a
// plain error with zero filesystem mutation.
//
// Phase 1 (Link): creates destination hard links. On failure at any op, all
// already-created links are removed best-effort and a *BatchError is returned.
//
// Phase 2 (Remove): removes source files. On success returns dest paths in op
// order and nil. On failure (TASK-002 completes the rollback): returns
// *BatchError.
//
// Return type is ([]string, error) — not ([]string, *BatchError) — to avoid
// the Go typed-nil footgun. Callers use errors.As to extract *BatchError.
func (s *Store) BatchRenameWithinDir(ops []RenameOp) ([]string, error) {
	// Pre-compute dest paths.
	dests := make([]string, len(ops))
	for i, op := range ops {
		dests[i] = filepath.Join(filepath.Dir(op.OldPath), op.NewBase)
	}

	// PRE-FLIGHT: validate every NewBase is a bare basename (no path separators).
	// This entire loop runs before any linkFile call, preserving the zero-mutation
	// guarantee: no filesystem change occurs if any op is invalid.
	for _, op := range ops {
		if op.NewBase != filepath.Base(op.NewBase) {
			return nil, fmt.Errorf("store: batch rename: new basename contains path separators: %s", op.NewBase)
		}
	}

	// PHASE 1 — Link: create destination hard links.
	// Track dests successfully linked so they can be removed on failure.
	linked := make([]string, 0, len(ops))
	for i, op := range ops {
		if err := linkFile(op.OldPath, dests[i]); err != nil {
			// Rollback Phase 1: best-effort remove every already-linked dest.
			var rollback []OpError
			for _, dest := range linked {
				if rmErr := removeFile(dest); rmErr != nil {
					rollback = append(rollback, OpError{
						Kind: OpRemove,
						Base: filepath.Base(dest),
						Err:  rmErr,
					})
				}
			}
			return nil, &BatchError{
				Failed: OpError{
					Kind: OpLink,
					Base: filepath.Base(dests[i]),
					Err:  err,
				},
				Rollback: rollback,
			}
		}
		linked = append(linked, dests[i])
	}

	// PHASE 2 — Remove: delete source files.
	for _, op := range ops {
		if err := removeFile(op.OldPath); err != nil {
			// TASK-002: restore removed sources before cleanup, then remove all dest links.
			return nil, &BatchError{
				Failed: OpError{
					Kind: OpRemove,
					Base: filepath.Base(op.OldPath),
					Err:  err,
				},
			}
		}
	}

	return dests, nil
}

// MoveToArchive moves path into ArchiveDir and returns the new path.
//
// ArchiveDir is created if necessary. MoveToArchive refuses to overwrite an
// existing destination file with the same basename. The move is performed
// atomically via os.Link + os.Remove to avoid a TOCTOU race.
func (s *Store) MoveToArchive(path string) (string, error) {
	if err := os.MkdirAll(s.ArchiveDir, 0o755); err != nil {
		return "", fmt.Errorf("store: move to archive: mkdir: %w", err)
	}

	dest := filepath.Join(s.ArchiveDir, filepath.Base(path))
	if err := os.Link(path, dest); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("store: move to archive: destination already exists: %s", filepath.Base(dest))
		}
		return "", fmt.Errorf("store: move to archive: link: %w", err)
	}

	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("store: move to archive: remove: %w", err)
	}
	return dest, nil
}

// MoveToActive moves path into TicketsDir and returns the new path.
//
// MoveToActive preserves the source basename and refuses to overwrite an
// existing destination file. The move is performed atomically via
// os.Link + os.Remove to avoid a TOCTOU race.
func (s *Store) MoveToActive(path string) (string, error) {
	dest := filepath.Join(s.TicketsDir, filepath.Base(path))
	if err := os.Link(path, dest); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("store: move to active: destination already exists: %s", filepath.Base(dest))
		}
		return "", fmt.Errorf("store: move to active: link: %w", err)
	}

	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("store: move to active: remove: %w", err)
	}
	return dest, nil
}

// RenameWithinDir renames path to newBase in the same directory and returns
// the new full path.
//
// The operation refuses to overwrite an existing destination and uses
// os.Link + os.Remove to match the collision behavior of ticket moves.
func (s *Store) RenameWithinDir(path, newBase string) (string, error) {
	if newBase != filepath.Base(newBase) {
		return "", fmt.Errorf("store: rename: new basename contains path separators: %s", newBase)
	}
	dest := filepath.Join(filepath.Dir(path), newBase)
	if err := os.Link(path, dest); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("store: rename: destination already exists: %s", filepath.Base(dest))
		}
		return "", fmt.Errorf("store: rename: link: %w", err)
	}

	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("store: rename: remove: %w", err)
	}
	return dest, nil
}
