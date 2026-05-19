package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Constants for new command tests.
const (
	newTestTitle    = "Fix login timeout on staging"
	newTestType     = "bug"
	newTestBody     = "This is the ticket body."
	newTestTag1     = "backend"
	newTestTag2     = "auth"
	newExpectedID   = "0001"
	newExpectedSlug = "fix-login-timeout-on-staging"
)

// runNew executes "clinban new --no-interactive [args...]" in workDir and
// returns stdout, stderr, and exit code.
func runNew(t *testing.T, bin, workDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmdArgs := append([]string{"new", "--no-interactive"}, args...)
	cmd := exec.Command(bin, cmdArgs...)
	cmd.Dir = workDir
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

// TestNewNoInteractiveHappyPath verifies that a ticket is created and the
// filename is printed to stdout when --title and --type are both provided.
func TestNewNoInteractiveHappyPath(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	stdout, stderr, code := runNew(t, bin, dir,
		"--title", newTestTitle,
		"--type", newTestType,
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	// Output must start with "created: ".
	if !strings.HasPrefix(strings.TrimSpace(stdout), "created:") {
		t.Errorf("stdout = %q, want prefix 'created:'", stdout)
	}

	// The expected filename.
	wantFile := fmt.Sprintf("%s-%s.md", newExpectedID, newExpectedSlug)
	if !strings.Contains(stdout, wantFile) {
		t.Errorf("stdout = %q, want to contain %q", stdout, wantFile)
	}

	// The ticket file must exist in the working directory.
	ticketPath := filepath.Join(dir, wantFile)
	if _, err := os.Stat(ticketPath); os.IsNotExist(err) {
		t.Errorf("ticket file %q not found in %q", wantFile, dir)
	}
}

// TestNewNoInteractiveCreatesCorrectContent verifies that the written ticket
// file contains correct frontmatter fields.
func TestNewNoInteractiveCreatesCorrectContent(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	_, _, code := runNew(t, bin, dir,
		"--title", newTestTitle,
		"--type", newTestType,
		"--body", newTestBody,
		"--tags", fmt.Sprintf("%s,%s", newTestTag1, newTestTag2),
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	wantFile := fmt.Sprintf("%s-%s.md", newExpectedID, newExpectedSlug)
	ticketPath := filepath.Join(dir, wantFile)

	content, err := os.ReadFile(ticketPath)
	if err != nil {
		t.Fatalf("reading ticket file: %v", err)
	}
	body := string(content)

	checks := []struct {
		desc    string
		wantStr string
	}{
		{"id field", fmt.Sprintf(`id: "%s"`, newExpectedID)},
		{"status field", `status: backlog`},
		{"type field", fmt.Sprintf(`type: %s`, newTestType)},
		{"title field", fmt.Sprintf(`title: %s`, newTestTitle)},
		{"tag1", newTestTag1},
		{"tag2", newTestTag2},
		{"body", newTestBody},
	}

	for _, c := range checks {
		if !strings.Contains(body, c.wantStr) {
			t.Errorf("%s: file content does not contain %q\nfull content:\n%s", c.desc, c.wantStr, body)
		}
	}
}

// TestNewNoInteractiveMissingTitle verifies exit 1 and stderr error when
// --title is omitted.
func TestNewNoInteractiveMissingTitle(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	stdout, stderr, code := runNew(t, bin, dir, "--type", newTestType)

	if code != 1 {
		t.Errorf("exit code = %d, want 1; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "--title") {
		t.Errorf("stderr = %q, want mention of '--title'", stderr)
	}
}

// TestNewNoInteractiveMissingType verifies exit 1 and stderr error when
// --type is omitted.
func TestNewNoInteractiveMissingType(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	stdout, stderr, code := runNew(t, bin, dir, "--title", newTestTitle)

	if code != 1 {
		t.Errorf("exit code = %d, want 1; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "--type") {
		t.Errorf("stderr = %q, want mention of '--type'", stderr)
	}
}

// TestNewNoInteractiveInvalidType verifies exit 1 and stderr error when
// --type is not one of the valid values.
func TestNewNoInteractiveInvalidType(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	invalidType := "wishlist"
	stdout, stderr, code := runNew(t, bin, dir,
		"--title", newTestTitle,
		"--type", invalidType,
	)

	if code != 1 {
		t.Errorf("exit code = %d, want 1; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, invalidType) {
		t.Errorf("stderr = %q, want mention of the invalid type value", stderr)
	}
}

// TestNewNoInteractiveAllTypes verifies each valid type value is accepted.
func TestNewNoInteractiveAllTypes(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)

	validTypes := []string{"bug", "task", "feature", "spike"}

	for _, tt := range validTypes {
		tt := tt // capture for parallel subtests
		t.Run(tt, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			_, stderr, code := runNew(t, bin, dir,
				"--title", newTestTitle,
				"--type", tt,
			)
			if code != 0 {
				t.Errorf("type=%q: exit code = %d, want 0; stderr=%q", tt, code, stderr)
			}
		})
	}
}

// TestNewNoInteractiveIDAssignment verifies that when tickets already exist,
// the new ticket gets the next sequential ID.
func TestNewNoInteractiveIDAssignment(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	// Pre-populate with ticket 0001.
	writeTicket(t, dir, "0001-existing-ticket.md", validTicketContent("0001"))

	stdout, stderr, code := runNew(t, bin, dir,
		"--title", "Next ticket title",
		"--type", "task",
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	// Expect ID 0002.
	if !strings.Contains(stdout, "0002") {
		t.Errorf("stdout = %q, want to contain '0002'", stdout)
	}

	wantFile := "0002-next-ticket-title.md"
	ticketPath := filepath.Join(dir, wantFile)
	if _, err := os.Stat(ticketPath); os.IsNotExist(err) {
		t.Errorf("expected ticket file %q not found", wantFile)
	}
}

// TestNewNoInteractiveWithoutBody verifies that the optional --body flag
// produces a ticket without body content when omitted.
func TestNewNoInteractiveWithoutBody(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	_, _, code := runNew(t, bin, dir,
		"--title", newTestTitle,
		"--type", newTestType,
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	wantFile := fmt.Sprintf("%s-%s.md", newExpectedID, newExpectedSlug)
	content, err := os.ReadFile(filepath.Join(dir, wantFile))
	if err != nil {
		t.Fatalf("reading ticket: %v", err)
	}
	// Body should be empty — the file must end immediately after the closing fence.
	// We just check that parsing works and body is absent.
	if strings.Contains(string(content), newTestBody) {
		t.Errorf("body appeared in file when --body was not set")
	}
}

// TestNewNoInteractiveOutputToStdout verifies that the "created: ..." message
// goes to stdout (not stderr) on success.
func TestNewNoInteractiveOutputToStdout(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	stdout, stderr, code := runNew(t, bin, dir,
		"--title", newTestTitle,
		"--type", newTestType,
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "created:") {
		t.Errorf("'created:' not on stdout; stdout=%q stderr=%q", stdout, stderr)
	}
	// Nothing meaningful on stderr.
	if strings.Contains(stderr, "created:") {
		t.Errorf("'created:' appeared on stderr instead of stdout")
	}
}

// TestNewNoInteractiveTagsParsed verifies that comma-separated tags are
// stored individually in the ticket frontmatter.
func TestNewNoInteractiveTagsParsed(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	_, _, code := runNew(t, bin, dir,
		"--title", newTestTitle,
		"--type", newTestType,
		"--tags", "alpha,beta,gamma",
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	wantFile := fmt.Sprintf("%s-%s.md", newExpectedID, newExpectedSlug)
	content, err := os.ReadFile(filepath.Join(dir, wantFile))
	if err != nil {
		t.Fatalf("reading ticket: %v", err)
	}
	body := string(content)

	for _, tag := range []string{"alpha", "beta", "gamma"} {
		if !strings.Contains(body, tag) {
			t.Errorf("tag %q not found in file content:\n%s", tag, body)
		}
	}
}

// TestNewNoInteractiveStatusIsBacklog verifies that newly created tickets
// always have status set to "backlog".
func TestNewNoInteractiveStatusIsBacklog(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	_, _, code := runNew(t, bin, dir,
		"--title", newTestTitle,
		"--type", newTestType,
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	wantFile := fmt.Sprintf("%s-%s.md", newExpectedID, newExpectedSlug)
	content, err := os.ReadFile(filepath.Join(dir, wantFile))
	if err != nil {
		t.Fatalf("reading ticket: %v", err)
	}
	if !strings.Contains(string(content), "status: backlog") {
		t.Errorf("status is not 'backlog' in file:\n%s", string(content))
	}
}

// TestNewNoInteractiveFourDigitPaddedID verifies that the ID is zero-padded
// to exactly 4 digits.
func TestNewNoInteractiveFourDigitPaddedID(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	_, _, code := runNew(t, bin, dir,
		"--title", newTestTitle,
		"--type", newTestType,
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	wantFile := fmt.Sprintf("%s-%s.md", newExpectedID, newExpectedSlug)
	content, err := os.ReadFile(filepath.Join(dir, wantFile))
	if err != nil {
		t.Fatalf("reading ticket: %v", err)
	}
	// The id field must be "0001" (4-digit zero-padded).
	if !strings.Contains(string(content), `id: "0001"`) {
		t.Errorf("expected 4-digit zero-padded id '0001' in file:\n%s", string(content))
	}
}

// TestNewNoInteractiveEmptyTitle verifies that an explicitly empty --title
// flag causes exit 1.
func TestNewNoInteractiveEmptyTitle(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	_, stderr, code := runNew(t, bin, dir,
		"--title", "",
		"--type", newTestType,
	)
	if code != 1 {
		t.Errorf("exit code = %d, want 1; stderr=%q", code, stderr)
	}
}

// TestNewNoInteractiveEmptyType verifies that an explicitly empty --type flag
// causes exit 1.
func TestNewNoInteractiveEmptyType(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	_, stderr, code := runNew(t, bin, dir,
		"--title", newTestTitle,
		"--type", "",
	)
	if code != 1 {
		t.Errorf("exit code = %d, want 1; stderr=%q", code, stderr)
	}
}

// TestNewNoInteractiveNoTmpFileLeft verifies that no *.tmp file is left
// behind after a successful write (atomic write contract).
func TestNewNoInteractiveNoTmpFileLeft(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	dir := t.TempDir()

	_, _, code := runNew(t, bin, dir,
		"--title", newTestTitle,
		"--type", newTestType,
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temp file left behind: %q", e.Name())
		}
	}
}
