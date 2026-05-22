package template_test

import (
	"strings"
	"testing"
	"time"

	"github.com/108adams/clinban/internal/template"
	"github.com/108adams/clinban/internal/ticket"
)

var fixedTime = time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)

func TestNewReturnsParseableTicket(t *testing.T) {
	t.Parallel()

	b, err := template.New(time.Now(), "", "")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("New returned empty bytes")
	}
	if _, err := ticket.Parse(b); err != nil {
		t.Errorf("ticket.Parse failed on New output: %v", err)
	}
}

func TestNewContainsTimestamp(t *testing.T) {
	t.Parallel()

	b, err := template.New(fixedTime, "", "")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	got := string(b)

	// Timestamp is rendered in RFC3339 format
	wantTS := fixedTime.Format(time.RFC3339)
	if !strings.Contains(got, wantTS) {
		t.Errorf("output does not contain timestamp %q:\n%s", wantTS, got)
	}
}

func TestNewOutputContainsNoIDField(t *testing.T) {
	t.Parallel()

	b, err := template.New(fixedTime, "task", "")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	got := string(b)
	if strings.Contains(got, "id:") {
		t.Errorf("template output must not contain 'id:' field, got:\n%s", got)
	}
}

func TestNewWithDefaultType(t *testing.T) {
	t.Parallel()

	b, err := template.New(fixedTime, "bug", "")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	got := string(b)
	want := `type: "bug"`
	if !strings.Contains(got, want) {
		t.Errorf("output does not contain %q:\n%s", want, got)
	}
}

// TestNewTemplateStatesComment verifies that the rendered template includes the
// states hint comment below the status field.
func TestNewTemplateStatesComment(t *testing.T) {
	t.Parallel()

	b, err := template.New(fixedTime, "task", "")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	const wantComment = "# states: backlog, in-progress, blocked, done"
	if !strings.Contains(string(b), wantComment) {
		t.Errorf("template output does not contain %q:\n%s", wantComment, string(b))
	}
}

// TestNewTitleIsDoubleQuoted verifies that the title field is always rendered as
// a double-quoted YAML scalar, consistent with status, type, created, and updated.
func TestNewTitleIsDoubleQuoted(t *testing.T) {
	t.Parallel()

	b, err := template.New(fixedTime, "", "title not quoted")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	want := `title: "title not quoted"`
	if !strings.Contains(string(b), want) {
		t.Errorf("title field not double-quoted:\n  want substring: %s\n  got:\n%s", want, string(b))
	}
}

// TestNewTitleRoundtrip verifies that for every title string, New renders it
// as valid YAML and ticket.Parse recovers the exact original string.
//
// This replaces the old string-containment title assertions. The roundtrip
// approach is more robust: it catches any yaml encoding/decoding mismatch
// regardless of whether the rendered form uses quotes, escapes, etc.
func TestNewTitleRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		title string
	}{
		{name: "empty string", title: ""},
		{name: "simple string", title: "Fix login timeout on staging"},
		{name: "string with double quote", title: `Fix "quoted" case`},
		{name: "string with single quote", title: "it's broken"},
		{name: "string with colon-space", title: "feat: add new flag"},
		{name: "string with backslash", title: `path\to\file`},
		{name: "string with newline", title: "line one\nline two"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b, err := template.New(fixedTime, "", tc.title)
			if err != nil {
				t.Fatalf("New(%q) returned error: %v", tc.title, err)
			}

			parsed, err := ticket.Parse(b)
			if err != nil {
				t.Fatalf("ticket.Parse failed for title %q: %v\nrendered:\n%s", tc.title, err, string(b))
			}

			if parsed.Title != tc.title {
				t.Errorf("roundtrip mismatch for %q:\n  got  %q\n  want %q\nrendered:\n%s",
					tc.name, parsed.Title, tc.title, string(b))
			}
		})
	}
}
