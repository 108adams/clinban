package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// tspec describes a fixture ticket: a 4-digit ID and a status.
type tspec struct {
	id     string
	status ticket.Status
}

// fixturePast is the timestamp written into fixtures so a later advanceStatus
// write produces a strictly newer Updated value.
var fixturePast = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

// newStoreWith returns a temp-dir Store seeded with the given fixture tickets,
// plus the tickets directory path.
func newStoreWith(t *testing.T, specs ...tspec) (*store.Store, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "archive"), 0o750); err != nil {
		t.Fatalf("mkdir archive: %v", err)
	}
	cfg := &config.Config{TicketsDir: dir, ArchiveDir: filepath.Join(dir, "archive")}
	for _, s := range specs {
		tk := &ticket.Ticket{
			Status: s.status, Type: ticket.TypeTask, Title: "Ticket " + s.id,
			Tags: []string{}, Created: fixturePast, Updated: fixturePast,
		}
		b, err := ticket.Marshal(tk)
		if err != nil {
			t.Fatalf("marshal %s: %v", s.id, err)
		}
		if err := os.WriteFile(filepath.Join(dir, s.id+"-t.md"), b, 0o600); err != nil {
			t.Fatalf("write %s: %v", s.id, err)
		}
	}
	return store.New(cfg), dir
}

// readStatus reads the on-disk status and Updated for id from st.
func readStatus(t *testing.T, st *store.Store, id string) (ticket.Status, time.Time) {
	t.Helper()
	path, _, err := st.FindByID(id)
	if err != nil {
		t.Fatalf("FindByID %s: %v", id, err)
	}
	tk, err := st.ReadTicket(path)
	if err != nil {
		t.Fatalf("ReadTicket %s: %v", id, err)
	}
	return tk.Status, tk.Updated
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

func TestUpdate_PreviewLoaded_SetsExactContent(t *testing.T) {
	t.Parallel()
	m := New(nil)
	raw := []byte("---\nstatus: backlog\n---\n# raw body\n")
	m, _ = update(t, m, previewLoadedMsg{Content: raw})
	if got := m.preview.GetContent(); got != string(raw) {
		t.Errorf("preview content = %q, want %q", got, raw)
	}
}

func TestUpdate_SelectionChange_IssuesPreviewForNewPath(t *testing.T) {
	t.Parallel()
	m := New(nil)
	m, _ = update(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = update(t, m, ticketsLoadedMsg{Records: []store.Record{
		rec("0001", ticket.StatusInProgress),
		rec("0002", ticket.StatusInProgress),
	}})

	_, cmd := update(t, m, keyPress("j")) // selection 0001 -> 0002
	if cmd == nil {
		t.Fatal("selection change produced no cmd")
	}
	msg := cmd()
	pl, ok := msg.(previewLoadedMsg)
	if !ok {
		t.Fatalf("selection-change cmd returned %T, want previewLoadedMsg", msg)
	}
	if pl.Path != "0002-x.md" {
		t.Errorf("preview path = %q, want %q", pl.Path, "0002-x.md")
	}
}

// TestLoadPreview_ReturnsRawFileBytes guards ADR-4: the preview is the original
// file bytes, never a re-marshaled Ticket.
func TestLoadPreview_ReturnsRawFileBytes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Non-canonical frontmatter (status first, not title) + a blank line, so the
	// raw bytes differ from ticket.Marshal output.
	raw := []byte("---\nstatus: backlog\ntype: task\ntitle: \"Weird order\"\ntags: []\n" +
		"created: 2026-01-01T00:00:00Z\nupdated: 2026-01-01T00:00:00Z\n---\n\n# Body\n\nverbatim\n")
	path := filepath.Join(dir, "0009-weird.md")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	msg, ok := loadPreview(path)().(previewLoadedMsg)
	if !ok {
		t.Fatalf("loadPreview returned %T, want previewLoadedMsg", msg)
	}
	if msg.Err != nil {
		t.Fatalf("loadPreview err = %v", msg.Err)
	}
	if string(msg.Content) != string(raw) {
		t.Errorf("preview not verbatim:\n got: %q\nwant: %q", msg.Content, raw)
	}
	// Sanity: confirm the fixture really is non-canonical, so the verbatim
	// assertion above genuinely exercises the no-remarshal guarantee.
	tk, err := ticket.Parse(raw)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	if marshaled, err := ticket.Marshal(tk); err == nil && string(marshaled) == string(raw) {
		t.Fatal("fixture is canonical; cannot prove non-remarshaling — adjust frontmatter")
	}
}

func TestUpdate_ScrollKeys_MovePreview(t *testing.T) {
	t.Parallel()
	m := New(nil)
	m, _ = update(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})

	var sb strings.Builder
	for i := range 200 {
		fmt.Fprintf(&sb, "line %d\n", i)
	}
	m, _ = update(t, m, previewLoadedMsg{Content: []byte(sb.String())})
	if !m.preview.AtTop() {
		t.Fatal("preview should start at top")
	}

	m, _ = update(t, m, tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl}) // ctrl+d
	down := m.preview.YOffset()
	if down == 0 {
		t.Fatal("ctrl+d did not scroll the preview down")
	}
	m, _ = update(t, m, tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}) // ctrl+u
	if m.preview.YOffset() >= down {
		t.Errorf("ctrl+u did not scroll up: YOffset %d >= %d", m.preview.YOffset(), down)
	}
}

