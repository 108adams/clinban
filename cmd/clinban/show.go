package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"clinban/internal/store"
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Print a ticket to stdout in human-readable format",
	Long: `Show prints the fields and body of a ticket to stdout.

The ticket is located by its 4-digit ID. Both the active directory and the
archive directory are searched. If the ticket lives in the archive, an
[archived] label is appended after the timestamps.

Exit code is 0 on success, 1 if the ID is not found.`,
	Args: cobra.ExactArgs(1),
	RunE: runShow,
}

func init() {
	rootCmd.AddCommand(showCmd)
}

// runShow is the handler for the show subcommand.
func runShow(_ *cobra.Command, args []string) error {
	id := args[0]

	path, inArchive, err := st.FindByID(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			fmt.Fprintln(os.Stderr, "ticket not found")
			os.Exit(1)
		}
		return fmt.Errorf("show: find ticket: %w", err)
	}

	t, err := st.ReadTicket(path)
	if err != nil {
		return fmt.Errorf("show: read ticket: %w", err)
	}

	// Required fields printed in documented order.
	fmt.Fprintf(os.Stdout, "ID:      %s\n", t.ID)
	fmt.Fprintf(os.Stdout, "Status:  %s\n", t.Status)
	fmt.Fprintf(os.Stdout, "Type:    %s\n", t.Type)
	fmt.Fprintf(os.Stdout, "Title:   %s\n", t.Title)

	// Tags line is omitted when Tags is empty.
	if len(t.Tags) > 0 {
		fmt.Fprintf(os.Stdout, "Tags:    %s\n", strings.Join(t.Tags, ", "))
	}

	fmt.Fprintf(os.Stdout, "Created: %s\n", t.Created.Format("2006-01-02T15:04:05Z07:00"))
	fmt.Fprintf(os.Stdout, "Updated: %s\n", t.Updated.Format("2006-01-02T15:04:05Z07:00"))

	// Archived label only when the ticket lives in the archive directory.
	if inArchive {
		fmt.Fprintln(os.Stdout, "[archived]")
	}

	// Blank line + body when the body is non-empty.
	if t.Body != "" {
		fmt.Fprintf(os.Stdout, "\n%s", t.Body)
	}

	return nil
}
