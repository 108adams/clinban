package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/108adams/clinban/internal/lint"
	"github.com/108adams/clinban/internal/slug"
	"github.com/108adams/clinban/internal/ticket"
)

var registerCmd = &cobra.Command{
	Use:   "register <path>",
	Short: "Adopt an externally created ticket file into the registry",
	Long: `Register reads the ticket file at <path>, assigns it a system ID and
timestamps, validates it with lint, then moves it into the tickets directory.

If the file fails to parse or has lint errors, it is not moved and an error
is printed to stderr with exit code 1.`,
	Args: cobra.ExactArgs(1),
	RunE: runRegister,
}

func init() {
	rootCmd.AddCommand(registerCmd)
}

// runRegister adopts an externally authored ticket file into the active
// Clinban registry.
//
// Registration overwrites system-owned fields, validates the resulting ticket,
// writes it to its canonical filename, and removes the source file after a
// successful write.
func runRegister(_ *cobra.Command, args []string) error {
	srcPath := args[0]

	// Step 1 — read the file; exit 1 if not found.
	content, err := os.ReadFile(srcPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(os.Stderr, "file not found")
			os.Exit(1)
		}
		return fmt.Errorf("register: read file: %w", err)
	}

	// Step 2 — preliminary parse to extract the title for filename construction.
	// ticket.Parse is called again inside ValidateForCommit; this first call
	// is necessary to build the canonical filename before lint can run.
	preliminary, err := ticket.Parse(content)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// Step 3 — overwrite system-owned fields: assign the next system ID and
	// compute the canonical filename from the (already-extracted) title.
	nextID, err := st.NextID()
	if err != nil {
		return fmt.Errorf("register: next id: %w", err)
	}
	id := fmt.Sprintf("%04d", nextID)

	sl := slug.Slugify(preliminary.Title)
	finalPath := st.TicketPath(nextID, sl)
	finalFilename := filepath.Base(finalPath)

	// Step 4 — collect all existing IDs and run the parse+ID+lint kernel via
	// ValidateForCommit. The kernel re-parses content, assigns the system id,
	// and runs all lint rules against the canonical filename.
	allIDs, err := st.AllIDs()
	if err != nil {
		return fmt.Errorf("register: collect ids: %w", err)
	}

	t, lintErrs, parseErr := lint.ValidateForCommit(content, id, finalFilename, allIDs)
	if parseErr != nil {
		// parse already succeeded above; this branch is unreachable in
		// practice but we handle it for correctness.
		fmt.Fprintf(os.Stderr, "%v\n", parseErr)
		os.Exit(1)
	}
	if len(lintErrs) > 0 {
		for _, e := range lintErrs {
			fmt.Fprintln(os.Stderr, e.String())
		}
		os.Exit(1)
	}

	// Step 5 — path containment check (prevents path traversal).
	// Ensure that the computed finalPath is within the store's TicketsDir.
	// We resolve both paths before comparing to handle any symlinks or ".." components.
	ticketsDir := st.TicketsDir
	rel, relErr := filepath.Rel(ticketsDir, finalPath)
	if relErr != nil || strings.HasPrefix(rel, "..") {
		fmt.Fprintln(os.Stderr, "register: computed path is outside tickets directory")
		os.Exit(1)
	}

	// Step 6 — overwrite system-owned timestamp fields and write atomically.
	now := time.Now()
	t.Created = now
	t.Updated = now
	if err := st.WriteTicket(t, finalPath); err != nil {
		return fmt.Errorf("register: write ticket: %w", err)
	}

	// Step 7 — delete the source file if it differs from the final path.
	srcAbs, err := filepath.Abs(srcPath)
	if err != nil {
		srcAbs = srcPath
	}
	finalAbs, err := filepath.Abs(finalPath)
	if err != nil {
		finalAbs = finalPath
	}
	if srcAbs != finalAbs {
		if err := os.Remove(srcPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("register: remove source file: %w", err)
		}
	}

	// Step 8 — report success.
	fmt.Printf("registered: %s\n", finalFilename)
	return nil
}
