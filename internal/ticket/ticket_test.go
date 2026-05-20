package ticket_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/108adams/clinban/internal/ticket"
)

// ---------------------------------------------------------------------------
// Fixture helpers
// ---------------------------------------------------------------------------

// fixedTime returns a deterministic RFC3339 time for use in fixture content.
// yaml.v3 marshals time.Time as RFC3339 with nanosecond precision when they
// are present, but as a plain date-time when they are not. To keep the
// round-trip byte-stable we use a time with no sub-second component.
func fixedTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic("fixedTime: " + err.Error())
	}
	return t
}

// fixtureTicket returns a canonical fixture ticket used across multiple tests.
func fixtureTicket() *ticket.Ticket {
	return &ticket.Ticket{
		ID:      "0042",
		Status:  ticket.StatusInProgress,
		Type:    ticket.TypeBug,
		Title:   "Fix login timeout on staging",
		Tags:    []string{},
		Created: fixedTime("2026-05-18T14:30:00Z"),
		Updated: fixedTime("2026-05-18T15:00:00Z"),
		Body:    "",
	}
}

// fixtureContent returns the canonical textual representation of fixtureTicket.
// This must match exactly what Marshal produces so that round-trip tests are valid.
const fixtureContent = `---
id: "0042"
status: in-progress
type: bug
title: Fix login timeout on staging
tags: []
created: 2026-05-18T14:30:00Z
updated: 2026-05-18T15:00:00Z
---
`

// fixtureContentWithBody is a ticket that has a non-empty markdown body.
const fixtureContentWithBody = `---
id: "0001"
status: backlog
type: task
title: A ticket with a body
tags: []
created: 2026-05-18T10:00:00Z
updated: 2026-05-18T10:00:00Z
---

## Details

Some markdown body here.
`

// fixtureContentWithTags has a non-empty tags list.
const fixtureContentWithTags = `---
id: "0007"
status: blocked
type: feature
title: Add tag support
tags:
    - alpha
    - beta
created: 2026-05-18T09:00:00Z
updated: 2026-05-18T09:00:00Z
---
`

// ---------------------------------------------------------------------------
// Status.Valid
// ---------------------------------------------------------------------------

func TestStatusValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input ticket.Status
		want  bool
	}{
		{name: "backlog", input: ticket.StatusBacklog, want: true},
		{name: "in-progress", input: ticket.StatusInProgress, want: true},
		{name: "blocked", input: ticket.StatusBlocked, want: true},
		{name: "done", input: ticket.StatusDone, want: true},
		{name: "unknown", input: ticket.Status("unknown"), want: false},
		{name: "empty", input: ticket.Status(""), want: false},
		{name: "BACKLOG uppercase", input: ticket.Status("BACKLOG"), want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.input.Valid()
			if got != tc.want {
				t.Errorf("Status(%q).Valid() = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Type.Valid
// ---------------------------------------------------------------------------

func TestTypeValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input ticket.Type
		want  bool
	}{
		{name: "bug", input: ticket.TypeBug, want: true},
		{name: "task", input: ticket.TypeTask, want: true},
		{name: "feature", input: ticket.TypeFeature, want: true},
		{name: "spike", input: ticket.TypeSpike, want: true},
		{name: "unknown", input: ticket.Type("unknown"), want: false},
		{name: "empty", input: ticket.Type(""), want: false},
		{name: "BUG uppercase", input: ticket.Type("BUG"), want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.input.Valid()
			if got != tc.want {
				t.Errorf("Type(%q).Valid() = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Parse
// ---------------------------------------------------------------------------

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("happy path no body", func(t *testing.T) {
		t.Parallel()
		tk, err := ticket.Parse([]byte(fixtureContent))
		if err != nil {
			t.Fatalf("Parse() unexpected error: %v", err)
		}
		if tk.ID != "0042" {
			t.Errorf("ID = %q, want %q", tk.ID, "0042")
		}
		if tk.Status != ticket.StatusInProgress {
			t.Errorf("Status = %q, want %q", tk.Status, ticket.StatusInProgress)
		}
		if tk.Type != ticket.TypeBug {
			t.Errorf("Type = %q, want %q", tk.Type, ticket.TypeBug)
		}
		if tk.Title != "Fix login timeout on staging" {
			t.Errorf("Title = %q, want %q", tk.Title, "Fix login timeout on staging")
		}
		if len(tk.Tags) != 0 {
			t.Errorf("Tags = %v, want empty slice", tk.Tags)
		}
		if tk.Body != "" {
			t.Errorf("Body = %q, want empty string", tk.Body)
		}
	})

	t.Run("happy path with body", func(t *testing.T) {
		t.Parallel()
		tk, err := ticket.Parse([]byte(fixtureContentWithBody))
		if err != nil {
			t.Fatalf("Parse() unexpected error: %v", err)
		}
		if tk.Title != "A ticket with a body" {
			t.Errorf("Title = %q, want %q", tk.Title, "A ticket with a body")
		}
		wantBody := "\n## Details\n\nSome markdown body here.\n"
		if tk.Body != wantBody {
			t.Errorf("Body = %q, want %q", tk.Body, wantBody)
		}
	})

	t.Run("happy path with tags", func(t *testing.T) {
		t.Parallel()
		tk, err := ticket.Parse([]byte(fixtureContentWithTags))
		if err != nil {
			t.Fatalf("Parse() unexpected error: %v", err)
		}
		if len(tk.Tags) != 2 {
			t.Fatalf("len(Tags) = %d, want 2", len(tk.Tags))
		}
		if tk.Tags[0] != "alpha" || tk.Tags[1] != "beta" {
			t.Errorf("Tags = %v, want [alpha beta]", tk.Tags)
		}
	})

	t.Run("timestamps decoded", func(t *testing.T) {
		t.Parallel()
		tk, err := ticket.Parse([]byte(fixtureContent))
		if err != nil {
			t.Fatalf("Parse() unexpected error: %v", err)
		}
		wantCreated := fixedTime("2026-05-18T14:30:00Z")
		wantUpdated := fixedTime("2026-05-18T15:00:00Z")
		if !tk.Created.Equal(wantCreated) {
			t.Errorf("Created = %v, want %v", tk.Created, wantCreated)
		}
		if !tk.Updated.Equal(wantUpdated) {
			t.Errorf("Updated = %v, want %v", tk.Updated, wantUpdated)
		}
	})
}

// ---------------------------------------------------------------------------
// TestParseEmptyBody (done criterion)
// ---------------------------------------------------------------------------

func TestParseEmptyBody(t *testing.T) {
	t.Parallel()

	const content = `---
id: "0001"
status: backlog
type: task
title: No body here
tags: []
created: 2026-05-18T10:00:00Z
updated: 2026-05-18T10:00:00Z
---
`
	tk, err := ticket.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}
	if tk.Body != "" {
		t.Errorf("Body = %q, want empty string", tk.Body)
	}
}

// ---------------------------------------------------------------------------
// TestParseMalformed (done criterion)
// ---------------------------------------------------------------------------

func TestParseMalformed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "no fences at all",
			content: "just plain text without any frontmatter fences",
		},
		{
			name:    "only opening fence",
			content: "---\nid: \"0001\"\n",
		},
		{
			name:    "invalid YAML in frontmatter",
			content: "---\n{invalid: yaml: [unclosed\n---\n",
		},
		{
			name:    "empty content",
			content: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ticket.Parse([]byte(tc.content))
			if err == nil {
				t.Fatalf("Parse(%q) returned nil error, want non-nil error", tc.content)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Marshal
// ---------------------------------------------------------------------------

func TestMarshal(t *testing.T) {
	t.Parallel()

	t.Run("empty tags serialises as flow sequence", func(t *testing.T) {
		t.Parallel()
		tk := fixtureTicket()
		tk.Tags = []string{}

		out, err := ticket.Marshal(tk)
		if err != nil {
			t.Fatalf("Marshal() unexpected error: %v", err)
		}
		if !strings.Contains(string(out), "tags: []") {
			t.Errorf("Marshal output does not contain 'tags: []':\n%s", out)
		}
	})

	t.Run("nil tags serialises as flow sequence", func(t *testing.T) {
		t.Parallel()
		tk := fixtureTicket()
		tk.Tags = nil

		out, err := ticket.Marshal(tk)
		if err != nil {
			t.Fatalf("Marshal() unexpected error: %v", err)
		}
		if !strings.Contains(string(out), "tags: []") {
			t.Errorf("Marshal output does not contain 'tags: []' for nil Tags:\n%s", out)
		}
	})

	t.Run("output is wrapped in fences", func(t *testing.T) {
		t.Parallel()
		tk := fixtureTicket()
		out, err := ticket.Marshal(tk)
		if err != nil {
			t.Fatalf("Marshal() unexpected error: %v", err)
		}
		s := string(out)
		if !strings.HasPrefix(s, "---\n") {
			t.Errorf("Marshal output does not start with '---\\n':\n%s", s)
		}
		if !strings.Contains(s, "\n---\n") {
			t.Errorf("Marshal output does not contain closing '---\\n':\n%s", s)
		}
	})

	t.Run("body is appended after closing fence", func(t *testing.T) {
		t.Parallel()
		tk := fixtureTicket()
		tk.Body = "\nsome body content\n"
		out, err := ticket.Marshal(tk)
		if err != nil {
			t.Fatalf("Marshal() unexpected error: %v", err)
		}
		s := string(out)
		if !strings.HasSuffix(s, "\nsome body content\n") {
			t.Errorf("Marshal output does not end with body:\n%s", s)
		}
	})
}

// ---------------------------------------------------------------------------
// TestParseRoundTrip (done criterion)
// ---------------------------------------------------------------------------

func TestParseRoundTrip(t *testing.T) {
	t.Parallel()

	fixtures := []struct {
		name    string
		content string
	}{
		{name: "no body", content: fixtureContent},
		{name: "with body", content: fixtureContentWithBody},
		{name: "with tags", content: fixtureContentWithTags},
	}

	for _, fx := range fixtures {
		t.Run(fx.name, func(t *testing.T) {
			t.Parallel()

			// First parse
			tk1, err := ticket.Parse([]byte(fx.content))
			if err != nil {
				t.Fatalf("first Parse() error: %v", err)
			}

			// Marshal
			out, err := ticket.Marshal(tk1)
			if err != nil {
				t.Fatalf("Marshal() error: %v", err)
			}

			// Second parse
			tk2, err := ticket.Parse(out)
			if err != nil {
				t.Fatalf("second Parse() error: %v", err)
			}

			// Assert field equality between both parsed tickets
			assertTicketsEqual(t, tk1, tk2)
		})
	}
}

// assertTicketsEqual compares two Ticket values field-by-field and reports
// any differences through t.Errorf.
func assertTicketsEqual(t *testing.T, a, b *ticket.Ticket) {
	t.Helper()

	if a.ID != b.ID {
		t.Errorf("ID: got %q, want %q", b.ID, a.ID)
	}
	if a.Status != b.Status {
		t.Errorf("Status: got %q, want %q", b.Status, a.Status)
	}
	if a.Type != b.Type {
		t.Errorf("Type: got %q, want %q", b.Type, a.Type)
	}
	if a.Title != b.Title {
		t.Errorf("Title: got %q, want %q", b.Title, a.Title)
	}
	if len(a.Tags) != len(b.Tags) {
		t.Errorf("Tags length: got %d, want %d", len(b.Tags), len(a.Tags))
	} else {
		for i := range a.Tags {
			if a.Tags[i] != b.Tags[i] {
				t.Errorf("Tags[%d]: got %q, want %q", i, b.Tags[i], a.Tags[i])
			}
		}
	}
	if !a.Created.Equal(b.Created) {
		t.Errorf("Created: got %v, want %v", b.Created, a.Created)
	}
	if !a.Updated.Equal(b.Updated) {
		t.Errorf("Updated: got %v, want %v", b.Updated, a.Updated)
	}
	if a.Body != b.Body {
		t.Errorf("Body: got %q, want %q", b.Body, a.Body)
	}
}

// ---------------------------------------------------------------------------
// ErrMissingFrontmatter sentinel check
// ---------------------------------------------------------------------------

func TestParseMissingFrontmatterSentinel(t *testing.T) {
	t.Parallel()

	_, err := ticket.Parse([]byte("no fences here"))
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !errors.Is(err, ticket.ErrMissingFrontmatter) {
		t.Errorf("error is %v; want errors.Is(..., ErrMissingFrontmatter) to be true", err)
	}
}
