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

// TestNewEmptyTitleRendersEmptyTitleField verifies that passing an empty title
// produces title: "" in the output (backwards-compatible behaviour).
func TestNewEmptyTitleRendersEmptyTitleField(t *testing.T) {
	t.Parallel()

	b, err := template.New(fixedTime, "", "")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	const want = `title: ""`
	if !strings.Contains(string(b), want) {
		t.Errorf("output does not contain %q:\n%s", want, string(b))
	}
}

// TestNewWithTitlePopulatesField verifies that a non-empty title is rendered
// into the frontmatter title field.
func TestNewWithTitlePopulatesField(t *testing.T) {
	t.Parallel()

	const testTitle = "My Title"
	b, err := template.New(fixedTime, "", testTitle)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	want := `title: "My Title"`
	if !strings.Contains(string(b), want) {
		t.Errorf("output does not contain %q:\n%s", want, string(b))
	}
}

// TestNewTitleTableDriven covers empty, non-empty, and special-character titles.
func TestNewTitleTableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		title string
		want  string
	}{
		{name: "empty title", title: "", want: `title: ""`},
		{name: "simple title", title: "Fix login timeout", want: `title: "Fix login timeout"`},
		{name: "title with numbers", title: "Bug 42", want: `title: "Bug 42"`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b, err := template.New(fixedTime, "", tc.title)
			if err != nil {
				t.Fatalf("New returned error: %v", err)
			}
			if !strings.Contains(string(b), tc.want) {
				t.Errorf("output does not contain %q:\n%s", tc.want, string(b))
			}
		})
	}
}
