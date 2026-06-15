package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/108adams/clinban/internal/store"
)

var resolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Renumber duplicate ticket IDs",
	Long: `Resolve repairs duplicate ticket IDs created by parallel work in separate
repository clones.

It scans active and archived managed ticket files. For each duplicated ID, the
oldest ticket by created timestamp keeps the original ID and younger tickets are
renamed to the next available IDs. Ticket contents are not rewritten.`,
	Args: cobra.NoArgs,
	RunE: runResolve,
}

func init() {
	rootCmd.AddCommand(resolveCmd)
}

type resolveTicket struct {
	file    store.ManagedFile
	created time.Time
}

func runResolve(_ *cobra.Command, _ []string) error {
	files, err := st.ManagedFiles()
	if err != nil {
		return fmt.Errorf("resolve: inventory: %w", err)
	}

	plan, err := planResolve(files)
	if err != nil {
		return err
	}

	if len(plan) == 0 {
		fmt.Fprintln(os.Stdout, "no conflicts found")
		return nil
	}

	newPaths, err := st.BatchRenameWithinDir(plan)
	if err != nil {
		return formatResolveError(err)
	}

	for i, op := range plan {
		fmt.Fprintf(os.Stdout, "renamed: %s -> %s\n", displayPath(op.OldPath), displayPath(newPaths[i]))
	}
	return nil
}

// formatResolveError translates a BatchRenameWithinDir error into the
// CLI-facing message. The store layer never emits the "resolve:" prefix; the
// CLI owns it. A *store.BatchError is rendered as the primary failure line plus
// one "resolve: rollback:" line per residual rollback error. Any other error is
// wrapped with a plain "resolve:" prefix.
func formatResolveError(err error) error {
	var be *store.BatchError
	if !errors.As(err, &be) {
		return fmt.Errorf("resolve: %w", err)
	}

	lines := []string{fmt.Sprintf("resolve: %s %s: %s", be.Failed.Kind, be.Failed.Base, be.Failed.Err)}
	for _, re := range be.Rollback {
		lines = append(lines, fmt.Sprintf("resolve: rollback: %s %s: %s", re.Kind, re.Base, re.Err))
	}
	return errors.New(strings.Join(lines, "\n"))
}

func planResolve(files []store.ManagedFile) ([]store.RenameOp, error) {
	groups := map[string][]store.ManagedFile{}
	for _, file := range files {
		groups[file.ID] = append(groups[file.ID], file)
	}

	nextID, err := st.NextID()
	if err != nil {
		return nil, fmt.Errorf("resolve: next id: %w", err)
	}

	dupNums := make([]int, 0, len(groups))
	dupMap := map[int]string{}
	for id, group := range groups {
		if len(group) < 2 {
			continue
		}
		n, err := strconv.Atoi(id)
		if err != nil {
			return nil, fmt.Errorf("resolve: parse id %q: %w", id, err)
		}
		dupNums = append(dupNums, n)
		dupMap[n] = id
	}
	sort.Ints(dupNums)

	var plan []store.RenameOp
	for _, n := range dupNums {
		id := dupMap[n]
		tickets, err := readResolveGroup(groups[id])
		if err != nil {
			return nil, err
		}
		sort.SliceStable(tickets, func(i, j int) bool {
			if !tickets[i].created.Equal(tickets[j].created) {
				return tickets[i].created.Before(tickets[j].created)
			}
			return tickets[i].file.Path < tickets[j].file.Path
		})

		for _, item := range tickets[1:] {
			if nextID > 9999 {
				return nil, errors.New("resolve: no available four-digit ticket IDs")
			}
			newID := fmt.Sprintf("%04d", nextID)
			nextID++
			newBase := newID + filepath.Base(item.file.Path)[4:]
			newPath := filepath.Join(filepath.Dir(item.file.Path), newBase)
			if _, err := os.Stat(newPath); err == nil {
				return nil, fmt.Errorf("resolve: destination already exists: %s", displayPath(newPath))
			} else if !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("resolve: check destination %s: %w", displayPath(newPath), err)
			}
			plan = append(plan, store.RenameOp{
				OldPath: item.file.Path,
				NewBase: newBase,
			})
		}
	}
	return plan, nil
}

func readResolveGroup(files []store.ManagedFile) ([]resolveTicket, error) {
	tickets := make([]resolveTicket, 0, len(files))
	for _, file := range files {
		t, err := st.ReadTicket(file.Path)
		if err != nil {
			return nil, fmt.Errorf("resolve: read %s: %w", displayPath(file.Path), err)
		}
		// Zero Created (field absent from frontmatter) sorts before any real
		// timestamp, so the ticket is treated as the oldest and keeps its ID.
		tickets = append(tickets, resolveTicket{
			file:    file,
			created: t.Created,
		})
	}
	return tickets, nil
}
