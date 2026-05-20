package main_test

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Constants for show tests.
const (
	showTestID    = "0042"
	showTestTitle = "Fix login timeout on staging"
	showTestType  = "bug"
	showTestTag1  = "auth"
	showTestTag2  = "urgent"
)

// ticketWithBody returns a valid ticket file body including a markdown body section.
func ticketWithBody(id, title, ticketType string, tags []string, body string) string {
	now := time.Now().UTC().Format(time.RFC3339)
	tagsStr := "[]"
	if len(tags) > 0 {
		tagsStr = fmt.Sprintf("[%s]", strings.Join(func() []string {
			quoted := make([]string, len(tags))
			for i, t := range tags {
				quoted[i] = fmt.Sprintf("%q", t)
			}
			return quoted
		}(), ", "))
	}
	content := fmt.Sprintf(`---
id: "%s"
status: backlog
type: %s
title: %s
tags: %s
created: %s
updated: %s
---
`, id, ticketType, title, tagsStr, now, now)
	if body != "" {
		content += "\n" + body
	}
	return content
}

// ticketNoTags returns a valid ticket with no tags.
func ticketNoTags(id, title, ticketType string) string {
	return ticketWithBody(id, title, ticketType, nil, "")
}

// ticketWithTags returns a valid ticket with the given tags.
func ticketWithTagsContent(id, title, ticketType string, tags []string) string {
	return ticketWithBody(id, title, ticketType, tags, "")
}

// ticketWithBodyContent returns a valid ticket with body text.
func ticketWithBodyContent(id, title, ticketType, body string) string {
	return ticketWithBody(id, title, ticketType, nil, body)
}

// runShow executes the clinban binary with "show <id>" in workDir and returns
// stdout, stderr, and the exit code.
func runShow(t *testing.T, bin, workDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmdArgs := append([]string{"show"}, args...)
	cmd := exec.Command(bin, cmdArgs...)
	cmd.Dir = workDir
	cmd.Env = coverEnv()
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return stdout, stderr, exitCode
}

// TestShowHappyPath tests that "clinban show <id>" prints all expected fields
// and exits 0 for a valid active ticket.
func TestShowHappyPath(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	filename := fmt.Sprintf("%s-fix-login-timeout-on-staging.md", showTestID)
	writeTicket(t, ticketsDir, filename, ticketNoTags(showTestID, showTestTitle, showTestType))

	stdout, stderr, code := runShow(t, bin, root, showTestID)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	tests := []struct {
		label   string
		contain string
	}{
		{"ID line", "ID:"},
		{"ID value", showTestID},
		{"Status line", "Status:"},
		{"Type line", "Type:"},
		{"Type value", showTestType},
		{"Title line", "Title:"},
		{"Title value", showTestTitle},
		{"Created line", "Created:"},
		{"Updated line", "Updated:"},
	}
	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			if !strings.Contains(stdout, tc.contain) {
				t.Errorf("stdout does not contain %q:\n%s", tc.contain, stdout)
			}
		})
	}
}

// TestShowNoTagsOmitsTagLine tests that the Tags line is omitted when Tags is empty.
func TestShowNoTagsOmitsTagLine(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	filename := fmt.Sprintf("%s-fix-login-timeout-on-staging.md", showTestID)
	writeTicket(t, ticketsDir, filename, ticketNoTags(showTestID, showTestTitle, showTestType))

	stdout, _, code := runShow(t, bin, root, showTestID)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q", code, stdout)
	}
	if strings.Contains(stdout, "Tags:") {
		t.Errorf("stdout should NOT contain 'Tags:' when tags are empty, got:\n%s", stdout)
	}
}

// TestShowWithTags tests that the Tags line is present and correctly formatted
// when the ticket has tags.
func TestShowWithTags(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	tags := []string{showTestTag1, showTestTag2}
	filename := fmt.Sprintf("%s-fix-login-timeout-on-staging.md", showTestID)
	writeTicket(t, ticketsDir, filename, ticketWithTagsContent(showTestID, showTestTitle, showTestType, tags))

	stdout, _, code := runShow(t, bin, root, showTestID)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q", code, stdout)
	}
	if !strings.Contains(stdout, "Tags:") {
		t.Errorf("stdout should contain 'Tags:' when tags are present, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, showTestTag1) {
		t.Errorf("stdout should contain tag %q, got:\n%s", showTestTag1, stdout)
	}
	if !strings.Contains(stdout, showTestTag2) {
		t.Errorf("stdout should contain tag %q, got:\n%s", showTestTag2, stdout)
	}
	// Tags should be comma-separated.
	if !strings.Contains(stdout, ", ") {
		t.Errorf("tags should be comma-separated, got:\n%s", stdout)
	}
}

// TestShowWithBody tests that a non-empty body appears after a blank line.
func TestShowWithBody(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	const body = "This is the ticket body.\n\nIt has multiple paragraphs."
	filename := fmt.Sprintf("%s-fix-login-timeout-on-staging.md", showTestID)
	writeTicket(t, ticketsDir, filename, ticketWithBodyContent(showTestID, showTestTitle, showTestType, body))

	stdout, _, code := runShow(t, bin, root, showTestID)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q", code, stdout)
	}
	if !strings.Contains(stdout, "This is the ticket body.") {
		t.Errorf("stdout should contain body text, got:\n%s", stdout)
	}
}