func TestAdvanceStatus_BacklogToInProgress(t *testing.T) {
	t.Parallel()
	st, _ := newStoreWith(t, tspec{"0001", ticket.StatusBacklog})

	msg, ok := advanceStatus(st, "0001")().(statusAdvancedMsg)
	if !ok {
		t.Fatalf("advanceStatus returned %T, want statusAdvancedMsg", msg)
	}
	if msg.Err != nil || msg.NoForward {
		t.Fatalf("advanceStatus = %+v, want success", msg)
	}
	gotStatus, gotUpdated := readStatus(t, st, "0001")
	if gotStatus != ticket.StatusInProgress {
		t.Errorf("on-disk status = %q, want in-progress", gotStatus)
	}
	if !gotUpdated.After(fixturePast) {
		t.Errorf("Updated = %v, want refreshed (after %v)", gotUpdated, fixturePast)
	}
}

func TestAdvanceStatus_DoneIsNoForward(t *testing.T) {
	t.Parallel()
	st, _ := newStoreWith(t, tspec{"0001", ticket.StatusDone})

	msg, ok := advanceStatus(st, "0001")().(statusAdvancedMsg)
	if !ok {
		t.Fatalf("advanceStatus returned %T, want statusAdvancedMsg", msg)
	}
	if !msg.NoForward {
		t.Fatalf("advanceStatus(done) = %+v, want NoForward", msg)
	}
	gotStatus, gotUpdated := readStatus(t, st, "0001")
	if gotStatus != ticket.StatusDone {
		t.Errorf("status changed to %q, want unchanged done", gotStatus)
	}
	if !gotUpdated.Equal(fixturePast) {
		t.Errorf("Updated changed to %v, want unchanged %v (no write)", gotUpdated, fixturePast)
	}
}

func TestAdvanceStatus_WriteError(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("write-permission errors do not apply to root")
	}
	st, dir := newStoreWith(t, tspec{"0001", ticket.StatusBacklog})

	// Make the tickets dir read-only so the atomic write (temp create) fails.
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	msg := advanceStatus(st, "0001")().(statusAdvancedMsg)
	if msg.Err == nil {
		t.Fatalf("advanceStatus = %+v, want a write error", msg)
	}
	// Restore perms and confirm the live file was not mutated.
	_ = os.Chmod(dir, 0o700)
	gotStatus, gotUpdated := readStatus(t, st, "0001")
	if gotStatus != ticket.StatusBacklog || !gotUpdated.Equal(fixturePast) {
		t.Errorf("file mutated despite write error: status=%q updated=%v", gotStatus, gotUpdated)
	}
}

