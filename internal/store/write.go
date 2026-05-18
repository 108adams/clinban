package store

import (
	"fmt"
	"os"
	"path/filepath"

	"clinban/internal/ticket"
)

// ReadTicket reads and parses the ticket file at the given path.
func (s *Store) ReadTicket(path string) (*ticket.Ticket, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("store: read ticket %s: %w", path, err)
	}
	t, err := ticket.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("store: parse ticket %s: %w", path, err)
	}
	return t, nil
}

// WriteTicket serialises t and writes it atomically to path.
// It writes to a temp file (path + ".tmp") in the same directory, then
// renames it to the final path. Caller must set t.Updated before calling.
func (s *Store) WriteTicket(t *ticket.Ticket, path string) error {
	b, err := ticket.Marshal(t)
	if err != nil {
		return fmt.Errorf("store: write ticket: marshal: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, b, 0o600); err != nil {
		return fmt.Errorf("store: write ticket: write temp: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		// Best-effort cleanup of the temp file; ignore cleanup error.
		_ = os.Remove(tmpPath)
		return fmt.Errorf("store: write ticket: rename: %w", err)
	}
	return nil
}

// TicketPath returns the canonical path for a ticket in TicketsDir,
// formatted as <TicketsDir>/<id>-<slug>.md where id is zero-padded to 4 digits.
func (s *Store) TicketPath(id int, slug string) string {
	return filepath.Join(s.TicketsDir, fmt.Sprintf("%04d-%s.md", id, slug))
}
