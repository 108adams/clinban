package store

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

// idPattern matches filenames of the form [0-9]{4}-*.md and captures the
// 4-digit numeric prefix.
var idPattern = regexp.MustCompile(`^([0-9]{4})-.*\.md$`)

// scanDir collects all numeric ID prefixes from *.md files in dir whose names
// match idPattern. If dir does not exist, it is treated as empty (no error).
func scanDir(dir string) ([]int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("store: scan %s: %w", dir, err)
	}

	var ids []int
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := idPattern.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			// Shouldn't happen given the regex, but guard anyway.
			continue
		}
		ids = append(ids, n)
	}
	return ids, nil
}

// scanDirIDs collects all 4-digit ID prefix strings (zero-padded) from *.md
// files in dir. If dir does not exist, it is treated as empty (no error).
func scanDirIDs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("store: scan ids %s: %w", dir, err)
	}

	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := idPattern.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		ids = append(ids, m[1])
	}
	return ids, nil
}

// NextID scans TicketsDir and ArchiveDir for the highest numeric filename
// prefix matching [0-9]{4}-*.md, and returns that value + 1.
// Returns 1 if no matching files exist.
func (s *Store) NextID() (int, error) {
	active, err := scanDir(s.TicketsDir)
	if err != nil {
		return 0, err
	}
	archive, err := scanDir(s.ArchiveDir)
	if err != nil {
		return 0, err
	}

	max := 0
	for _, n := range active {
		if n > max {
			max = n
		}
	}
	for _, n := range archive {
		if n > max {
			max = n
		}
	}
	return max + 1, nil
}

// FindByID locates a ticket file by its 4-digit ID prefix string.
// Searches TicketsDir first, then ArchiveDir.
// Returns the full path, whether it is in the archive, and any error.
// Returns ("", false, ErrNotFound) if the ID does not exist.
func (s *Store) FindByID(id string) (path string, inArchive bool, err error) {
	for _, dir := range []string{s.TicketsDir, s.ArchiveDir} {
		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				continue
			}
			return "", false, fmt.Errorf("store: find %s: %w", id, readErr)
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			m := idPattern.FindStringSubmatch(e.Name())
			if m == nil {
				continue
			}
			if m[1] == id {
				full := filepath.Join(dir, e.Name())
				return full, dir == s.ArchiveDir, nil
			}
		}
	}
	return "", false, ErrNotFound
}

// AllIDs returns every ID string (numeric prefix) found in *.md files in
// both TicketsDir and ArchiveDir. Used by lint for uniqueness checking.
func (s *Store) AllIDs() ([]string, error) {
	active, err := scanDirIDs(s.TicketsDir)
	if err != nil {
		return nil, err
	}
	archive, err := scanDirIDs(s.ArchiveDir)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(active)+len(archive))
	result = append(result, active...)
	result = append(result, archive...)
	return result, nil
}

// listDir reads all *.md files in dir, parses each as a Ticket, and returns
// Records with InArchive set to the given flag. Non-matching filenames are
// silently skipped. If dir does not exist, returns empty slice (no error).
func (s *Store) listDir(dir string, inArchive bool) ([]Record, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Record{}, nil
		}
		return nil, fmt.Errorf("store: list %s: %w", dir, err)
	}

	records := make([]Record, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) != ".md" {
			continue
		}
		full := filepath.Join(dir, e.Name())
		t, err := s.ReadTicket(full)
		if err != nil {
			return nil, fmt.Errorf("store: list: read %s: %w", e.Name(), err)
		}
		records = append(records, Record{
			Ticket:    t,
			Path:      full,
			InArchive: inArchive,
		})
	}
	return records, nil
}

// ListActive returns all tickets in TicketsDir as Records.
// Returns an empty (never nil) slice if the directory is empty or absent.
func (s *Store) ListActive() ([]Record, error) {
	return s.listDir(s.TicketsDir, false)
}

// ListArchive returns all tickets in ArchiveDir as Records.
// Returns an empty (never nil) slice if the directory is empty or absent.
func (s *Store) ListArchive() ([]Record, error) {
	return s.listDir(s.ArchiveDir, true)
}