// TestShowArchivedTicket tests that "clinban show <id>" finds a ticket in the
// archive directory and prints the [archived] label.
func TestShowArchivedTicket(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, archiveDir := setupWorkDir(t)

	filename := fmt.Sprintf("%s-fix-login-timeout-on-staging.md", showTestID)
	writeTicket(t, archiveDir, filename, ticketNoTags(showTestID, showTestTitle, showTestType))

	stdout, _, code := runShow(t, bin, root, showTestID)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q", code, stdout)
	}
	if !strings.Contains(stdout, "[archived]") {
		t.Errorf("stdout should contain '[archived]' for archived ticket, got:\n%s", stdout)
	}
}

// TestShowArchivedTicketNoArchivedLabelForActive tests that an active ticket
// does NOT show the [archived] label.
func TestShowArchivedTicketNoArchivedLabelForActive(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	filename := fmt.Sprintf("%s-fix-login-timeout-on-staging.md", showTestID)
	writeTicket(t, ticketsDir, filename, ticketNoTags(showTestID, showTestTitle, showTestType))

	stdout, _, code := runShow(t, bin, root, showTestID)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q", code, stdout)
	}
	if strings.Contains(stdout, "[archived]") {
		t.Errorf("stdout should NOT contain '[archived]' for active ticket, got:\n%s", stdout)
	}
}

// TestShowUnknownID tests that "clinban show <id>" prints "ticket not found" to
// stderr and exits 1 when the ID does not exist.
func TestShowUnknownID(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	// No tickets in the directory.
	_, stderr, code := runShow(t, bin, root, "9999")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(strings.ToLower(stderr), "ticket not found") {
		t.Errorf("stderr = %q, want 'ticket not found'", stderr)
	}
}

// TestShowOutputFieldOrder tests that the output fields appear in the documented
// order: ID, Status, Type, Title, [Tags], Created, Updated, [archived], [body].
func TestShowOutputFieldOrder(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	tags := []string{"alpha"}
	const body = "Some body text."
	filename := fmt.Sprintf("%s-fix-login-timeout-on-staging.md", showTestID)
	writeTicket(t, ticketsDir, filename, ticketWithBody(showTestID, showTestTitle, showTestType, tags, body))

	stdout, _, code := runShow(t, bin, root, showTestID)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q", code, stdout)
	}

	// Verify order by finding positions.
	posID := strings.Index(stdout, "ID:")
	posStatus := strings.Index(stdout, "Status:")
	posType := strings.Index(stdout, "Type:")
	posTitle := strings.Index(stdout, "Title:")
	posTags := strings.Index(stdout, "Tags:")
	posCreated := strings.Index(stdout, "Created:")
	posUpdated := strings.Index(stdout, "Updated:")
	posBody := strings.Index(stdout, body)

	orderedChecks := []struct {
		name  string
		left  int
		right int
		lname string
		rname string
	}{
		{"ID before Status", posID, posStatus, "ID:", "Status:"},
		{"Status before Type", posStatus, posType, "Status:", "Type:"},
		{"Type before Title", posType, posTitle, "Type:", "Title:"},
		{"Title before Tags", posTitle, posTags, "Title:", "Tags:"},
		{"Tags before Created", posTags, posCreated, "Tags:", "Created:"},
		{"Created before Updated", posCreated, posUpdated, "Created:", "Updated:"},
		{"Updated before body", posUpdated, posBody, "Updated:", "body"},
	}
	for _, tc := range orderedChecks {
		t.Run(tc.name, func(t *testing.T) {
			if tc.left == -1 {
				t.Errorf("%q not found in output", tc.lname)
				return
			}
			if tc.right == -1 {
				t.Errorf("%q not found in output", tc.rname)
				return
			}
			if tc.left >= tc.right {
				t.Errorf("%q (pos %d) should appear before %q (pos %d) in output:\n%s",
					tc.lname, tc.left, tc.rname, tc.right, stdout)
			}
		})
	}
}

// TestShowNoArgsError tests that "clinban show" with no arguments produces an
// error (Cobra validates ExactArgs(1)).
func TestShowNoArgsError(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	_, _, code := runShow(t, bin, root)

	if code == 0 {
		t.Error("exit code = 0, want non-zero for missing argument")
	}
}

// TestShowTimestampsRFC3339 tests that Created and Updated are formatted as RFC3339.
func TestShowTimestampsRFC3339(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	filename := fmt.Sprintf("%s-fix-login-timeout-on-staging.md", showTestID)
	writeTicket(t, ticketsDir, filename, ticketNoTags(showTestID, showTestTitle, showTestType))

	stdout, _, code := runShow(t, bin, root, showTestID)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q", code, stdout)
	}

	// Find the Created line and check the timestamp format.
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Created:") || strings.HasPrefix(line, "Updated:") {
			// Should contain a T character (RFC3339 separator) and a Z or +offset.
			if !strings.ContainsAny(line, "TZ+") {
				t.Errorf("timestamp line does not look like RFC3339: %q", line)
			}
		}
	}
}
