package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildEditBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "clinban-edit-test")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/clinban/")
	cmd.Dir = "/home/adam/src/go-trello"
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

func runEditCmd(t *testing.T, bin, dir string, env map[string]string, stdin string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	e := os.Environ()
	e = append(e, "HOME="+dir)
	for k, v := range env {
		e = append(e, k+"="+v)
	}
	cmd.Env = e
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		if ex, ok := err.(*exec.ExitError); ok {
			exitCode = ex.ExitCode()
		} else {
			exitCode = 1
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

func createTicketForEdit(t *testing.T, bin, dir string) {
	t.Helper()
	_, errStr, code := runEditCmd(t, bin, dir,nil, "", "new", "--no-interactive", "--title", "Test ticket for edit", "--type", "task")
	if code != 0 {
		t.Fatalf("setup: create ticket failed: %s", errStr)
	}
}

func TestEditUnknownID(t *testing.T) {
	t.Parallel()
	bin := buildEditBinary(t)
	dir := t.TempDir()

	_, stderr, code := runEditCmd(t, bin, dir,nil, "", "edit", "9999")
	if code == 0 {
		t.Fatal("expected exit 1 for unknown ID")
	}
	if !strings.Contains(stderr, "ticket not found") {
		t.Errorf("expected 'ticket not found' in stderr, got: %q", stderr)
	}
}

func TestEditHappyPath(t *testing.T) {
	t.Parallel()
	bin := buildEditBinary(t)
	dir := t.TempDir()
	createTicketForEdit(t, bin, dir)

	editorScript := filepath.Join(dir, "editor.sh")
	script := "#!/bin/sh\nsed -i 's/title: Test ticket for edit/title: Updated title/' \"$1\"\n"
	if err := os.WriteFile(editorScript, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	_, stderr, code := runEditCmd(t, bin, dir, map[string]string{"EDITOR": editorScript}, "", "edit", "0001")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}

	files, _ := filepath.Glob(filepath.Join(dir, "0001-*.md"))
	if len(files) == 0 {
		t.Fatal("ticket file not found after edit")
	}
	raw, _ := os.ReadFile(files[0])
	if !strings.Contains(string(raw), "Updated title") {
		t.Errorf("expected updated title in file, got:\n%s", raw)
	}
}

func TestEditUpdatesTimestamp(t *testing.T) {
	t.Parallel()
	bin := buildEditBinary(t)
	dir := t.TempDir()
	createTicketForEdit(t, bin, dir)

	// Capture original updated timestamp.
	files, _ := filepath.Glob(filepath.Join(dir, "0001-*.md"))
	raw, _ := os.ReadFile(files[0])
	original := string(raw)

	editorScript := filepath.Join(dir, "editor.sh")
	// Change the title so lint passes and updated is refreshed.
	script := "#!/bin/sh\nsed -i 's/title: Test ticket for edit/title: Edited title/' \"$1\"\n"
	if err := os.WriteFile(editorScript, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	_, _, code := runEditCmd(t, bin, dir,map[string]string{"EDITOR": editorScript}, "", "edit", "0001")
	if code != 0 {
		t.Fatal("expected exit 0")
	}

	after, _ := os.ReadFile(files[0])
	// The updated: field should have changed.
	if string(after) == original {
		t.Error("expected file content to change after edit (updated timestamp)")
	}
}

func TestEditLintErrorNoReopen(t *testing.T) {
	t.Parallel()
	bin := buildEditBinary(t)
	dir := t.TempDir()
	createTicketForEdit(t, bin, dir)

	// Editor sets type to an invalid value (lint failure).
	editorScript := filepath.Join(dir, "editor.sh")
	script := "#!/bin/sh\nsed -i 's/type: task/type: invalid/' \"$1\"\n"
	if err := os.WriteFile(editorScript, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	// Send "n" to decline reopen.
	_, stderr, code := runEditCmd(t, bin, dir,map[string]string{"EDITOR": editorScript}, "n\n", "edit", "0001")
	if code == 0 {
		t.Fatal("expected exit 1 on lint failure")
	}
	if !strings.Contains(stderr, "Re-open in editor?") {
		t.Errorf("expected re-open prompt in stderr, got: %q", stderr)
	}
}

func TestEditLintPassAfterReopen(t *testing.T) {
	t.Parallel()
	bin := buildEditBinary(t)
	dir := t.TempDir()
	createTicketForEdit(t, bin, dir)

	// First editor call breaks the type; second fixes it.
	callCount := filepath.Join(dir, "calls")
	editorScript := filepath.Join(dir, "editor.sh")
	script := `#!/bin/sh
if [ ! -f ` + callCount + ` ]; then
  touch ` + callCount + `
  sed -i 's/type: task/type: invalid/' "$1"
else
  sed -i 's/type: invalid/type: task/' "$1"
fi
`
	if err := os.WriteFile(editorScript, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	// Send "y" to reopen after lint error.
	_, stderr, code := runEditCmd(t, bin, dir,map[string]string{"EDITOR": editorScript}, "y\n", "edit", "0001")
	if code != 0 {
		t.Fatalf("expected exit 0 after successful reopen, got %d; stderr: %s", code, stderr)
	}
}
