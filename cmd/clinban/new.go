package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"clinban/internal/lint"
	"clinban/internal/slug"
	"clinban/internal/ticket"
)

// newFlags holds the parsed flag values for the new subcommand.
type newFlags struct {
	title         string
	ticketType    string
	body          string
	tags          string
	noInteractive bool
}

var newFlagValues newFlags

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new ticket",
	Long: `Create a new kanban ticket.

By default opens an editor with a pre-populated template (interactive mode).
Use --no-interactive together with --title and --type to create a ticket
from flags without opening an editor.`,
	SilenceUsage: true,
	RunE:         runNew,
}

func init() {
	newCmd.Flags().StringVar(&newFlagValues.title, "title", "", "Ticket title (required for --no-interactive)")
	newCmd.Flags().StringVar(&newFlagValues.ticketType, "type", "", "Ticket type: bug, task, feature, spike (required for --no-interactive)")
	newCmd.Flags().StringVar(&newFlagValues.body, "body", "", "Ticket body (markdown)")
	newCmd.Flags().StringVar(&newFlagValues.tags, "tags", "", "Comma-separated list of tags")
	newCmd.Flags().BoolVar(&newFlagValues.noInteractive, "no-interactive", false, "Create ticket from flags without opening an editor")
	rootCmd.AddCommand(newCmd)
}

// runNew dispatches to the interactive or non-interactive creation path.
func runNew(_ *cobra.Command, _ []string) error {
	if newFlagValues.noInteractive {
		return runNewNonInteractive(newFlagValues)
	}
	return runNewInteractive()
}

// runNewInteractive is the interactive ticket-creation path (T-17).
// Placeholder — not yet implemented.
func runNewInteractive() error {
	fmt.Fprintln(os.Stderr, "not yet implemented")
	os.Exit(1)
	return nil // unreachable
}

// runNewNonInteractive creates a ticket from flags without opening an editor.
func runNewNonInteractive(flags newFlags) error {
	// Validate required flags.
	if flags.title == "" {
		fmt.Fprintln(os.Stderr, "error: --title is required")
		os.Exit(1)
	}
	if flags.ticketType == "" {
		fmt.Fprintln(os.Stderr, "error: --type is required")
		os.Exit(1)
	}

	// Validate --type value.
	tt := ticket.Type(flags.ticketType)
	if !tt.Valid() {
		fmt.Fprintf(os.Stderr, "error: --type must be one of: bug, task, feature, spike (got %q)\n", flags.ticketType)
		os.Exit(1)
	}

	// Assign next ID.
	nextID, err := st.NextID()
	if err != nil {
		return fmt.Errorf("new: get next id: %w", err)
	}
	idStr := fmt.Sprintf("%04d", nextID)

	// Parse optional tags.
	var tagList []string
	if flags.tags != "" {
		for _, tag := range strings.Split(flags.tags, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tagList = append(tagList, tag)
			}
		}
	}
	if tagList == nil {
		tagList = []string{}
	}

	now := time.Now()

	// Build ticket struct.
	t := &ticket.Ticket{
		ID:      idStr,
		Status:  ticket.StatusBacklog,
		Type:    tt,
		Title:   flags.title,
		Tags:    tagList,
		Created: now,
		Updated: now,
		Body:    flags.body,
	}

	// Run lint.
	allIDs, err := st.AllIDs()
	if err != nil {
		return fmt.Errorf("new: collect ids: %w", err)
	}

	titleSlug := slug.Slugify(flags.title)
	path := st.TicketPath(nextID, titleSlug)
	filename := fmt.Sprintf("%s-%s.md", idStr, titleSlug)

	// Append the new ticket's ID to the allIDs list so rule 7 sees it.
	// (It is not yet on disk, so AllIDs would not include it.)
	allIDsWithNew := append(allIDs, idStr) //nolint:gocritic // intentional copy via append

	lintErrs := lint.Lint(t, filename, allIDsWithNew)
	if len(lintErrs) > 0 {
		for _, e := range lintErrs {
			fmt.Fprintln(os.Stderr, e.String())
		}
		os.Exit(1)
	}

	// Write ticket atomically.
	if err := st.WriteTicket(t, path); err != nil {
		return fmt.Errorf("new: write ticket: %w", err)
	}

	// Report success.
	fmt.Printf("created: %s\n", filename)
	return nil
}
