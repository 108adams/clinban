package store_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/108adams/clinban/internal/config"
	"github.com/108adams/clinban/internal/store"
	"github.com/108adams/clinban/internal/ticket"
)

// ---- constants ----

const (
	testTitle  = "Fix login timeout on staging"
	testTitle2 = "Add feature flag support"
	testTitle3 = "Investigate memory leak"
)

// ---- helpers ----

// newStore creates a Store backed by a fresh temp directory. The TicketsDir is
// the temp dir itself; ArchiveDir is a subdirectory named "archive".
func newStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{
		TicketsDir: dir,
		ArchiveDir: filepath.Join(dir, "archive"),
	}
	return store.New(cfg)
}

// writeTicketFile writes a minimal valid ticket markdown file into dir with
// the given filename and ticket fields.
func writeTicketFile(t *testing.T, dir, filename string, tk *ticket.Ticket) string {
	t.Helper()
	b, err := ticket.Marshal(tk)
	if err != nil {
		t.Fatalf("writeTicketFile: marshal: %v", err)
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatalf("writeTicketFile: write %s: %v", path, err)
	}
	return path
}

// makeTicket returns a pointer to a minimal valid Ticket for testing.
func makeTicket(id, title string) *ticket.Ticket {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	return &ticket.Ticket{
		ID:      id,
		Status:  ticket.StatusBacklog,
		Type:    ticket.TypeTask,
		Title:   title,
		Tags:    []string{},
		Created: now,
		Updated: now,
	}
}

// ---- TestNew ----

func TestNew(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	if s.TicketsDir == "" {
		t.Error("TicketsDir must not be empty")
	}
	if s.ArchiveDir == "" {
		t.Error("ArchiveDir must not be empty")
	}
}

// ---- TestNextID ----

func TestNextIDEmpty(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	id, err := s.NextID()
	if err != nil {
		t.Fatalf("NextID returned unexpected error: %v", err)
	}
	if id != 1 {
		t.Errorf("NextID = %d, want 1 for empty directory", id)
	}
}

func TestNextIDWithTickets(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	// Seed the TicketsDir with several ticket files; highest is 0003.
	files := []string{
		"0001-first-ticket.md",
		"0002-second-ticket.md",
		"0003-third-ticket.md",
	}
	for _, name := range files {
		path := filepath.Join(s.TicketsDir, name)
		if err := os.WriteFile(path, []byte("# placeholder"), 0o600); err != nil {
			t.Fatalf("setup: write %s: %v", name, err)
		}
	}

	id, err := s.NextID()
	if err != nil {
		t.Fatalf("NextID returned unexpected error: %v", err)
	}
	if id != 4 {
		t.Errorf("NextID = %d, want 4 (max prefix 0003 + 1)", id)
	}
}

func TestNextIDWithArchive(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	// Active ticket lower than archive ticket.
	activeFile := filepath.Join(s.TicketsDir, "0002-active.md")
	if err := os.WriteFile(activeFile, []byte("# placeholder"), 0o600); err != nil {
		t.Fatalf("setup active: %v", err)
	}

	if err := os.MkdirAll(s.ArchiveDir, 0o755); err != nil {
		t.Fatalf("setup archive dir: %v", err)
	}
	archiveFile := filepath.Join(s.ArchiveDir, "0007-old.md")
	if err := os.WriteFile(archiveFile, []byte("# placeholder"), 0o600); err != nil {
		t.Fatalf("setup archive: %v", err)
	}

	id, err := s.NextID()
	if err != nil {
		t.Fatalf("NextID returned unexpected error: %v", err)
	}
	if id != 8 {
		t.Errorf("NextID = %d, want 8 (max prefix 0007 in archive)", id)
	}
}

func TestNextIDIgnoresNonMatchingFiles(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	// Files that do NOT match [0-9]{4}-*.md should be ignored.
	names := []string{"README.md", "notes.txt", "abc-ticket.md", ".clinban-draft.md"}
	for _, name := range names {
		path := filepath.Join(s.TicketsDir, name)
		if err := os.WriteFile(path, []byte("ignore me"), 0o600); err != nil {
			t.Fatalf("setup: write %s: %v", name, err)
		}
	}

	id, err := s.NextID()
	if err != nil {
		t.Fatalf("NextID returned unexpected error: %v", err)
	}
	if id != 1 {
		t.Errorf("NextID = %d, want 1 (no matching files)", id)
	}
}

