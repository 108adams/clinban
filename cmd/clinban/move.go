package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"clinban/internal/fsm"
	"clinban/internal/store"
	"clinban/internal/ticket"
)

var moveCmd = &cobra.Command{
	Use:   "move <id> <status>",
	Short: "Transition a ticket to a new status",
	Long: `Move transitions a ticket from its current status to the specified target status.

The transition must be permitted by the FSM rules. If the ticket is already in
the target status the command exits silently with code 0.

Special case: moving a done ticket back to backlog also moves the ticket file
from the archive directory back to the active tickets directory.`,
	Args: cobra.ExactArgs(2),
	RunE: runMove,
}

func init() {
	rootCmd.AddCommand(moveCmd)
}

// runMove applies a workflow transition to one ticket.
//
// The command validates the requested target status, enforces the state
// machine, updates the ticket's status and updated timestamp, and handles the
// archive-to-active reopen path for done -> backlog.
func runMove(_ *cobra.Command, args []string) error {
	id := args[0]
	targetStr := args[1]

	// Validate the target status string before doing any I/O.
	target := ticket.Status(targetStr)
	if !target.Valid() {
		fmt.Fprintf(os.Stderr, "invalid status %q: must be one of backlog, in-progress, blocked, done\n", targetStr)
		os.Exit(1)
	}

	// Locate the ticket file.
	path, inArchive, err := st.FindByID(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			fmt.Fprintln(os.Stderr, "ticket not found")
			os.Exit(1)
		}
		return fmt.Errorf("move: find ticket: %w", err)
	}

	// Parse the ticket.
	t, err := st.ReadTicket(path)
	if err != nil {
		return fmt.Errorf("move: read ticket: %w", err)
	}

	// No-op when current == target.
	if t.Status == target {
		return nil
	}

	// Validate the FSM transition.
	if err := fsm.ValidateTransition(t.Status, target); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Special case: done → backlog moves the file back to the active directory.
	// Write to the active path first so that a write failure leaves the archive
	// intact (the file is not moved until the write succeeds).
	if inArchive && t.Status == ticket.StatusDone && target == ticket.StatusBacklog {
		t.Status = target
		t.Updated = time.Now()
		activePath := st.ActivePath(path)
		if err := st.WriteTicket(t, activePath); err != nil {
			return fmt.Errorf("move: write to active: %w", err)
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("move: remove from archive: %w", err)
		}
		fmt.Fprintf(os.Stdout, "%s moved to %s\n", id, target)
		return nil
	}

	// Standard transition: update fields and write in place.
	t.Status = target
	t.Updated = time.Now()

	if err := st.WriteTicket(t, path); err != nil {
		return fmt.Errorf("move: write ticket: %w", err)
	}

	fmt.Fprintf(os.Stdout, "%s moved to %s\n", id, target)
	return nil
}
