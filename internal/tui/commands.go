package tui

import (
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/108adams/clinban/internal/fsm"
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