func TestUpdate_AdvanceKey_IssuesAdvanceThenReload(t *testing.T) {
	t.Parallel()
	st, _ := newStoreWith(t, tspec{"0001", ticket.StatusBacklog})
	m := New(st)
	m, _ = update(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = update(t, m, loadTickets(st)().(ticketsLoadedMsg))

	// ">" issues an advanceStatus cmd for the selected ticket.
	_, cmd := update(t, m, keyPress(">"))
	if cmd == nil {
		t.Fatal("> produced no cmd")
	}
	adv, ok := cmd().(statusAdvancedMsg)
	if !ok {
		t.Fatalf("> cmd returned %T, want statusAdvancedMsg", adv)
	}
	if adv.Err != nil || adv.NoForward {
		t.Fatalf("advance = %+v, want success", adv)
	}
	// A successful advance triggers a reload.
	_, reloadCmd := update(t, m, statusAdvancedMsg{})
	if reloadCmd == nil {
		t.Fatal("successful advance did not issue a reload cmd")
	}
	if _, ok := reloadCmd().(ticketsLoadedMsg); !ok {
		t.Errorf("reload cmd returned %T, want ticketsLoadedMsg", reloadCmd())
	}
}

func TestUpdate_SelectionStableAfterAdvanceResort(t *testing.T) {
	t.Parallel()
	// Board order: in-progress(0001), backlog(0002), backlog(0003).
	st, _ := newStoreWith(t,
		tspec{"0001", ticket.StatusInProgress},
		tspec{"0002", ticket.StatusBacklog},
		tspec{"0003", ticket.StatusBacklog},
	)
	m := New(st)
	m, _ = update(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = update(t, m, loadTickets(st)().(ticketsLoadedMsg))

	// Move to 0003 (index 2).
	m, _ = update(t, m, keyPress("j"))
	m, _ = update(t, m, keyPress("j"))
	if id, _ := m.selectedID(); id != "0003" {
		t.Fatalf("pre-advance selection = %q, want 0003", id)
	}

	// Advance 0003 backlog -> in-progress, then run the reload it triggers.
	adv := advanceStatus(st, "0003")().(statusAdvancedMsg)
	if adv.Err != nil || adv.NoForward {
		t.Fatalf("advance = %+v, want success", adv)
	}
	m, reloadCmd := update(t, m, statusAdvancedMsg{})
	mm := reloadCmd().(ticketsLoadedMsg)
	m, _ = update(t, m, mm)

	// After re-sort: in-progress(0001, 0003), backlog(0002) -> 0003 at index 1.
	if id, _ := m.selectedID(); id != "0003" {
		t.Errorf("post-advance selection = %q, want 0003 (cursor follows the ticket)", id)
	}
	if m.list.Index() != 1 {
		t.Errorf("post-advance index = %d, want 1 (re-sorted position)", m.list.Index())
	}
}

func TestUpdate_AdvanceKey_EmptyBoardNoop(t *testing.T) {
	t.Parallel()
	m := New(nil)
	m, _ = update(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	// No tickets loaded — > must be a safe no-op.
	_, cmd := update(t, m, keyPress(">"))
	if cmd != nil {
		t.Errorf("> on empty board issued a cmd, want nil")
	}
}

func TestApplyPendingSelection_MissingIDClamps(t *testing.T) {
	t.Parallel()
	m := New(nil)
	m, _ = update(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = update(t, m, ticketsLoadedMsg{Records: []store.Record{
		rec("0001", ticket.StatusInProgress),
		rec("0002", ticket.StatusInProgress),
	}})
	m.pendingSelectID = "9999" // gone
	m = m.applyPendingSelection()
	if idx := m.list.Index(); idx < 0 || idx >= len(m.records) {
		t.Errorf("clamped index = %d, want a valid index in [0,%d)", idx, len(m.records))
	}
	if m.pendingSelectID != "" {
		t.Errorf("pendingSelectID = %q, want cleared", m.pendingSelectID)
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
