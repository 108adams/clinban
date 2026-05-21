package editor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/108adams/clinban/internal/editor"
)

func TestEditorSuccess(t *testing.T) {
	t.Setenv("EDITOR", "/bin/true")
	tmp := t.TempDir()
	if err := editor.Open(tmp + "/file.md"); err != nil {
		t.Errorf("Open returned unexpected error: %v", err)
	}
}

func TestEditorFailure(t *testing.T) {
	t.Setenv("EDITOR", "/bin/false")
	tmp := t.TempDir()
	err := editor.Open(tmp + "/file.md")
	if err == nil {
		t.Fatal("Open returned nil; expected non-nil error")
	}
	if !strings.Contains(err.Error(), "exit status") {
		t.Errorf("error %q does not contain 'exit status'", err.Error())
	}
}

func TestEditorCommandWithArgs(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "editor-script")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '%s' \"$1\" > \"$2\"\n"), 0o700); err != nil {
		t.Fatalf("write editor script: %v", err)
	}

	target := filepath.Join(dir, "ticket.md")
	t.Setenv("EDITOR", script+" marker")

	if err := editor.Open(target); err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read edited file: %v", err)
	}
	if string(got) != "marker" {
		t.Errorf("file content = %q, want %q", string(got), "marker")
	}
}

func TestEditorAddsWaitForCode(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "code")
	body := `#!/bin/sh
last=""
wait_seen=0
for arg in "$@"; do
	if [ "$arg" = "--wait" ]; then
		wait_seen=1
	fi
	last="$arg"
done
if [ "$wait_seen" -eq 1 ]; then
	printf 'waited' > "$last"
fi
`
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatalf("write code script: %v", err)
	}

	target := filepath.Join(dir, "ticket.md")
	t.Setenv("EDITOR", script)

	if err := editor.Open(target); err != nil {
		t.Fatalf("Open returned unexpected error: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read edited file: %v", err)
	}
	if string(got) != "waited" {
		t.Errorf("file content = %q, want %q", string(got), "waited")
	}
}

func TestEditorFallback(t *testing.T) {
	// EDITOR="" triggers the vi fallback; override PATH so vi is not found
	t.Setenv("EDITOR", "")
	t.Setenv("PATH", t.TempDir()) // empty dir — no executables
	err := editor.Open("somefile.md")
	if err == nil {
		t.Fatal("Open returned nil; expected error when vi not in PATH")
	}
	if !strings.Contains(err.Error(), "executable file not found") {
		t.Errorf("error %q does not contain 'executable file not found'", err.Error())
	}
}