// ---- TestFindByID ----

func TestFindByID(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	tk := makeTicket("0005", testTitle)
	writeTicketFile(t, s.TicketsDir, "0005-fix-login-timeout-on.md", tk)

	tests := []struct {
		name      string
		id        string
		wantPath  string
		wantInArc bool
		wantErr   error
	}{
		{
			name:      "found in TicketsDir",
			id:        "0005",
			wantPath:  filepath.Join(s.TicketsDir, "0005-fix-login-timeout-on.md"),
			wantInArc: false,
		},
		{
			name:      "short id 005 normalised to 0005",
			id:        "005",
			wantPath:  filepath.Join(s.TicketsDir, "0005-fix-login-timeout-on.md"),
			wantInArc: false,
		},
		{
			name:      "short id 5 normalised to 0005",
			id:        "5",
			wantPath:  filepath.Join(s.TicketsDir, "0005-fix-login-timeout-on.md"),
			wantInArc: false,
		},
		{
			name:    "missing returns ErrNotFound",
			id:      "9999",
			wantErr: store.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path, inArchive, err := s.FindByID(tc.id)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if path != tc.wantPath {
				t.Errorf("path = %q, want %q", path, tc.wantPath)
			}
			if inArchive != tc.wantInArc {
				t.Errorf("inArchive = %v, want %v", inArchive, tc.wantInArc)
			}
		})
	}
}

func TestFindByIDInArchive(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	if err := os.MkdirAll(s.ArchiveDir, 0o755); err != nil {
		t.Fatalf("setup: mkdir archive: %v", err)
	}

	tk := makeTicket("0003", testTitle3)
	writeTicketFile(t, s.ArchiveDir, "0003-investigate-memory-leak.md", tk)

	path, inArchive, err := s.FindByID("0003")
	if err != nil {
		t.Fatalf("FindByID error: %v", err)
	}
	want := filepath.Join(s.ArchiveDir, "0003-investigate-memory-leak.md")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	if !inArchive {
		t.Error("inArchive = false, want true")
	}
}

func TestFindByIDPrefersTicketsDir(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	if err := os.MkdirAll(s.ArchiveDir, 0o755); err != nil {
		t.Fatalf("setup: mkdir archive: %v", err)
	}

	// Same ID in both dirs — TicketsDir must win.
	tk := makeTicket("0001", testTitle)
	writeTicketFile(t, s.TicketsDir, "0001-fix-login-timeout-on.md", tk)
	writeTicketFile(t, s.ArchiveDir, "0001-old-title.md", tk)

	_, inArchive, err := s.FindByID("0001")
	if err != nil {
		t.Fatalf("FindByID error: %v", err)
	}
	if inArchive {
		t.Error("inArchive = true; expected TicketsDir result when ID exists in both")
	}
}

// ---- TestFindAllByID ----

