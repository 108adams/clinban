package tui

import (
	"os"

	tea "charm.land/bubbletea/v2"

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
