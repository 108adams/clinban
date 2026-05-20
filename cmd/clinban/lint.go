package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/108adams/clinban/internal/lint"
	"github.com/108adams/clinban/internal/store"
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

// runLint dispatches lint to whole-repository or single-ticket validation.
func runLint(_ *cobra.Command, args []string) error {
	if len(args) == 1 {
		return runLintSingle(args[0])
	}
	return runLintAll()
}

// runLintSingle validates one active or archived ticket by ID.
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

// runLintAll validates every managed ticket in the active and archive
// directories.
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

// reportLintErrors prints lint errors in the CLI's canonical output format.
//
// Lint violations are normal command output, so they are written to stdout. An
// empty slice means validation succeeded.
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
