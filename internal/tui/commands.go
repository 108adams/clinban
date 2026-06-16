package tui

import (
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/108adams/clinban/internal/editor"
	"github.com/108adams/clinban/internal/fsm"
	"github.com/108adams/clinban/internal/lint"
	"github.com/108adams/clinban/internal/store"
)

// loadTickets returns a tea.Cmd that calls st.ListActive() and delivers the
// result as a ticketsLoadedMsg. This is the only place active tickets are read.
func loadTickets(st *store.Store) tea.Cmd {
	return func() tea.Msg {
		recs, err := st.ListActive()
		return ticketsLoadedMsg{Records: recs, Err: err}
	}
}

// loadPreview returns a tea.Cmd that reads the raw file bytes at path and
// delivers them as a previewLoadedMsg. Per ADR-4 the preview is the original
// file content (os.ReadFile on Record.Path), never a re-marshaled Ticket.
func loadPreview(path string) tea.Cmd {
	return func() tea.Msg {
		b, err := os.ReadFile(path)
		return previewLoadedMsg{Path: path, Content: b, Err: err}
	}
}

// advanceStatus returns a tea.Cmd that advances the ticket with the given id to
// its next workflow status. It re-resolves and re-reads the ticket fresh from
// disk (never the in-memory snapshot), computes the next status via fsm,
// refreshes Updated, and writes via store.WriteTicket — mirroring the CLI move
// command. A terminal ticket reports NoForward and writes nothing; any
// find/read/write error is surfaced with the live file left unchanged.
func advanceStatus(st *store.Store, id string) tea.Cmd {
	return func() tea.Msg {
		path, _, err := st.FindByID(id)
		if err != nil {
			return statusAdvancedMsg{Err: err}
		}
		tk, err := st.ReadTicket(path)
		if err != nil {
			return statusAdvancedMsg{Err: err}
		}
		next, ok := fsm.NextStatus(tk.Status)
		if !ok {
			return statusAdvancedMsg{NoForward: true}
		}
		tk.Status = next
		tk.Updated = time.Now()
		if err := st.WriteTicket(tk, path); err != nil {
			return statusAdvancedMsg{Err: err}
		}
		return statusAdvancedMsg{}
	}
}

// beginEdit returns a tea.Cmd that prepares an editor handoff WITHOUT any
// blocking I/O on the Update path. It fresh-resolves the live path (FindByID,
// not the snapshot), copies the live bytes into a same-directory scratch file
// (mirroring cmd/clinban/edit.go), and builds the editor command (stdio unset —
// the caller wires the tty via tea.ExecProcess). Any partial scratch is removed
// on failure. The scratch name is dot-prefixed so ListActive ignores it.
func beginEdit(st *store.Store, id string) tea.Cmd {
	return func() tea.Msg {
		livePath, _, err := st.FindByID(id)
		if err != nil {
			return editBeginFailedMsg{Err: err}
		}
		raw, err := os.ReadFile(livePath)
		if err != nil {
			return editBeginFailedMsg{Err: err}
		}
		f, err := os.CreateTemp(filepath.Dir(livePath), ".clinban-edit-*.md")
		if err != nil {
			return editBeginFailedMsg{Err: err}
		}
		scratch := f.Name()
		if _, err := f.Write(raw); err != nil {
			f.Close() //nolint:errcheck // already failing; cleanup below
			os.Remove(scratch)
			return editBeginFailedMsg{Err: err}
		}
		if err := f.Close(); err != nil {
			os.Remove(scratch)
			return editBeginFailedMsg{Err: err}
		}
		cmd, err := editor.Command(scratch)
		if err != nil {
			os.Remove(scratch)
			return editBeginFailedMsg{Err: err}
		}
		return editReadyMsg{Scratch: scratch, LivePath: livePath, Cmd: cmd}
	}
}

// commitEdit returns a tea.Cmd that validates the edited scratch file and, only
// if it is clean, commits it to the live path. It mirrors the cmd/clinban/edit
// commit kernel: read scratch -> AllIDs -> ValidateForCommit -> WriteTicket.
// Scratch-read, AllIDs-scan, parse, and write failures are reported as
// ParseOrIOErr (distinct from lint); on any failure the live file is untouched.
// The scratch itself is removed by the Update handler, not here.
func commitEdit(st *store.Store, scratch, livePath, filename string) tea.Cmd {
	return func() tea.Msg {
		raw, err := os.ReadFile(scratch)
		if err != nil {
			return editCommittedMsg{ParseOrIOErr: err}
		}
		allIDs, err := st.AllIDs()
		if err != nil {
			return editCommittedMsg{ParseOrIOErr: err}
		}
		id := filename
		if len(id) >= 4 {
			id = id[:4]
		}
		tk, lintErrs, parseErr := lint.ValidateForCommit(raw, id, filename, allIDs)
		if parseErr != nil {
			return editCommittedMsg{ParseOrIOErr: parseErr}
		}
		if len(lintErrs) > 0 {
			return editCommittedMsg{LintErrs: lintErrs}
		}
		tk.Updated = time.Now()
		if err := st.WriteTicket(tk, livePath); err != nil {
			return editCommittedMsg{ParseOrIOErr: err}
		}
		return editCommittedMsg{}
	}
}
