package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

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
