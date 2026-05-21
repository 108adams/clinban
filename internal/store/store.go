package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/108adams/clinban/internal/config"
	"github.com/108adams/clinban/internal/ticket"
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

// Remove deletes the ticket file at path from disk.
//
// The path must be an absolute path to an existing file. If the file cannot be
// removed, the error is wrapped with context identifying the filename.
func (s *Store) Remove(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("store: remove %s: %w", filepath.Base(path), err)
	}
	return nil
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
