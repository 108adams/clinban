package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"clinban/internal/store"
	"clinban/internal/ticket"
)

// statusOrder defines the list display order. Lower values sort earlier.
var statusOrder = map[ticket.Status]int{
	ticket.StatusInProgress: 0,
	ticket.StatusBlocked:    1,
	ticket.StatusBacklog:    2,
	ticket.StatusDone:       3,
}

// listFlags holds the parsed flag values for the list command.
type listFlags struct {
	status string
	typ    string
	tag    string
}

var listOpts listFlags

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List active tickets",
	Long: `List all active tickets (those not in the archive directory).

Optionally filter by status, type, or tag. Multiple filters combine with AND logic.
Output is sorted in-progress → blocked → backlog → done, then by ID ascending
within each group. Each line is truncated to fit the terminal width.`,
	Args: cobra.NoArgs,
	RunE: runList,
}

func init() {
	listCmd.Flags().StringVar(&listOpts.status, "status", "", "Filter by status (backlog, in-progress, blocked, done)")
	listCmd.Flags().StringVar(&listOpts.typ, "type", "", "Filter by type (bug, task, feature, spike)")
	listCmd.Flags().StringVar(&listOpts.tag, "tag", "", "Filter by tag (exact match)")
	rootCmd.AddCommand(listCmd)
}

// runList prints active tickets, optionally filtered by status, type, and tag.
func runList(_ *cobra.Command, _ []string) error {
	records, err := st.ListActive()
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}

	filtered, err := applyFilters(records, listOpts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if len(filtered) == 0 {
		fmt.Fprintln(os.Stdout, "No active tickets")
		return nil
	}

	sortRecords(filtered)

	width := terminalWidth()
	for _, r := range filtered {
		fmt.Fprintln(os.Stdout, formatRecord(r, width))
	}
	return nil
}

// applyFilters returns records matching all non-empty list flags.
//
// Status and type filters are validated before filtering so the command fails
// clearly on misspelled controlled-vocabulary values.
func applyFilters(records []store.Record, opts listFlags) ([]store.Record, error) {
	// Validate flag values upfront.
	if opts.status != "" {
		s := ticket.Status(opts.status)
		if !s.Valid() {
			return nil, fmt.Errorf("invalid status %q: must be one of backlog, in-progress, blocked, done", opts.status)
		}
	}
	if opts.typ != "" {
		tp := ticket.Type(opts.typ)
		if !tp.Valid() {
			return nil, fmt.Errorf("invalid type %q: must be one of bug, task, feature, spike", opts.typ)
		}
	}

	out := make([]store.Record, 0, len(records))
	for _, r := range records {
		if opts.status != "" && string(r.Ticket.Status) != opts.status {
			continue
		}
		if opts.typ != "" && string(r.Ticket.Type) != opts.typ {
			continue
		}
		if opts.tag != "" && !hasTag(r.Ticket.Tags, opts.tag) {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

// hasTag reports whether tags contains the target string (exact match).
func hasTag(tags []string, target string) bool {
	for _, t := range tags {
		if t == target {
			return true
		}
	}
	return false
}

// sortRecords sorts records in the documented board order, then by numeric ID.
func sortRecords(records []store.Record) {
	sort.SliceStable(records, func(i, j int) bool {
		oi := statusOrder[records[i].Ticket.Status]
		oj := statusOrder[records[j].Ticket.Status]
		if oi != oj {
			return oi < oj
		}
		// Within the same status group, sort ascending by numeric ID.
		ni, _ := strconv.Atoi(records[i].Ticket.ID)
		nj, _ := strconv.Atoi(records[j].Ticket.ID)
		return ni < nj
	})
}

// terminalWidth returns the current terminal column count, or 80 if stdout is
// not a terminal or the width cannot be determined.
func terminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}

// formatRecord formats one list row and truncates the title to width.
//
// The row format is:
//
//	<id>  <status>  <type>  <title>
func formatRecord(r store.Record, width int) string {
	id := r.Ticket.ID
	status := string(r.Ticket.Status)
	typ := string(r.Ticket.Type)
	title := r.Ticket.Title

	// Build the prefix without the title: "<id>  <status>  <type>  "
	prefix := id + "  " + status + "  " + typ + "  "

	// Calculate how many runes are available for the title.
	prefixLen := utf8.RuneCountInString(prefix)
	available := width - prefixLen
	if available <= 0 {
		// No room for title at all — still emit the prefix.
		return strings.TrimRight(prefix, " ")
	}

	runes := []rune(title)
	if len(runes) > available {
		// Truncate and add ellipsis if there is room.
		if available > 1 {
			runes = runes[:available-1]
			title = string(runes) + "…"
		} else {
			title = string(runes[:available])
		}
	}

	return prefix + title
}
