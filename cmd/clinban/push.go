package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/108adams/clinban/internal/fsm"
	"github.com/108adams/clinban/internal/store"
)

var pushCmd = &cobra.Command{
	Use:          "push <id>",
	Short:        "Advance a ticket to its next status",
	SilenceUsage: true,
	Args:         cobra.ExactArgs(1),
	RunE:         runPush,
}

func init() {
	rootCmd.AddCommand(pushCmd)
}

func runPush(_ *cobra.Command, args []string) error {
	id := args[0]

	path, _, err := st.FindByID(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			fmt.Fprintln(os.Stderr, "ticket not found")
			return ExitError{Code: 1, Err: fmt.Errorf("ticket not found")}
		}
		return fmt.Errorf("push: find ticket: %w", err)
	}

	t, err := st.ReadTicket(path)
	if err != nil {
		return fmt.Errorf("push: read ticket: %w", err)
	}

	next, ok := fsm.NextStatus(t.Status)
	if !ok {
		fmt.Fprintf(os.Stdout, "ticket %s is already at the final status (%s)\n", t.ID, t.Status)
		return nil
	}

	t.Status = next
	t.Updated = time.Now()
	if err := st.WriteTicket(t, path); err != nil {
		return fmt.Errorf("push: write ticket: %w", err)
	}

	fmt.Fprintf(os.Stdout, "ticket %s moved to %s\n", t.ID, next)
	return nil
}
