package template_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/108adams/clinban/internal/template"
	"github.com/108adams/clinban/internal/ticket"
)

const testID = 42

func TestNewReturnsParseableTicket(t *testing.T) {
	t.Parallel()

	b, err := template.New(1, time.Now(), "")
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

func TestNewContainsIDAndTimestamp(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	b, err := template.New(testID, fixedTime, "")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	got := string(b)

	// ID is rendered as a zero-padded 4-digit string, e.g. "0042"
	wantID := fmt.Sprintf("%04d", testID)
	if !strings.Contains(got, wantID) {
		t.Errorf("output does not contain ID %q:\n%s", wantID, got)
	}

	// Timestamp is rendered in RFC3339 format
	wantTS := fixedTime.Format(time.RFC3339)
	if !strings.Contains(got, wantTS) {
		t.Errorf("output does not contain timestamp %q:\n%s", wantTS, got)
	}
}

func TestNewWithDefaultType(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	b, err := template.New(1, fixedTime, "bug")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	got := string(b)
	want := `type: "bug"`
	if !strings.Contains(got, want) {
		t.Errorf("output does not contain %q:\n%s", want, got)
	}
}

// TestNewTemplateFieldOrder verifies that "title:" appears before "id:" in the
// rendered template output.
func TestNewTemplateFieldOrder(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	b, err := template.New(testID, fixedTime, "task")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	got := string(b)

	titleIdx := strings.Index(got, "title:")
	idIdx := strings.Index(got, "id:")

	if titleIdx == -1 {
		t.Fatal("template output does not contain 'title:'")
	}
	if idIdx == -1 {
		t.Fatal("template output does not contain 'id:'")
	}
	if titleIdx >= idIdx {
		t.Errorf("'title:' (offset %d) must appear before 'id:' (offset %d) in template output:\n%s", titleIdx, idIdx, got)
	}
}

// TestNewTemplateStatesComment verifies that the rendered template includes the
// states hint comment below the status field.
func TestNewTemplateStatesComment(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	b, err := template.New(testID, fixedTime, "task")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	const wantComment = "# states: backlog, in-progress, blocked, done"
	if !strings.Contains(string(b), wantComment) {
		t.Errorf("template output does not contain %q:\n%s", wantComment, string(b))
	}
}
