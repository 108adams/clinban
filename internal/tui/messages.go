package tui

import (
	"os/exec"

	"github.com/108adams/clinban/internal/lint"
	"github.com/108adams/clinban/internal/store"
)

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

// editReadyMsg is sent by beginEdit when the scratch copy and editor command
// are ready. The Update loop runs Cmd via tea.ExecProcess (the only place the
// terminal is released for the editor).
type editReadyMsg struct {
	Scratch  string
	LivePath string
	Cmd      *exec.Cmd
}

// editBeginFailedMsg is sent when beginEdit could not prepare the edit
// (find/read/scratch/editor-resolution failure). Any partial scratch has
// already been removed.
type editBeginFailedMsg struct {
	Err error
}

// editFinishedMsg is sent by tea.ExecProcess when the editor child exits.
type editFinishedMsg struct {
	Err error
}

// editCommittedMsg is the terminal result of commitEdit. On success all fields
// are zero. ParseOrIOErr groups scratch-read, AllIDs-scan, ticket.Parse, and
// write failures (reported distinctly from lint); LintErrs holds lint
// violations. In every non-success case the live file is left unchanged.
type editCommittedMsg struct {
	LintErrs     []lint.LintError
	ParseOrIOErr error
}
