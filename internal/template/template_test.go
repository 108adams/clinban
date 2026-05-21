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

	b, err := template.New(time.Now(), "")
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

	b, err := template.New(fixedTime, "")
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

	b, err := template.New(fixedTime, "task")
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

	b, err := template.New(fixedTime, "bug")
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

	b, err := template.New(fixedTime, "task")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	const wantComment = "# states: backlog, in-progress, blocked, done"
	if !strings.Contains(string(b), wantComment) {
		t.Errorf("template output does not contain %q:\n%s", wantComment, string(b))
	}
}