func TestFindAllByID(t *testing.T) {
	t.Parallel()

	t.Run("no match returns empty slice", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)

		paths, err := s.FindAllByID("0042")
		if err != nil {
			t.Fatalf("FindAllByID error: %v", err)
		}
		if paths == nil {
			t.Fatal("FindAllByID returned nil, want non-nil empty slice")
		}
		if len(paths) != 0 {
			t.Errorf("FindAllByID = %v, want empty slice", paths)
		}
	})

	t.Run("single active match returns slice of 1", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)

		tk := makeTicket("0005", testTitle)
		want := writeTicketFile(t, s.TicketsDir, "0005-fix-login-timeout-on.md", tk)

		paths, err := s.FindAllByID("0005")
		if err != nil {
			t.Fatalf("FindAllByID error: %v", err)
		}
		if len(paths) != 1 {
			t.Fatalf("FindAllByID returned %d paths, want 1: %v", len(paths), paths)
		}
		if paths[0] != want {
			t.Errorf("path = %q, want %q", paths[0], want)
		}
	})

	t.Run("single archive match returns slice of 1", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)

		if err := os.MkdirAll(s.ArchiveDir, 0o755); err != nil {
			t.Fatalf("setup: mkdir archive: %v", err)
		}
		tk := makeTicket("0003", testTitle3)
		want := writeTicketFile(t, s.ArchiveDir, "0003-investigate-memory-leak.md", tk)

		paths, err := s.FindAllByID("0003")
		if err != nil {
			t.Fatalf("FindAllByID error: %v", err)
		}
		if len(paths) != 1 {
			t.Fatalf("FindAllByID returned %d paths, want 1: %v", len(paths), paths)
		}
		if paths[0] != want {
			t.Errorf("path = %q, want %q", paths[0], want)
		}
	})

	t.Run("collision two files same ID prefix returns slice of 2", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)

		tk := makeTicket("0001", testTitle)
		p1 := writeTicketFile(t, s.TicketsDir, "0001-fix-login-timeout-on.md", tk)
		p2 := writeTicketFile(t, s.TicketsDir, "0001-old-title.md", tk)

		paths, err := s.FindAllByID("0001")
		if err != nil {
			t.Fatalf("FindAllByID error: %v", err)
		}
		if len(paths) != 2 {
			t.Fatalf("FindAllByID returned %d paths, want 2: %v", len(paths), paths)
		}
		found := map[string]bool{p1: false, p2: false}
		for _, p := range paths {
			if _, ok := found[p]; !ok {
				t.Errorf("unexpected path %q in result", p)
			}
			found[p] = true
		}
		for p, seen := range found {
			if !seen {
				t.Errorf("expected path %q not found in result %v", p, paths)
			}
		}
	})

	t.Run("short id normalised to 4 digits", func(t *testing.T) {
		t.Parallel()
		s := newStore(t)

		tk := makeTicket("0005", testTitle)
		want := writeTicketFile(t, s.TicketsDir, "0005-fix-login-timeout-on.md", tk)

		paths, err := s.FindAllByID("5")
		if err != nil {
			t.Fatalf("FindAllByID error: %v", err)
		}
		if len(paths) != 1 {
			t.Fatalf("FindAllByID returned %d paths, want 1: %v", len(paths), paths)
		}
		if paths[0] != want {
			t.Errorf("path = %q, want %q", paths[0], want)
		}
	})
}

// ---- TestAllIDs ----

func TestAllIDs(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	writeTicketFile(t, s.TicketsDir, "0001-ticket.md", makeTicket("0001", testTitle))
	writeTicketFile(t, s.TicketsDir, "0002-ticket.md", makeTicket("0002", testTitle2))

	if err := os.MkdirAll(s.ArchiveDir, 0o755); err != nil {
		t.Fatalf("setup: mkdir archive: %v", err)
	}
	writeTicketFile(t, s.ArchiveDir, "0003-ticket.md", makeTicket("0003", testTitle3))

	ids, err := s.AllIDs()
	if err != nil {
		t.Fatalf("AllIDs error: %v", err)
	}
	want := map[string]bool{"0001": true, "0002": true, "0003": true}
	if len(ids) != len(want) {
		t.Errorf("got %d IDs (%v), want %d", len(ids), ids, len(want))
	}
	for _, id := range ids {
		if !want[id] {
			t.Errorf("unexpected ID %q in AllIDs result", id)
		}
	}
}

func TestAllIDsEmptyDirs(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	ids, err := s.AllIDs()
	if err != nil {
		t.Fatalf("AllIDs error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("AllIDs = %v, want empty slice", ids)
	}
}

// ---- TestWriteTicket ----

func TestWriteTicketAtomic(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	tk := makeTicket("0001", testTitle)
	finalPath := filepath.Join(s.TicketsDir, "0001-fix-login-timeout-on.md")
	tmpPath := finalPath + ".tmp"

	if err := s.WriteTicket(tk, finalPath); err != nil {
		t.Fatalf("WriteTicket error: %v", err)
	}

	// Temp file must NOT exist after a successful write.
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("temp file %q still exists after WriteTicket", tmpPath)
	}

	// Final file must exist.
	if _, err := os.Stat(finalPath); err != nil {
		t.Errorf("final file %q does not exist: %v", finalPath, err)
	}

	// Content must be parseable and match the original ticket.
	b, err := os.ReadFile(finalPath)
	if err != nil {
		t.Fatalf("read final file: %v", err)
	}
	parsed, err := ticket.Parse(b)
	if err != nil {
		t.Fatalf("parse written file: %v", err)
	}
	if parsed.ID != tk.ID {
		t.Errorf("parsed.ID = %q, want %q", parsed.ID, tk.ID)
	}
	if parsed.Title != tk.Title {
		t.Errorf("parsed.Title = %q, want %q", parsed.Title, tk.Title)
	}
}

