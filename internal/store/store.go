package store

import (
	"errors"

	"clinban/internal/config"
	"clinban/internal/ticket"
)

// ErrNotFound is returned by FindByID when no active or archived ticket file
// matches the requested ID.
var ErrNotFound = errors.New("ticket not found")

// Store manages ticket files on disk.
//
// Store owns filesystem concerns only: locating, reading, writing, listing, and
// moving ticket files. It does not enforce schema validity or workflow
// transitions.
type Store struct {
	// TicketsDir is the directory containing active ticket files.
	TicketsDir string
	// ArchiveDir is the directory containing archived ticket files.
	ArchiveDir string
}

// New constructs a Store from cfg.
func New(cfg *config.Config) *Store {
	return &Store{
		TicketsDir: cfg.TicketsDir,
		ArchiveDir: cfg.ArchiveDir,
	}
}

// Record pairs a parsed ticket with its filesystem location.
type Record struct {
	// Ticket is the parsed ticket content.
	Ticket *ticket.Ticket
	// Path is the full filesystem path from which Ticket was read.
	Path string
	// InArchive reports whether Path is under the configured archive directory.
	InArchive bool
}
