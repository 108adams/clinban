package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/108adams/clinban/internal/editor"
	"github.com/108adams/clinban/internal/lint"
	"github.com/108adams/clinban/internal/slug"
	"github.com/108adams/clinban/internal/template"
	"github.com/108adams/clinban/internal/ticket"
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
from flags without opening an editor.

Positional arguments are joined with spaces and appended to the template as
body text before the editor opens. Note: an unquoted '#' preceded by
whitespace is treated as a shell comment and stripped before Go receives the
arguments. To include '#' literally, either escape it ('\#') or quote the
entire string (e.g. clinban new "title # body").`,
	SilenceUsage: true,
	Args:         cobra.ArbitraryArgs,
	RunE:         runNew,
}

func init() {
	newCmd.Flags().StringVar(&newFlagValues.title, "title", "", "Ticket title (required for --no-interactive)")
	newCmd.Flags().StringVar(&newFlagValues.ticketType, "type", "", "Ticket type: bug, task, feature, spike (required for --no-interactive)")
	newCmd.Flags().StringVar(&newFlagValues.body, "body", "", "Ticket body (markdown)")
	newCmd.Flags().StringVar(&newFlagValues.tags, "tags", "", "Comma-separated list of tags")
	newCmd.Flags().BoolVar(&newFlagValues.noInteractive, "no-interactive", false, "Create ticket from flags without opening an editor")
	// Treat everything after the subcommand name as positional args so that
	// body text containing "--flag-like" words is not parsed as unknown flags.
	newCmd.Flags().SetInterspersed(false)
	rootCmd.AddCommand(newCmd)
}

// runNew dispatches to the interactive or non-interactive creation path.
func runNew(_ *cobra.Command, args []string) error {
	if newFlagValues.noInteractive {
		return runNewNonInteractive(newFlagValues)
	}
	return runNewInteractive(strings.Join(args, " "))
}

// runNewInteractive is the interactive ticket-creation path (T-17).
func runNewInteractive(raw string) error {
	// Assign next ID.
	nextID, err := st.NextID()
	if err != nil {
		return fmt.Errorf("new: get next id: %w", err)
	}

	now := time.Now()

	// Split raw on the first '#' when the feature is enabled.
	title := ""
	body := raw
	if cfg.SplitRawNew && raw != "" {
		title, body = splitRawBody(raw)
	}

	// Render the template. Only pass the default type when it is a valid type
	// value; an invalid or empty config value renders as an empty type field.
	defType := cfg.DefaultType
	if !ticket.Type(defType).Valid() {
		defType = ""
	}
	tmplBytes, err := template.New(now, defType, title)
	if err != nil {
		return fmt.Errorf("new: render template: %w", err)
	}

	// Create a temp file in TicketsDir (same filesystem as final destination
	// — required for os.Rename to work atomically per ADR-3).
	tmpFile, err := os.CreateTemp(st.TicketsDir, ".clinban-*.md")
	if err != nil {
		return fmt.Errorf("new: create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Write template content into temp file.
	if _, err := tmpFile.Write(tmplBytes); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("new: write template: %w", err)
	}
	if body != "" {
		if _, err := tmpFile.Write([]byte("\n" + body + "\n")); err != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
			return fmt.Errorf("new: write body: %w", err)
		}
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("new: close temp file: %w", err)
	}

	// Open editor for the temp file.
	if err := editor.Open(tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("new: editor: %w", err)
	}

	// Read file back after editor closes.
	content, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("new: read after editor: %w", err)
	}

	// Parse the ticket from the edited file.
	t, err := ticket.Parse(content)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not parse edited ticket: %v\n", err)
		_ = os.Remove(tmpPath)
		return ExitError{Code: 1, Err: fmt.Errorf("new: parse ticket: %w", err)}
	}
	t.ID = fmt.Sprintf("%04d", nextID)

	// Collect existing IDs before the Rename so the new ticket is not yet
	// visible to the disk scan. AllIDs must be called here — after Rename the
	// new file lands on disk and the subsequent append would double-count its
	// ID, causing ruleIDUnique to fire a false-positive.
	allIDs, err := st.AllIDs()
	if err != nil {
		return fmt.Errorf("new: collect ids: %w", err)
	}
	// Append the new ticket's ID so rule 7 can see it alongside all existing IDs.
	// The ticket is not yet on disk at this point, so AllIDs did not include it.
	idStr := fmt.Sprintf("%04d", nextID)
	allIDsWithNew := append(allIDs, idStr) //nolint:gocritic // intentional copy via append

	// Lint/re-open loop. The ticket is still in the temp file here, so invalid
	// frontmatter cannot create a managed ticket with a bad filename such as
	// "0001-.md".
	for {
		filename := filepath.Base(st.TicketPath(nextID, slug.Slugify(t.Title)))
		lintErrs := lint.Lint(t, filename, allIDsWithNew)
		if len(lintErrs) == 0 {
			break
		}

		for _, e := range lintErrs {
			fmt.Fprintln(os.Stderr, e.String())
		}

		fmt.Fprint(os.Stderr, "Re-open in editor? [y/N] ")
		answer := readLine(os.Stdin)
		if answer != "y" && answer != "Y" {
			_ = os.Remove(tmpPath)
			return ExitError{Code: 1, Err: fmt.Errorf("new: lint failed")}
		}

		if err := editor.Open(tmpPath); err != nil {
			return fmt.Errorf("new: editor re-open: %w", err)
		}

		// Re-read and re-parse.
		content, err = os.ReadFile(tmpPath)
		if err != nil {
			return fmt.Errorf("new: read after re-open: %w", err)
		}
		t, err = ticket.Parse(content)
		if err != nil {
			return fmt.Errorf("new: parse after re-open: %w", err)
		}
		t.ID = fmt.Sprintf("%04d", nextID)
	}

	// Compute final path after the ticket has passed lint, because the title may
	// have changed during the re-open loop.
	titleSlug := slug.Slugify(t.Title)
	finalPath := st.TicketPath(nextID, titleSlug)
	filename := filepath.Base(finalPath)

	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("new: rename to final path: %w", err)
	}

	fmt.Printf("created: %s\n", filename)
	return nil
}

// readLine reads one line from r (trimming trailing newline).
func readLine(r *os.File) string {
	scanner := bufio.NewScanner(r)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

// splitRawBody splits raw on the first '#' character.
// The trimmed left side becomes title; the trimmed right side becomes body.
// When no '#' is present, title is empty and body is the full trimmed raw string.
// When raw is empty, both return values are empty strings.
func splitRawBody(raw string) (title, body string) {
	left, right, found := strings.Cut(raw, "#")
	if !found {
		return "", strings.TrimSpace(raw)
	}
	return strings.TrimSpace(left), strings.TrimSpace(right)
}

// runNewNonInteractive creates a ticket from flags without opening an editor.
func runNewNonInteractive(flags newFlags) error {
	// Validate required flags.
	if flags.title == "" {
		fmt.Fprintln(os.Stderr, "error: --title is required")
		os.Exit(1)
	}
	if flags.ticketType == "" {
		if cfg.DefaultType != "" && ticket.Type(cfg.DefaultType).Valid() {
			flags.ticketType = cfg.DefaultType
		} else {
			fmt.Fprintln(os.Stderr, "error: --type is required")
			os.Exit(1)
		}
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
