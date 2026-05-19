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

// runEdit opens a ticket in an editor and commits the edit only after parse and
// lint both succeed.
//
// The live ticket is copied to a same-directory scratch file before the editor
// opens. Invalid edits never replace the original unless the user reopens and
// fixes them.
func runEdit(cmd *cobra.Command, args []string) error {
	id := args[0]

	livePath, _, err := st.FindByID(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			fmt.Fprintln(os.Stderr, "ticket not found")
			os.Exit(1)
		}
		return fmt.Errorf("edit: find ticket: %w", err)
	}

	filename := filepath.Base(livePath)

	// Copy the live file to a temp in the same directory so the user edits
	// a scratch copy. The original is only replaced when parse+lint pass.
	original, err := os.ReadFile(livePath)
	if err != nil {
		return fmt.Errorf("edit: read original: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(livePath), ".clinban-edit-*.md")
	if err != nil {
		return fmt.Errorf("edit: create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // best-effort cleanup; ignored if already renamed

	if _, err := tmp.Write(original); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("edit: write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("edit: close temp: %w", err)
	}

	for {
		if err := editor.Open(tmpPath); err != nil {
			return fmt.Errorf("edit: open editor: %w", err)
		}

		raw, err := os.ReadFile(tmpPath)
		if err != nil {
			return fmt.Errorf("edit: read temp: %w", err)
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

		// Parse+lint passed — write updated timestamp and atomically replace the original.
		t.Updated = time.Now()
		if err := st.WriteTicket(t, livePath); err != nil {
			return fmt.Errorf("edit: write ticket: %w", err)
		}

		return nil
	}
}

// promptReopen asks whether an invalid interactive edit should be reopened.
func promptReopen() bool {
	fmt.Fprint(os.Stderr, "Re-open in editor? [y/N] ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		return strings.EqualFold(line, "y")
	}
	return false
}
