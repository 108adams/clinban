package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove [id]",
	Short: "Delete a ticket file from disk",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemove,
}

func init() {
	rootCmd.AddCommand(removeCmd)
}

func runRemove(_ *cobra.Command, args []string) error {
	id := args[0]
	paths, err := st.FindAllByID(id)
	if err != nil {
		return fmt.Errorf("remove: find ticket: %w", err)
	}
	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "ticket not found")
		os.Exit(1)
	}
	if len(paths) > 1 {
		fmt.Fprintf(os.Stderr, "multiple files share ID %s:\n", id)
		for _, p := range paths {
			fmt.Fprintf(os.Stderr, "  %s\n", filepath.Base(p))
		}
		fmt.Fprintln(os.Stderr, "run 'clinban lint' to identify and resolve collisions")
		os.Exit(1)
	}
	path := paths[0]
	if err := st.Remove(path); err != nil {
		return fmt.Errorf("remove: %w", err)
	}
	fmt.Fprintf(os.Stdout, "removed: %s\n", filepath.Base(path))
	return nil
}
