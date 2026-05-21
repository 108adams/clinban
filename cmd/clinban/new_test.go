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

// TestNewNoInteractiveHappyPath verifies that a ticket is created and the
// filename is printed to stdout when --title and --type are both provided.
func TestNewNoInteractiveHappyPath(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	stdout, stderr, code := runNew(t, bin, root,
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

	// The ticket file must exist in the tickets directory.
	ticketPath := filepath.Join(ticketsDir, wantFile)
	if _, err := os.Stat(ticketPath); os.IsNotExist(err) {
		t.Errorf("ticket file %q not found in %q", wantFile, ticketsDir)
	}
}

// TestNewNoInteractiveCreatesCorrectContent verifies that the written ticket
// file contains correct frontmatter fields.
func TestNewNoInteractiveCreatesCorrectContent(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	_, _, code := runNew(t, bin, root,
		"--title", newTestTitle,
		"--type", newTestType,
		"--body", newTestBody,
		"--tags", fmt.Sprintf("%s,%s", newTestTag1, newTestTag2),
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	wantFile := fmt.Sprintf("%s-%s.md", newExpectedID, newExpectedSlug)
	ticketPath := filepath.Join(ticketsDir, wantFile)

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
	root, _, _ := setupWorkDir(t)

	stdout, stderr, code := runNew(t, bin, root, "--type", newTestType)

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
	root, _, _ := setupWorkDir(t)

	stdout, stderr, code := runNew(t, bin, root, "--title", newTestTitle)

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
	root, _, _ := setupWorkDir(t)

	invalidType := "wishlist"
	stdout, stderr, code := runNew(t, bin, root,
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
			root, _, _ := setupWorkDir(t)
			_, stderr, code := runNew(t, bin, root,
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
	root, ticketsDir, _ := setupWorkDir(t)

	// Pre-populate with ticket 0001.
	writeTicket(t, ticketsDir, "0001-existing-ticket.md", validTicketContent("0001"))

	stdout, stderr, code := runNew(t, bin, root,
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
	ticketPath := filepath.Join(ticketsDir, wantFile)
	if _, err := os.Stat(ticketPath); os.IsNotExist(err) {
		t.Errorf("expected ticket file %q not found", wantFile)
	}
}

// TestNewNoInteractiveWithoutBody verifies that the optional --body flag
// produces a ticket without body content when omitted.
func TestNewNoInteractiveWithoutBody(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)

	_, _, code := runNew(t, bin, root,
		"--title", newTestTitle,
		"--type", newTestType,
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	wantFile := fmt.Sprintf("%s-%s.md", newExpectedID, newExpectedSlug)
	content, err := os.ReadFile(filepath.Join(ticketsDir, wantFile))
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
	root, _, _ := setupWorkDir(t)

	stdout, stderr, code := runNew(t, bin, root,
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
	root, ticketsDir, _ := setupWorkDir(t)

	_, _, code := runNew(t, bin, root,
		"--title", newTestTitle,
		"--type", newTestType,
		"--tags", "alpha,beta,gamma",
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	wantFile := fmt.Sprintf("%s-%s.md", newExpectedID, newExpectedSlug)
	content, err := os.ReadFile(filepath.Join(ticketsDir, wantFile))
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
	root, ticketsDir, _ := setupWorkDir(t)

	_, _, code := runNew(t, bin, root,
		"--title", newTestTitle,
		"--type", newTestType,
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	wantFile := fmt.Sprintf("%s-%s.md", newExpectedID, newExpectedSlug)
	content, err := os.ReadFile(filepath.Join(ticketsDir, wantFile))
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
	root, ticketsDir, _ := setupWorkDir(t)

	_, _, code := runNew(t, bin, root,
		"--title", newTestTitle,
		"--type", newTestType,
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	wantFile := fmt.Sprintf("%s-%s.md", newExpectedID, newExpectedSlug)
	content, err := os.ReadFile(filepath.Join(ticketsDir, wantFile))
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
	root, _, _ := setupWorkDir(t)

	_, stderr, code := runNew(t, bin, root,
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
	root, _, _ := setupWorkDir(t)

	_, stderr, code := runNew(t, bin, root,
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
	root, ticketsDir, _ := setupWorkDir(t)

	_, _, code := runNew(t, bin, root,
		"--title", newTestTitle,
		"--type", newTestType,
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	entries, err := os.ReadDir(ticketsDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temp file left behind: %q", e.Name())
		}
	}
}

// --- Default type fallback tests ---

// setupWorkDirWithConfig creates a temp work dir like setupWorkDir but also
// writes a .clinban config file with the given content into the root.
func setupWorkDirWithConfig(t *testing.T, configContent string) (root, ticketsDir, archiveDir string) {
	t.Helper()
	root, ticketsDir, archiveDir = setupWorkDir(t)
	cfgPath := filepath.Join(root, ".clinban")
	if err := os.WriteFile(cfgPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("setupWorkDirWithConfig: write .clinban: %v", err)
	}
	return root, ticketsDir, archiveDir
}

// TestNewNoInteractiveDefaultTypeUsedWhenNoFlagSet verifies that when
// default_type = "task" is in .clinban and --type is omitted, the command
// exits 0 and creates a ticket with type: task.
func TestNewNoInteractiveDefaultTypeUsedWhenNoFlagSet(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDirWithConfig(t, `default_type = "task"`)

	stdout, stderr, code := runNew(t, bin, root, "--title", "Test default type ticket")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	// Find the created ticket file.
	entries, err := os.ReadDir(ticketsDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var mdFiles []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			mdFiles = append(mdFiles, e.Name())
		}
	}
	if len(mdFiles) == 0 {
		t.Fatal("no ticket file created")
	}

	content, err := os.ReadFile(filepath.Join(ticketsDir, mdFiles[0]))
	if err != nil {
		t.Fatalf("reading ticket: %v", err)
	}
	if !strings.Contains(string(content), "type: task") {
		t.Errorf("expected 'type: task' in ticket; content:\n%s", string(content))
	}
}

// TestNewNoInteractiveNoDefaultTypeRequiresFlag verifies that when no
// default_type is set in .clinban and --type is omitted, the command exits 1
// with "required" on stderr.
func TestNewNoInteractiveNoDefaultTypeRequiresFlag(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	// Use a config without default_type.
	root, _, _ := setupWorkDirWithConfig(t, `# no default_type`)

	stdout, stderr, code := runNew(t, bin, root, "--title", "Should fail without type")

	if code != 1 {
		t.Errorf("exit code = %d, want 1; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "required") {
		t.Errorf("stderr = %q, want 'required'", stderr)
	}
}

// TestNewNoInteractiveInvalidDefaultTypeRequiresFlag verifies that when
// default_type is set to an invalid value in .clinban and --type is omitted,
// the command exits 1 (falls through to the "type required" error path).
func TestNewNoInteractiveInvalidDefaultTypeRequiresFlag(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDirWithConfig(t, `default_type = "notavalidtype"`)

	stdout, stderr, code := runNew(t, bin, root, "--title", "Should fail with invalid default type")

	if code != 1 {
		t.Errorf("exit code = %d, want 1; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "required") {
		t.Errorf("stderr = %q, want 'required'", stderr)
	}
}

// --- Interactive (T-17) tests ---

const (
	interactiveTestTitle = "Fix session expiry bug"
	interactiveTestType  = "bug"
	interactiveTestSlug  = "fix-session-expiry-bug"
	interactiveTestID    = "0001"
)

// makeEditorScript writes a shell script to dir that, when run as $EDITOR with
// a file path argument, replaces the title and type placeholder lines with the
// given values, then exits 0.
func makeEditorScript(t *testing.T, dir, title, ticketType string) string {
	t.Helper()
	// Use sed to replace the empty placeholder lines in-place.
	script := "#!/bin/sh\n" +
		"set -e\n" +
		// Replace 'title: ""' with the provided title.
		"sed -i 's|title: \"\"|title: \"" + title + "\"|' \"$1\"\n" +
		// Replace 'type: ""' with the provided type.
		"sed -i 's|type: \"\"|type: \"" + ticketType + "\"|' \"$1\"\n" +
		"exit 0\n"
	scriptPath := dir + "/fake-editor.sh"
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		t.Fatalf("makeEditorScript: %v", err)
	}
	return scriptPath
}

// makeDiscardEditorScript writes a script that does NOT modify the file
// (simulates the user opening and closing without any changes).
func makeDiscardEditorScript(t *testing.T, dir string) string {
	t.Helper()
	script := "#!/bin/sh\nexit 0\n"
	scriptPath := dir + "/discard-editor.sh"
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		t.Fatalf("makeDiscardEditorScript: %v", err)
	}
	return scriptPath
}

// makeLintErrorEditorScript writes a script that sets a valid title but leaves
// type empty (produces a lint error).
func makeLintErrorEditorScript(t *testing.T, dir, title string) string {
	t.Helper()
	script := "#!/bin/sh\n" +
		"set -e\n" +
		"sed -i 's|title: \"\"|title: \"" + title + "\"|' \"$1\"\n" +
		"exit 0\n"
	scriptPath := dir + "/lint-error-editor.sh"
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		t.Fatalf("makeLintErrorEditorScript: %v", err)
	}
	return scriptPath
}

// runNewInteractive executes "clinban new" (interactive) in workDir, with the
// given EDITOR environment variable and stdin input.
func runNewInteractive(t *testing.T, bin, workDir, editor, stdin string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(bin, "new")
	cmd.Dir = workDir
	cmd.Env = append(coverEnv(), "EDITOR="+editor)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	} else {
		cmd.Stdin = strings.NewReader("")
	}
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

// TestNewInteractiveHappyPath verifies that when $EDITOR fills in a valid title
// and type, the ticket file appears in TicketsDir with correct content.
func TestNewInteractiveHappyPath(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)
	scriptDir := t.TempDir()

	editor := makeEditorScript(t, scriptDir, interactiveTestTitle, interactiveTestType)

	stdout, stderr, code := runNewInteractive(t, bin, root, editor, "")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	wantFile := fmt.Sprintf("%s-%s.md", interactiveTestID, interactiveTestSlug)
	if !strings.Contains(stdout, "created:") {
		t.Errorf("stdout = %q, want 'created:' prefix", stdout)
	}
	if !strings.Contains(stdout, wantFile) {
		t.Errorf("stdout = %q, want to contain %q", stdout, wantFile)
	}

	ticketPath := filepath.Join(ticketsDir, wantFile)
	if _, err := os.Stat(ticketPath); os.IsNotExist(err) {
		t.Fatalf("ticket file %q not found in %q", wantFile, ticketsDir)
	}

	content, err := os.ReadFile(ticketPath)
	if err != nil {
		t.Fatalf("reading ticket: %v", err)
	}
	body := string(content)

	checks := []struct {
		desc    string
		wantStr string
	}{
		{"id field", fmt.Sprintf(`id: "%s"`, interactiveTestID)},
		{"status field", `status:`},
		{"type field", interactiveTestType},
		{"title field", interactiveTestTitle},
	}
	for _, c := range checks {
		if !strings.Contains(body, c.wantStr) {
			t.Errorf("%s: file does not contain %q\ncontent:\n%s", c.desc, c.wantStr, body)
		}
	}
}

// TestNewInteractiveDiscard verifies that when $EDITOR does not change the
// template, no ticket file is written and "Ticket discarded." is printed.
func TestNewInteractiveDiscard(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)
	scriptDir := t.TempDir()

	editor := makeDiscardEditorScript(t, scriptDir)

	stdout, stderr, code := runNewInteractive(t, bin, root, editor, "")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	if !strings.Contains(stdout+stderr, "discarded") {
		t.Errorf("expected 'discarded' in output; stdout=%q stderr=%q", stdout, stderr)
	}

	// No ticket files should exist in ticketsDir.
	entries, err := os.ReadDir(ticketsDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			t.Errorf("unexpected .md file found after discard: %q", e.Name())
		}
	}
}

// TestNewInteractiveLintErrorPromptsReopen verifies that when the editor
// produces lint errors, errors are listed and the user is prompted to re-open.
// The user declines (inputs "n"), so the file is still written and the command
// exits 0.
func TestNewInteractiveLintErrorPromptsReopen(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)
	scriptDir := t.TempDir()

	// Editor sets a title but leaves type as "" — will produce a lint error.
	editor := makeLintErrorEditorScript(t, scriptDir, interactiveTestTitle)

	// User declines re-open with "n".
	stdout, stderr, code := runNewInteractive(t, bin, root, editor, "n\n")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	// Lint errors should be mentioned in output.
	combined := stdout + stderr
	if !strings.Contains(combined, "type") && !strings.Contains(combined, "lint") && !strings.Contains(combined, "field") {
		t.Errorf("expected lint error output; stdout=%q stderr=%q", stdout, stderr)
	}

	// Re-open prompt should appear.
	if !strings.Contains(combined, "Re-open") && !strings.Contains(combined, "re-open") {
		t.Errorf("expected re-open prompt; stdout=%q stderr=%q", stdout, stderr)
	}

	// The ticket file must still exist in ticketsDir (written regardless of lint).
	entries, err := os.ReadDir(ticketsDir)
	if err != nil {
		t.Fatal(err)
	}
	var mdFiles []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			mdFiles = append(mdFiles, e.Name())
		}
	}
	if len(mdFiles) == 0 {
		t.Error("expected ticket file to exist after lint-error path (written regardless of lint)")
	}
}

// TestNewInteractiveStatusIsBacklog verifies that interactively created tickets
// start with status "backlog".
func TestNewInteractiveStatusIsBacklog(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)
	scriptDir := t.TempDir()

	editor := makeEditorScript(t, scriptDir, interactiveTestTitle, interactiveTestType)
	_, _, code := runNewInteractive(t, bin, root, editor, "")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	wantFile := fmt.Sprintf("%s-%s.md", interactiveTestID, interactiveTestSlug)
	content, err := os.ReadFile(filepath.Join(ticketsDir, wantFile))
	if err != nil {
		t.Fatalf("reading ticket: %v", err)
	}
	if !strings.Contains(string(content), "backlog") {
		t.Errorf("status is not 'backlog' in:\n%s", content)
	}
}

// TestNewInteractiveNoTmpFileLeft verifies that no .clinban-*.md temp file
// remains in TicketsDir after successful interactive creation.
func TestNewInteractiveNoTmpFileLeft(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)
	scriptDir := t.TempDir()

	editor := makeEditorScript(t, scriptDir, interactiveTestTitle, interactiveTestType)
	_, _, code := runNewInteractive(t, bin, root, editor, "")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	entries, err := os.ReadDir(ticketsDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".clinban-") {
			t.Errorf("temp file left behind: %q", e.Name())
		}
	}
}

