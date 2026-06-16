package tui

import "github.com/108adams/clinban/internal/store"

// ticketsLoadedMsg is sent by loadTickets after ListActive completes.
// On success, Records holds the active records and Err is nil.
// On failure, Err is set and Records is nil — the board must not render a
// partial list.
type ticketsLoadedMsg struct {
	Records []store.Record
	Err     error
}

// previewLoadedMsg is sent by loadPreview after reading a ticket's file.
// Content is the exact on-disk bytes — never a re-marshaled Ticket (ADR-4).
type previewLoadedMsg struct {
	Path    string
	Content []byte
	Err     error
}

// statusAdvancedMsg is sent by advanceStatus. On success both fields are zero.
// NoForward is true when the ticket is already at a terminal status (no write
// happened); Err carries any find/read/write failure (snapshot left unchanged).
type statusAdvancedMsg struct {
	Err       error
	NoForward bool
}
