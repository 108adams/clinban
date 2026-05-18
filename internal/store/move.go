package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// MoveToArchive moves the file at path into ArchiveDir.
// Creates ArchiveDir if it does not exist.
// Returns the new path in the archive.
func (s *Store) MoveToArchive(path string) (string, error) {
	if err := os.MkdirAll(s.ArchiveDir, 0o755); err != nil {
		return "", fmt.Errorf("store: move to archive: mkdir: %w", err)
	}

	dest := filepath.Join(s.ArchiveDir, filepath.Base(path))
	if err := os.Rename(path, dest); err != nil {
		return "", fmt.Errorf("store: move to archive: rename: %w", err)
	}
	return dest, nil
}

// MoveToActive moves the file at path from ArchiveDir into TicketsDir.
// Returns the new path in the active directory.
func (s *Store) MoveToActive(path string) (string, error) {
	dest := filepath.Join(s.TicketsDir, filepath.Base(path))
	if err := os.Rename(path, dest); err != nil {
		return "", fmt.Errorf("store: move to active: rename: %w", err)
	}
	return dest, nil
}
