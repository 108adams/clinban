package store

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/108adams/clinban/internal/ticket"
)

// ReadTicket reads path and parses it as a Clinban ticket.
//
// The returned error wraps either the filesystem read error or the ticket parse
// error. ReadTicket does not run lint; callers that need schema validation
// should call package lint after a successful read.
func (s *Store) ReadTicket(path string) (*ticket.Ticket, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("store: read ticket %s: %w", path, err)
	}
	t, err := ticket.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("store: parse ticket %s: %w", path, err)
	}
	base := filepath.Base(path)
	m := idPattern.FindStringSubmatch(base)
	if m == nil {
		return nil, fmt.Errorf("store: read ticket: filename %q is not a managed ticket", base)
	}
	t.ID = m[1]
	return t, nil
}

// WriteTicket serialises t and writes it to path using a same-directory
// temporary file followed by rename.
//
// The temporary file is created in the target directory so the final rename is
// on the same filesystem. WriteTicket does not modify t; callers are responsible
// for setting system-owned fields such as Updated before calling.
func (s *Store) WriteTicket(t *ticket.Ticket, path string) error {
	b, err := ticket.Marshal(t)
	if err != nil {
		return fmt.Errorf("store: write ticket: marshal: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".clinban-*.tmp")
	if err != nil {
		return fmt.Errorf("store: write ticket: create temp: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("store: write ticket: write temp: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("store: write ticket: chmod temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("store: write ticket: sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("store: write ticket: close temp: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("store: write ticket: rename: %w", err)
	}
	if dir, err := os.Open(filepath.Dir(path)); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return nil
}

// TicketPath returns the canonical active path for id and slug.
//
// The filename format is <id>-<slug>.md, with id rendered as a zero-padded
// four-digit decimal number.
func (s *Store) TicketPath(id int, slug string) string {
	return filepath.Join(s.TicketsDir, fmt.Sprintf("%04d-%s.md", id, slug))
}

// ActivePath returns the active-directory path for archivePath's basename.
//
// It is used when moving a ticket from archive back to active while preserving
// the existing filename.
func (s *Store) ActivePath(archivePath string) string {
	return filepath.Join(s.TicketsDir, filepath.Base(archivePath))
}
