package store

import (
	"errors"

	"clinban/internal/config"
	"clinban/internal/ticket"
)

// ErrNotFound is returned by FindByID when no ticket file matches the given ID.
var ErrNotFound = errors.New("ticket not found")

// Store manages ticket files on disk. It knows the locations of the active
// tickets directory and the archive directory.
type Store struct {
	TicketsDir string
	ArchiveDir string
}

// New constructs a Store from the provided configuration.
func New(cfg *config.Config) *Store {
	return &Store{
		TicketsDir: cfg.TicketsDir,
		ArchiveDir: cfg.ArchiveDir,
	}
}

// Record pairs a parsed Ticket with its filesystem location and archive status.
type Record struct {
	Ticket    *ticket.Ticket
	Path      string
	InArchive bool
}