// TestNewInteractiveNoDoubleCountID is a regression test for the bug where
// AllIDs() was called after os.Rename(), causing the new ticket's ID to appear
// twice in allIDsWithNew and triggering a false-positive ruleIDUnique lint
// error immediately after a successful creation.
//
// Setup: pre-seed ticket 0001, create ticket 0002 interactively.
// Assertion: no lint error fires (stderr must not contain "unique" or "not unique"),
// and the new file exists with exit code 0.
func TestNewInteractiveNoDoubleCountID(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)
	scriptDir := t.TempDir()

	// Pre-populate tickets dir with ticket 0001 so NextID returns 2.
	writeTicket(t, ticketsDir, "0001-existing-ticket.md", validTicketContent("0001"))

	// Editor fills in a valid title and type for the new ticket.
	editor := makeEditorScript(t, scriptDir, "Regression double count id", "task")

	stdout, stderr, code := runNewInteractive(t, bin, root, editor, "")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	// The bug manifested as a false-positive "not unique" lint error on stderr.
	if strings.Contains(stderr, "unique") || strings.Contains(stderr, "not unique") {
		t.Errorf("false-positive ID uniqueness lint error fired; stderr=%q", stderr)
	}

	// The new ticket file must be created successfully.
	wantFile := "0002-regression-double-count-id.md"
	if _, err := os.Stat(filepath.Join(ticketsDir, wantFile)); os.IsNotExist(err) {
		t.Errorf("expected ticket file %q not found in %q", wantFile, ticketsDir)
	}

	// stdout must contain the "created:" success message.
	if !strings.Contains(stdout, "created:") {
		t.Errorf("stdout = %q, want 'created:' message", stdout)
	}
}

