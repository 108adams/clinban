package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"clinban/internal/editor"
	"clinban/internal/lint"
	"clinban/internal/store"
	"clinban/internal/ticket"
)

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Open a ticket in $EDITOR",
	Long: `Open a ticket file in $EDITOR (fallback: vi). On editor close, the file is
re-parsed and linted. If both pass, the updated timestamp is refreshed and the
file is written atomically. If parse or lint fails, the user is prompted to
re-open the editor. Exiting the prompt without fixing errors exits with code 1.`,
	Args: cobra.ExactArgs(1),
	RunE: runEdit,
}

func init() {
	rootCmd.AddCommand(editCmd)
}

func runEdit(cmd *cobra.Command, args []string) error {
	id := args[0]

	path, _, err := st.FindByID(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			fmt.Fprintln(os.Stderr, "ticket not found")
			os.Exit(1)
		}
		return fmt.Errorf("edit: find ticket: %w", err)
	}

	filename := filepath.Base(path)

	for {
		if err := editor.Open(path); err != nil {
			return fmt.Errorf("edit: open editor: %w", err)
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("edit: read file: %w", err)
		}

		t, parseErr := ticket.Parse(raw)
		if parseErr != nil {
			fmt.Fprintln(os.Stderr, parseErr.Error())
			if !promptReopen() {
				os.Exit(1)
			}
			continue
		}

		allIDs, err := st.AllIDs()
		if err != nil {
			return fmt.Errorf("edit: collect ids: %w", err)
		}

		lintErrs := lint.Lint(t, filename, allIDs)
		if len(lintErrs) > 0 {
			for _, e := range lintErrs {
				fmt.Fprintln(os.Stderr, e.String())
			}
			if !promptReopen() {
				os.Exit(1)
			}
			continue
		}

		t.Updated = time.Now()
		if err := st.WriteTicket(t, path); err != nil {
			return fmt.Errorf("edit: write ticket: %w", err)
		}

		return nil
	}
}

func promptReopen() bool {
	fmt.Fprint(os.Stderr, "Re-open in editor? [y/N] ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		return strings.EqualFold(line, "y")
	}
	return false
}
