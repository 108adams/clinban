package tui

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/108adams/clinban/internal/board"
	"github.com/108adams/clinban/internal/store"
)

// ticketItem wraps a store.Record so it satisfies the bubbles list.Item
// interface. It also satisfies list.DefaultItem so the DefaultDelegate renders
// a title + description row.
type ticketItem struct {
	record store.Record
}

func (i ticketItem) FilterValue() string { return i.record.Ticket.Title }
func (i ticketItem) Title() string {
	return fmt.Sprintf("[%s] %s", i.record.Ticket.ID, i.record.Ticket.Title)
}
func (i ticketItem) Description() string {
	return fmt.Sprintf("%s • %s", i.record.Ticket.Status, i.record.Ticket.Type)
}

// Model is the Bubble Tea model for the board TUI.
//
// It is a pure consumer: it never calls os.Link/os.Remove/os.Rename on
// managed tickets; all store access is funnelled through commands.go.
type Model struct {
	st      *store.Store
	records []store.Record // current active snapshot

	list    list.Model
	preview viewport.Model
	keys    keyMap
	help    help.Model
	width   int
	height  int
	err     error  // non-nil when the last load failed
	status  string // transient status/lint message line

	// pendingSelectID, when non-empty, is the ticket ID to re-select on the next
	// ticketsLoadedMsg so a reload keeps the cursor on the acted-on ticket.
	pendingSelectID string

	// scratch and editLive are set during an in-flight edit (T7).
	scratch  string
	editLive string
}

// New constructs the board Model. It performs no I/O; I/O starts in Init.
func New(st *store.Store) Model {
	delegate := list.NewDefaultDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.SetShowHelp(false) // we render our own help bar
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Title = "Board"

	vp := viewport.New()

	return Model{
		st:      st,
		list:    l,
		preview: vp,
		keys:    defaultKeyMap(),
		help:    help.New(),
	}
}

// Init returns the initial command: load tickets from the store.
func (m Model) Init() tea.Cmd {
	return loadTickets(m.st)
}

// Update handles incoming messages and updates the model.
// It contains no blocking I/O — all store access runs inside Cmds in commands.go.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.recalcLayout()
		return m, nil

	case ticketsLoadedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.records = nil
			m.list.SetItems(nil)
			return m, nil
		}
		m.err = nil
		recs := msg.Records
		sort.SliceStable(recs, func(i, j int) bool {
			return board.Less(recs[i].Ticket, recs[j].Ticket)
		})
		m.records = recs
		items := make([]list.Item, len(recs))
		for i, r := range recs {
			items[i] = ticketItem{record: r}
		}
		cmd := m.list.SetItems(items)
		m = m.applyPendingSelection()
		if path, ok := m.selectedPath(); ok {
			return m, tea.Batch(cmd, loadPreview(path))
		}
		return m, cmd

	case previewLoadedMsg:
		if msg.Err != nil {
			m.status = "preview: " + msg.Err.Error()
			return m, nil
		}
		m.preview.SetContent(string(msg.Content))
		return m, nil

	case statusAdvancedMsg:
		switch {
		case msg.Err != nil:
			m.status = "advance: " + msg.Err.Error()
			return m, nil
		case msg.NoForward:
			m.status = "no further status"
			return m, nil
		default:
			m.status = ""
			return m.withReload()
		}

	case tea.KeyPressMsg:
		switch {
		case isQuit(msg, m.keys):
			return m, func() tea.Msg { return tea.Quit() }

		case isHelp(msg, m.keys):
			m.help.ShowAll = !m.help.ShowAll
			m = m.recalcLayout()
			return m, nil

		case isReload(msg, m.keys):
			return m.withReload()

		case isAdvance(msg, m.keys):
			if id, ok := m.selectedID(); ok {
				return m, advanceStatus(m.st, id)
			}
			return m, nil

		case isScrollDown(msg, m.keys):
			m.preview.HalfPageDown()
			return m, nil

		case isScrollUp(msg, m.keys):
			m.preview.HalfPageUp()
			return m, nil

		case isUp(msg, m.keys):
			before := m.list.Index()
			m.list.CursorUp()
			return m.previewOnChange(before)

		case isDown(msg, m.keys):
			before := m.list.Index()
			m.list.CursorDown()
			return m.previewOnChange(before)
		}
	}

	// Delegate remaining messages to the list (handles pagination etc.).
	var listCmd tea.Cmd
	m.list, listCmd = m.list.Update(msg)
	return m, listCmd
}

// View renders the two-pane board layout. It is a pure read: all sub-model
// sizing happens in recalcLayout, never during render.
func (m Model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		return tea.NewView("")
	}
	v := tea.NewView(m.renderContent())
	v.AltScreen = true
	return v
}

// statusLine returns the transient status/error line ("" when there is none).
func (m Model) statusLine() string {
	if m.err != nil {
		return "error: " + m.err.Error()
	}
	return m.status
}

// helpHeight is the number of rows the help view currently occupies.
func (m Model) helpHeight() int {
	if m.help.ShowAll {
		return strings.Count(m.help.View(m.keys), "\n") + 1
	}
	return 1
}