// TestNewInteractiveIDAssignment verifies sequential ID assignment in
// interactive mode when existing tickets are present.
func TestNewInteractiveIDAssignment(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)
	scriptDir := t.TempDir()

	// Pre-populate with ticket 0001.
	writeTicket(t, ticketsDir, "0001-existing-ticket.md", validTicketContent("0001"))

	editor := makeEditorScript(t, scriptDir, interactiveTestTitle, interactiveTestType)
	stdout, stderr, code := runNewInteractive(t, bin, root, editor, "")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	if !strings.Contains(stdout, "0002") {
		t.Errorf("stdout = %q, want to contain '0002'", stdout)
	}

	wantFile := fmt.Sprintf("0002-%s.md", interactiveTestSlug)
	if _, err := os.Stat(filepath.Join(ticketsDir, wantFile)); os.IsNotExist(err) {
		t.Errorf("expected ticket file %q not found", wantFile)
	}
}

// Constants for body-args interactive tests.
const (
	bodyArgsTestTitle = "Fix session expiry bug"
	bodyArgsTestType  = "bug"
	bodyArgsTestBody  = "body text here"
	bodyArgsTestSlug  = "fix-session-expiry-bug"
	bodyArgsTestID    = "0001"
)

// runNewInteractiveWithArgs executes "clinban new [args...]" (interactive) in
// workDir, with the given EDITOR environment variable and stdin input.
func runNewInteractiveWithArgs(t *testing.T, bin, workDir, editorPath, stdin string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmdArgs := append([]string{"new"}, args...)
	cmd := exec.Command(bin, cmdArgs...)
	cmd.Dir = workDir
	cmd.Env = append(coverEnv(), "EDITOR="+editorPath)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	} else {
		cmd.Stdin = strings.NewReader("")
	}
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

