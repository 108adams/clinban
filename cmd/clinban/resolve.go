package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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

type resolveRename struct {
	oldPath string
	newBase string
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

	for _, item := range plan {
		newPath, err := st.RenameWithinDir(item.oldPath, item.newBase)
		if err != nil {
			return fmt.Errorf("resolve: rename %s: %w", filepath.Base(item.oldPath), err)
		}
		fmt.Fprintf(os.Stdout, "renamed: %s -> %s\n", displayPath(item.oldPath), displayPath(newPath))
	}
	return nil
}

func planResolve(files []store.ManagedFile) ([]resolveRename, error) {
	groups := map[string][]store.ManagedFile{}
	maxID := 0
	for _, file := range files {
		groups[file.ID] = append(groups[file.ID], file)
		n, err := strconv.Atoi(file.ID)
		if err != nil {
			return nil, fmt.Errorf("resolve: parse id %q: %w", file.ID, err)
		}
		if n > maxID {
			maxID = n
		}
	}

	ids := make([]string, 0, len(groups))
	for id, group := range groups {
		if len(group) > 1 {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		ni, _ := strconv.Atoi(ids[i])
		nj, _ := strconv.Atoi(ids[j])
		return ni < nj
	})

	var plan []resolveRename
	nextID := maxID + 1
	for _, id := range ids {
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
			plan = append(plan, resolveRename{
				oldPath: item.file.Path,
				newBase: newBase,
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