func TestWriteTicketDoesNotModifyUpdated(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	// The store must not touch t.Updated — that is the caller's responsibility.
	sentinel := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	tk := makeTicket("0002", testTitle2)
	tk.Updated = sentinel

	path := filepath.Join(s.TicketsDir, "0002-add-feature-flag.md")
	if err := s.WriteTicket(tk, path); err != nil {
		t.Fatalf("WriteTicket error: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	parsed, err := ticket.Parse(b)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !parsed.Updated.Equal(sentinel) {
		t.Errorf("Updated = %v, want sentinel %v — store must not modify Updated", parsed.Updated, sentinel)
	}
}

// ---- TestReadTicket ----

func TestReadTicket(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	tk := makeTicket("0001", testTitle)
	path := writeTicketFile(t, s.TicketsDir, "0001-fix-login.md", tk)

	got, err := s.ReadTicket(path)
	if err != nil {
		t.Fatalf("ReadTicket error: %v", err)
	}
	if got.ID != "0001" {
		t.Errorf("ID = %q, want %q", got.ID, "0001")
	}
	if got.Title != testTitle {
		t.Errorf("Title = %q, want %q", got.Title, testTitle)
	}
}

func TestReadTicketNotFound(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	_, err := s.ReadTicket(filepath.Join(s.TicketsDir, "9999-missing.md"))
	if err == nil {
		t.Fatal("ReadTicket: expected error for missing file, got nil")
	}
}

// ---- TestTicketPath ----

func TestTicketPath(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	tests := []struct {
		name string
		id   int
		slug string
		want string
	}{
		{
			name: "zero-pads ID to 4 digits",
			id:   1,
			slug: "fix-login-timeout-on",
			want: filepath.Join(s.TicketsDir, "0001-fix-login-timeout-on.md"),
		},
		{
			name: "large ID still formatted",
			id:   42,
			slug: "my-ticket",
			want: filepath.Join(s.TicketsDir, "0042-my-ticket.md"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := s.TicketPath(tc.id, tc.slug)
			if got != tc.want {
				t.Errorf("TicketPath(%d, %q) = %q, want %q", tc.id, tc.slug, got, tc.want)
			}
		})
	}
}

// ---- TestMoveToArchive ----

func TestMoveToArchiveCreatesDir(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	// ArchiveDir does NOT exist yet — MoveToArchive must create it.
	if _, err := os.Stat(s.ArchiveDir); !os.IsNotExist(err) {
		t.Fatalf("setup: archive dir must not exist before test, got stat err: %v", err)
	}

	tk := makeTicket("0001", testTitle)
	src := writeTicketFile(t, s.TicketsDir, "0001-fix-login.md", tk)

	newPath, err := s.MoveToArchive(src)
	if err != nil {
		t.Fatalf("MoveToArchive error: %v", err)
	}

	// Archive dir must now exist.
	if _, err := os.Stat(s.ArchiveDir); err != nil {
		t.Errorf("archive dir does not exist after MoveToArchive: %v", err)
	}
	// File must be at new path.
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("archived file %q does not exist: %v", newPath, err)
	}
	// Source must be gone.
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source file %q still exists after MoveToArchive", src)
	}
	// New path should be inside ArchiveDir.
	if !strings.HasPrefix(newPath, s.ArchiveDir) {
		t.Errorf("newPath %q not inside ArchiveDir %q", newPath, s.ArchiveDir)
	}
}

func TestMoveToArchivePreservesFilename(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	tk := makeTicket("0002", testTitle2)
	src := writeTicketFile(t, s.TicketsDir, "0002-add-feature.md", tk)

	newPath, err := s.MoveToArchive(src)
	if err != nil {
		t.Fatalf("MoveToArchive error: %v", err)
	}

	want := filepath.Join(s.ArchiveDir, "0002-add-feature.md")
	if newPath != want {
		t.Errorf("newPath = %q, want %q", newPath, want)
	}
}

// ---- TestMoveToActive ----

