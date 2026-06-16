package tui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/108adams/clinban/internal/config"
	"github.com/108adams/clinban/internal/store"
	"github.com/108adams/clinban/internal/ticket"
)

// ---- helpers ----

// rec builds a store.Record with the given id and status for ordering tests.
func rec(id string, st ticket.Status) store.Record {
	return store.Record{
		Ticket: &ticket.Ticket{ID: id, Status: st, Type: ticket.TypeTask, Title: "Ticket " + id},
		Path:   id + "-x.md",
	}
}

// update applies msg to m and asserts the returned model is a tui.Model.
func update(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	nm, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", next)
	}
	return nm, cmd
}

// keyPress builds a printable-key KeyPressMsg whose String() is s.
func keyPress(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
}

// newStoreWithFixture returns a temp-dir Store containing one active ticket.
func newStoreWithFixture(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "archive"), 0o750); err != nil {
		t.Fatalf("mkdir archive: %v", err)
	}
	cfg := &config.Config{TicketsDir: dir, ArchiveDir: filepath.Join(dir, "archive")}
	now := time.Now().UTC()
	tk := &ticket.Ticket{
		Status: ticket.StatusBacklog, Type: ticket.TypeTask,
		Title: "Fixture", Tags: []string{}, Created: now, Updated: now,
	}
	b, err := ticket.Marshal(tk)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "0001-fixture.md"), b, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return store.New(cfg)
}

// ---- tests ----

func TestUpdate_TicketsLoaded_SortsBoardOrder(t *testing.T) {
	t.Parallel()
	m := New(nil)
	recs := []store.Record{
		rec("0001", ticket.StatusDone),
		rec("0002", ticket.StatusBacklog),
		rec("0003", ticket.StatusInProgress),
		rec("0004", ticket.StatusBlocked),
	}
	m, _ = update(t, m, ticketsLoadedMsg{Records: recs})

	if m.err != nil {
		t.Fatalf("err = %v, want nil", m.err)
	}
	want := []string{"0003", "0004", "0002", "0001"} // in-progress, blocked, backlog, done
	if len(m.records) != len(want) {
		t.Fatalf("records len = %d, want %d", len(m.records), len(want))
	}
	for i, id := range want {
		if m.records[i].Ticket.ID != id {
			t.Errorf("records[%d].ID = %q, want %q", i, m.records[i].Ticket.ID, id)
		}
	}
	if got := len(m.list.Items()); got != len(want) {
		t.Errorf("list items = %d, want %d", got, len(want))
	}
}

func TestUpdate_TicketsLoaded_ErrorState(t *testing.T) {
	t.Parallel()
	m := New(nil)
	// Populate first, then fail — the board must not keep a half-list.
	m, _ = update(t, m, ticketsLoadedMsg{Records: []store.Record{rec("0001", ticket.StatusBacklog)}})
	m, _ = update(t, m, ticketsLoadedMsg{Err: errors.New("boom")})

	if m.err == nil {
		t.Fatal("err = nil, want set")
	}
	if m.records != nil {
		t.Errorf("records = %v, want nil on error", m.records)
	}
	if got := len(m.list.Items()); got != 0 {
		t.Errorf("list items = %d, want 0 (no half-board)", got)
	}
}

func TestUpdate_Navigation_Clamps(t *testing.T) {
	t.Parallel()
	m := New(nil)
	m, _ = update(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = update(t, m, ticketsLoadedMsg{Records: []store.Record{
		rec("0001", ticket.StatusInProgress),
		rec("0002", ticket.StatusInProgress),
		rec("0003", ticket.StatusInProgress),
	}})

	if m.list.Index() != 0 {
		t.Fatalf("initial index = %d, want 0", m.list.Index())
	}
	m, _ = update(t, m, keyPress("j"))
	if m.list.Index() != 1 {
		t.Fatalf("after j, index = %d, want 1", m.list.Index())
	}
	// Over-scroll down: clamp at last.
	m, _ = update(t, m, keyPress("j"))
	m, _ = update(t, m, keyPress("j"))
	if m.list.Index() != 2 {
		t.Fatalf("after 3x j, index = %d, want 2 (clamped)", m.list.Index())
	}
	// Over-scroll up: clamp at first.
	m, _ = update(t, m, keyPress("k"))
	m, _ = update(t, m, keyPress("k"))
	m, _ = update(t, m, keyPress("k"))
	if m.list.Index() != 0 {
		t.Fatalf("after 3x k, index = %d, want 0 (clamped)", m.list.Index())
	}
}

func TestUpdate_Reload_IssuesLoadTickets(t *testing.T) {
	t.Parallel()
	st := newStoreWithFixture(t)
	m := New(st)
	_, cmd := update(t, m, keyPress("r"))
	if cmd == nil {
		t.Fatal("r produced no cmd, want a reload")
	}
	msg := cmd()
	loaded, ok := msg.(ticketsLoadedMsg)
	if !ok {
		t.Fatalf("reload cmd returned %T, want ticketsLoadedMsg", msg)
	}
	if loaded.Err != nil {
		t.Fatalf("reload err = %v, want nil", loaded.Err)
	}
	if len(loaded.Records) != 1 {
		t.Errorf("reload loaded %d records, want 1", len(loaded.Records))
	}
}

func TestUpdate_HelpToggle(t *testing.T) {
	t.Parallel()
	m := New(nil)
	m, _ = update(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	if m.help.ShowAll {
		t.Fatal("help should start collapsed")
	}
	m, _ = update(t, m, keyPress("?"))
	if !m.help.ShowAll {
		t.Fatal("after ?, help should be expanded")
	}
	m, _ = update(t, m, keyPress("?"))
	if m.help.ShowAll {
		t.Fatal("after second ?, help should be collapsed")
	}
}

func TestUpdate_WindowSize_StoresDims_ViewNoPanic(t *testing.T) {
	t.Parallel()
	m := New(nil)
	m, _ = update(t, m, ticketsLoadedMsg{Records: []store.Record{rec("0001", ticket.StatusInProgress)}})

	for _, sz := range []struct{ w, h int }{{80, 24}, {1, 1}, {0, 0}, {3, 2}} {
		m, _ = update(t, m, tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
		if m.width != sz.w || m.height != sz.h {
			t.Errorf("dims = %dx%d, want %dx%d", m.width, m.height, sz.w, sz.h)
		}
		// Must not panic at any size, including degenerate ones.
		_ = m.View()
	}
}

func TestUpdate_QuitKeys_ReturnQuit(t *testing.T) {
	t.Parallel()
	keys := []tea.KeyPressMsg{
		keyPress("q"),
		{Code: 'c', Mod: tea.ModCtrl}, // ctrl+c
		{Code: tea.KeyEscape},         // esc
	}
	for _, k := range keys {
		m := New(nil)
		_, cmd := update(t, m, k)
		if cmd == nil {
			t.Fatalf("quit key %q produced no cmd", k.String())
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("quit key %q cmd returned %T, want tea.QuitMsg", k.String(), cmd())
		}
	}
}
