package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"clinban/internal/lint"
	"clinban/internal/store"
)

var lintCmd = &cobra.Command{
	Use:   "lint [id]",
	Short: "Validate ticket files against the schema",
	Long: `Lint validates ticket files against the Clinban schema.

Without an argument, all tickets in the active and archive directories are checked.
With a single ID argument, only that ticket is checked.

Exit code is 0 when no errors are found, 1 when any errors are found or the ID
is unknown.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLint,
}

func init() {
	rootCmd.AddCommand(lintCmd)
}

// runLint is the handler for the lint subcommand.
func runLint(_ *cobra.Command, args []string) error {
	if len(args) == 1 {
		return runLintSingle(args[0])
	}
	return runLintAll()
}

// runLintSingle lints a single ticket identified by id.
func runLintSingle(id string) error {
	path, _, err := st.FindByID(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			fmt.Fprintln(os.Stderr, "ticket not found")
			os.Exit(1)
		}
		return fmt.Errorf("lint: find ticket: %w", err)
	}

	t, err := st.ReadTicket(path)
	if err != nil {
		return fmt.Errorf("lint: read ticket: %w", err)
	}

	allIDs, err := st.AllIDs()
	if err != nil {
		return fmt.Errorf("lint: collect ids: %w", err)
	}

	filename := filepath.Base(path)
	errs := lint.Lint(t, filename, allIDs)
	return reportLintErrors(errs)
}

// runLintAll lints all tickets in the active and archive directories.
func runLintAll() error {
	allIDs, err := st.AllIDs()
	if err != nil {
		return fmt.Errorf("lint: collect ids: %w", err)
	}

	active, err := st.ListActive()
	if err != nil {
		return fmt.Errorf("lint: list active: %w", err)
	}

	archive, err := st.ListArchive()
	if err != nil {
		return fmt.Errorf("lint: list archive: %w", err)
	}

	var allErrors []lint.LintError
	for _, r := range append(active, archive...) {
		filename := filepath.Base(r.Path)
		errs := lint.Lint(r.Ticket, filename, allIDs)
		allErrors = append(allErrors, errs...)
	}

	return reportLintErrors(allErrors)
}

// reportLintErrors prints each LintError to stdout and exits 1 when there are
// any errors. Returns nil (exit 0) when the slice is empty.
func reportLintErrors(errs []lint.LintError) error {
	if len(errs) == 0 {
		return nil
	}
	for _, e := range errs {
		fmt.Fprintln(os.Stdout, e.String())
	}
	os.Exit(1)
	return nil // unreachable; satisfies compiler
}