// TestNewInteractiveWithBodyArgs verifies that positional args are pre-filled
// into the temp file as body text, survive the editor round-trip, and appear
// in the created ticket file.
func TestNewInteractiveWithBodyArgs(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)
	scriptDir := t.TempDir()

	// Editor sets title and type; leaves body untouched.
	editorScript := makeEditorScript(t, scriptDir, bodyArgsTestTitle, bodyArgsTestType)

	stdout, stderr, code := runNewInteractiveWithArgs(t, bin, root, editorScript, "", bodyArgsTestBody)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	wantFile := fmt.Sprintf("%s-%s.md", bodyArgsTestID, bodyArgsTestSlug)
	if !strings.Contains(stdout, "created:") {
		t.Errorf("stdout = %q, want 'created:' prefix", stdout)
	}
	if !strings.Contains(stdout, wantFile) {
		t.Errorf("stdout = %q, want to contain %q", stdout, wantFile)
	}

	ticketPath := filepath.Join(ticketsDir, wantFile)
	content, err := os.ReadFile(ticketPath)
	if err != nil {
		t.Fatalf("reading ticket file: %v", err)
	}

	if !strings.Contains(string(content), bodyArgsTestBody) {
		t.Errorf("ticket body does not contain %q\nfull content:\n%s", bodyArgsTestBody, string(content))
	}
}

