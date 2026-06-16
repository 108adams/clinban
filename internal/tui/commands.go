package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/108adams/clinban/internal/store"
)

// loadTickets returns a tea.Cmd that calls st.ListActive() and delivers the
// result as a ticketsLoadedMsg. This is the only place the store is read in T4.
func loadTickets(st *store.Store) tea.Cmd {
	return func() tea.Msg {
		recs, err := st.ListActive()
		return ticketsLoadedMsg{Records: recs, Err: err}
	}
}
