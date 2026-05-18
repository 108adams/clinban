package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"clinban/internal/store"
	"clinban/internal/ticket"
)

var archiveCmd = &cobra.Command{
	Use:   "archive [id]",
	Short: "Move done tickets to the archive directory",
	Long: `Archive moves done tickets from the active directory to the archive directory.

Without an argument, all tickets with status=done are listed and the user is
prompted to confirm before moving them. With a single ID argument, only that
ticket is archived (it must have status=done).

The archive directory is created automatically if it does not exist.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runArchive,
}

func init() {
	rootCmd.AddCommand(archiveCmd)
}

// runArchive is the handler for the archive subcommand.
func runArchive(_ *cobra.Command, args []string) error {
	if len(args) == 1 {
		return runArchiveSingle(args[0])
	}
	return runArchiveBulk()
}

// runArchiveSingle archives a single ticket identified by id.
// The ticket must have status=done; otherwise an error is printed and the
// process exits with code 1.
func runArchiveSingle(id string) error {
	path, _, err := st.FindByID(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			fmt.Fprintln(os.Stderr, "ticket not found")
			os.Exit(1)
		}
		return fmt.Errorf("archive: find ticket: %w", err)
	}

	t, err := st.ReadTicket(path)
	if err != nil {
		return fmt.Errorf("archive: read ticket: %w", err)
	}

	if t.Status != ticket.StatusDone {
		fmt.Fprintln(os.Stderr, "ticket must be in 'done' status to archive")
		os.Exit(1)
	}

	if _, err := st.MoveToArchive(path); err != nil {
		return fmt.Errorf("archive: move: %w", err)
	}

	fmt.Fprintf(os.Stdout, "archived: %s\n", filepath.Base(path))
	return nil
}

// runArchiveBulk finds all done tickets in the active directory, lists them,
// prompts the user for confirmation, and moves them to the archive directory.
func runArchiveBulk() error {
	records, err := st.ListActive()
	if err != nil {
		return fmt.Errorf("archive: list active: %w", err)
	}

	var done []store.Record
	for _, r := range records {
		if r.Ticket.Status == ticket.StatusDone {
			done = append(done, r)
		}
	}

	if len(done) == 0 {
		fmt.Fprintln(os.Stdout, "No done tickets to archive")
		return nil
	}

	// List the filenames that will be archived.
	for _, r := range done {
		fmt.Fprintf(os.Stdout, "  %s\n", filepath.Base(r.Path))
	}

	// Prompt for confirmation.
	fmt.Fprintf(os.Stdout, "Archive %d ticket(s)? [y/N] ", len(done))

	var input [1]byte
	n, err := os.Stdin.Read(input[:])
	if err != nil || n == 0 {
		// Cannot read from stdin — treat as 'N'.
		fmt.Fprintln(os.Stdout)
		return nil
	}

	ch := input[0]
	if ch != 'y' && ch != 'Y' {
		return nil
	}

	count := 0
	for _, r := range done {
		if _, err := st.MoveToArchive(r.Path); err != nil {
			return fmt.Errorf("archive: move %s: %w", filepath.Base(r.Path), err)
		}
		count++
	}

	fmt.Fprintf(os.Stdout, "archived %d ticket(s)\n", count)
	return nil
}