func TestMoveToActive(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	if err := os.MkdirAll(s.ArchiveDir, 0o755); err != nil {
		t.Fatalf("setup: mkdir archive: %v", err)
	}

	tk := makeTicket("0001", testTitle)
	src := writeTicketFile(t, s.ArchiveDir, "0001-fix-login.md", tk)

	newPath, err := s.MoveToActive(src)
	if err != nil {
		t.Fatalf("MoveToActive error: %v", err)
	}

	want := filepath.Join(s.TicketsDir, "0001-fix-login.md")
	if newPath != want {
		t.Errorf("newPath = %q, want %q", newPath, want)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("active file %q does not exist: %v", newPath, err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source %q still exists after MoveToActive", src)
	}
}

func TestMoveToArchiveRefusesExistingDestination(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	// Pre-create the destination in ArchiveDir.
	if err := os.MkdirAll(s.ArchiveDir, 0o755); err != nil {
		t.Fatalf("setup: mkdir archive: %v", err)
	}
	tk := makeTicket("0001", testTitle)
	src := writeTicketFile(t, s.TicketsDir, "0001-fix-login.md", tk)
	writeTicketFile(t, s.ArchiveDir, "0001-fix-login.md", tk)

	_, err := s.MoveToArchive(src)
	if err == nil {
		t.Fatal("MoveToArchive: expected error when destination already exists, got nil")
	}
	if !strings.Contains(err.Error(), "destination already exists") {
		t.Errorf("error message %q does not contain 'destination already exists'", err.Error())
	}

	// Source file must still exist after the failed move.
	if _, statErr := os.Stat(src); statErr != nil {
		t.Errorf("source file %q must still exist after failed MoveToArchive: %v", src, statErr)
	}
}

func TestMoveToActiveRefusesExistingDestination(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	if err := os.MkdirAll(s.ArchiveDir, 0o755); err != nil {
		t.Fatalf("setup: mkdir archive: %v", err)
	}

	// Pre-create the destination in TicketsDir.
	tk := makeTicket("0001", testTitle)
	src := writeTicketFile(t, s.ArchiveDir, "0001-fix-login.md", tk)
	writeTicketFile(t, s.TicketsDir, "0001-fix-login.md", tk)

	_, err := s.MoveToActive(src)
	if err == nil {
		t.Fatal("MoveToActive: expected error when destination already exists, got nil")
	}
	if !strings.Contains(err.Error(), "destination already exists") {
		t.Errorf("error message %q does not contain 'destination already exists'", err.Error())
	}

	// Source file must still exist after the failed move.
	if _, statErr := os.Stat(src); statErr != nil {
		t.Errorf("source file %q must still exist after failed MoveToActive: %v", src, statErr)
	}
}

// ---- TestListActive ----

func TestListActive(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	writeTicketFile(t, s.TicketsDir, "0001-ticket-a.md", makeTicket("0001", testTitle))
	writeTicketFile(t, s.TicketsDir, "0002-ticket-b.md", makeTicket("0002", testTitle2))

	records, err := s.ListActive()
	if err != nil {
		t.Fatalf("ListActive error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListActive returned %d records, want 2", len(records))
	}
	for _, r := range records {
		if r.InArchive {
			t.Errorf("record %q has InArchive=true, want false", r.Path)
		}
		if r.Ticket == nil {
			t.Errorf("record %q has nil Ticket", r.Path)
		}
	}
}

func TestListActiveEmptyDir(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	records, err := s.ListActive()
	if err != nil {
		t.Fatalf("ListActive error: %v", err)
	}
	if records == nil {
		t.Error("ListActive returned nil, want empty non-nil slice")
	}
	if len(records) != 0 {
		t.Errorf("ListActive returned %d records, want 0", len(records))
	}
}

// ---- TestListArchive ----

func TestListArchive(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	if err := os.MkdirAll(s.ArchiveDir, 0o755); err != nil {
		t.Fatalf("setup: mkdir archive: %v", err)
	}

	writeTicketFile(t, s.ArchiveDir, "0003-old.md", makeTicket("0003", testTitle3))
	writeTicketFile(t, s.ArchiveDir, "0004-another.md", makeTicket("0004", testTitle))

	records, err := s.ListArchive()
	if err != nil {
		t.Fatalf("ListArchive error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListArchive returned %d records, want 2", len(records))
	}
	for _, r := range records {
		if !r.InArchive {
			t.Errorf("record %q has InArchive=false, want true", r.Path)
		}
		if r.Ticket == nil {
			t.Errorf("record %q has nil Ticket", r.Path)
		}
	}
}

func TestListArchiveEmptyOrAbsent(t *testing.T) {
	t.Parallel()
	s := newStore(t)

	// ArchiveDir does not exist at all.
	records, err := s.ListArchive()
	if err != nil {
		t.Fatalf("ListArchive error when dir absent: %v", err)
	}
	if records == nil {
		t.Error("ListArchive returned nil, want empty non-nil slice")
	}
	if len(records) != 0 {
		t.Errorf("ListArchive returned %d records, want 0", len(records))
	}
}