// TestNewInteractiveBodyArgWithDashDash verifies that body text containing
// "--flag-like" words (e.g. "add a '--archived' flag") is treated as a
// positional argument, not parsed as an unknown flag. Without
// SetInterspersed(false) on the newCmd flags, Cobra would error on "--archived"
// and SilenceErrors would swallow it, producing a silent no-op.
func TestNewInteractiveBodyArgWithDashDash(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)
	scriptDir := t.TempDir()

	editorScript := makeEditorScript(t, scriptDir, bodyArgsTestTitle, bodyArgsTestType)

	// Pass body text that contains a --flag-like word; should not silently fail.
	stdout, stderr, code := runNewInteractiveWithArgs(t, bin, root, editorScript, "",
		"add", "a", "--archived", "flag", "to", "list")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "created:") {
		t.Errorf("stdout = %q, want 'created:' — body with --flag-like word should not silently fail", stdout)
	}

	// The created ticket should exist.
	entries, err := os.ReadDir(ticketsDir)
	if err != nil {
		t.Fatal(err)
	}
	var mdFiles []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			mdFiles = append(mdFiles, e.Name())
		}
	}
	if len(mdFiles) == 0 {
		t.Error("no ticket file created when body contains --flag-like word")
	}
}

// TestNewInteractiveNoArgsUnchanged verifies that the existing happy-path
// behaviour is preserved when no positional args are passed.
func TestNewInteractiveNoArgsUnchanged(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, ticketsDir, _ := setupWorkDir(t)
	scriptDir := t.TempDir()

	editorScript := makeEditorScript(t, scriptDir, interactiveTestTitle, interactiveTestType)

	stdout, stderr, code := runNewInteractive(t, bin, root, editorScript, "")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, stdout, stderr)
	}

	wantFile := fmt.Sprintf("%s-%s.md", interactiveTestID, interactiveTestSlug)
	if !strings.Contains(stdout, "created:") {
		t.Errorf("stdout = %q, want 'created:' prefix", stdout)
	}
	if !strings.Contains(stdout, wantFile) {
		t.Errorf("stdout = %q, want to contain %q", stdout, wantFile)
	}

	ticketPath := filepath.Join(ticketsDir, wantFile)
	if _, err := os.Stat(ticketPath); os.IsNotExist(err) {
		t.Fatalf("ticket file %q not found", wantFile)
	}

	content, err := os.ReadFile(ticketPath)
	if err != nil {
		t.Fatalf("reading ticket: %v", err)
	}
	body := string(content)

	checks := []struct {
		desc    string
		wantStr string
	}{
		{"id field", fmt.Sprintf(`id: "%s"`, interactiveTestID)},
		{"status field", `status:`},
		{"type field", interactiveTestType},
		{"title field", interactiveTestTitle},
	}
	for _, c := range checks {
		if !strings.Contains(body, c.wantStr) {
			t.Errorf("%s: file does not contain %q\ncontent:\n%s", c.desc, c.wantStr, body)
		}
	}
}