// contentHeight is the height available to the two panes after the status and
// help lines are reserved (always at least 1).
func (m Model) contentHeight() int {
	h := m.height - m.helpHeight()
	if m.statusLine() != "" {
		h--
	}
	return max(h, 1)
}

// paneWidths returns the left and right pane widths; right excludes the
// single-column divider.
func (m Model) paneWidths() (left, right int) {
	left = max(m.width*40/100, 10)
	right = max(m.width-left-1, 1)
	return left, right
}

// recalcLayout sizes the list and preview sub-models to the current window,
// help, and status dimensions. Called whenever those inputs change so View can
// stay a pure read.
func (m Model) recalcLayout() Model {
	if m.width == 0 || m.height == 0 {
		return m
	}
	ch := m.contentHeight()
	lw, rw := m.paneWidths()
	m.list.SetWidth(lw)
	m.list.SetHeight(ch)
	m.preview.SetWidth(rw)
	m.preview.SetHeight(ch)
	return m
}

// renderContent joins the already-sized sub-models into the board layout.
// It performs no resizing — that is recalcLayout's job.
func (m Model) renderContent() string {
	ch := m.contentHeight()
	leftLines := splitLines(padLines(m.list.View(), ch))
	rightLines := splitLines(padLines(m.preview.View(), ch))

	rows := make([]string, 0, ch)
	for i := range ch {
		var l, r string
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		rows = append(rows, l+"│"+r)
	}

	content := joinLines(rows)
	if s := m.statusLine(); s != "" {
		content += "\n" + s
	}
	content += "\n" + m.help.View(m.keys)
	return content
}

// Key-match helpers (keep Update readable). key.Matches handles the binding's
// Enabled() check and key-set comparison via the message's String().

func isQuit(msg tea.KeyPressMsg, km keyMap) bool       { return key.Matches(msg, km.Quit) }
func isHelp(msg tea.KeyPressMsg, km keyMap) bool       { return key.Matches(msg, km.Help) }
func isReload(msg tea.KeyPressMsg, km keyMap) bool     { return key.Matches(msg, km.Reload) }
func isUp(msg tea.KeyPressMsg, km keyMap) bool         { return key.Matches(msg, km.Up) }
func isDown(msg tea.KeyPressMsg, km keyMap) bool       { return key.Matches(msg, km.Down) }
func isScrollDown(msg tea.KeyPressMsg, km keyMap) bool { return key.Matches(msg, km.ScrollDown) }
func isScrollUp(msg tea.KeyPressMsg, km keyMap) bool   { return key.Matches(msg, km.ScrollUp) }
func isAdvance(msg tea.KeyPressMsg, km keyMap) bool    { return key.Matches(msg, km.Advance) }

// selectedPath returns the file path of the currently selected record.
func (m Model) selectedPath() (string, bool) {
	i := m.list.Index()
	if i < 0 || i >= len(m.records) {
		return "", false
	}
	return m.records[i].Path, true
}

// selectedID returns the ticket ID of the currently selected record.
func (m Model) selectedID() (string, bool) {
	i := m.list.Index()
	if i < 0 || i >= len(m.records) {
		return "", false
	}
	return m.records[i].Ticket.ID, true
}

// withReload remembers the currently selected ticket ID and issues a board
// reload. applyPendingSelection re-selects that ID once the reload lands, so
// the cursor stays on the acted-on ticket even after it re-sorts into another
// group. Every reload path goes through here.
func (m Model) withReload() (tea.Model, tea.Cmd) {
	if id, ok := m.selectedID(); ok {
		m.pendingSelectID = id
	}
	return m, loadTickets(m.st)
}

// applyPendingSelection re-selects pendingSelectID in the freshly loaded list.
// If that ID is gone, it clamps to the nearest valid index. No-op when no
// selection is pending.
func (m Model) applyPendingSelection() Model {
	if m.pendingSelectID == "" {
		return m
	}
	target := m.pendingSelectID
	m.pendingSelectID = ""
	for i, r := range m.records {
		if r.Ticket.ID == target {
			m.list.Select(i)
			return m
		}
	}
	if n := len(m.records); n > 0 {
		m.list.Select(min(m.list.Index(), n-1))
	}
	return m
}

// previewOnChange issues a preview load only when the selection actually moved
// from idxBefore, avoiding a redundant re-read when navigation clamps.
func (m Model) previewOnChange(idxBefore int) (tea.Model, tea.Cmd) {
	if m.list.Index() == idxBefore {
		return m, nil
	}
	if path, ok := m.selectedPath(); ok {
		return m, loadPreview(path)
	}
	return m, nil
}

// padLines ensures the string has exactly n lines by appending blank lines.
func padLines(s string, n int) string {
	lines := splitLines(s)
	for len(lines) < n {
		lines = append(lines, "")
	}
	return joinLines(lines[:n])
}

func splitLines(s string) []string {
	if s == "" {
		return []string{""}
	}
	var out []string
	start := 0
	for i, ch := range s {
		if ch == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	total := len(lines) - 1
	for _, l := range lines {
		total += len(l)
	}
	buf := make([]byte, 0, total)
	for i, l := range lines {
		if i > 0 {
			buf = append(buf, '\n')
		}
		buf = append(buf, l...)
	}
	return string(buf)
}
