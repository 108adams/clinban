package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// MoveToArchive moves path into ArchiveDir and returns the new path.
//
// ArchiveDir is created if necessary. MoveToArchive refuses to overwrite an
// existing destination file with the same basename.
func (s *Store) MoveToArchive(path string) (string, error) {
	if err := os.MkdirAll(s.ArchiveDir, 0o755); err != nil {
		return "", fmt.Errorf("store: move to archive: mkdir: %w", err)
	}

	dest := filepath.Join(s.ArchiveDir, filepath.Base(path))
	if _, err := os.Stat(dest); err == nil {
		return "", fmt.Errorf("store: move to archive: destination already exists: %s", filepath.Base(dest))
	}

	if err := os.Rename(path, dest); err != nil {
		return "", fmt.Errorf("store: move to archive: rename: %w", err)
	}
	return dest, nil
}

// MoveToActive moves path into TicketsDir and returns the new path.
//
// MoveToActive preserves the source basename and refuses to overwrite an
// existing destination file.
func (s *Store) MoveToActive(path string) (string, error) {
	dest := filepath.Join(s.TicketsDir, filepath.Base(path))
	if _, err := os.Stat(dest); err == nil {
		return "", fmt.Errorf("store: move to active: destination already exists: %s", filepath.Base(dest))
	}

	if err := os.Rename(path, dest); err != nil {
		return "", fmt.Errorf("store: move to active: rename: %w", err)
	}
	return dest, nil
}
