package editor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/108adams/clinban/internal/editor"
)

const testFilePath = "/tmp/test-ticket.md"

// TestCommand_Nano verifies Command with EDITOR=nano returns correct name, path last, stdio nil.
func TestCommand_Nano(t *testing.T) {
	t.Setenv("EDITOR", "nano")

	cmd, err := editor.Command(testFilePath)
	if err != nil {
		t.Fatalf("Command returned unexpected error: %v", err)
	}
	if cmd == nil {
		t.Fatal("Command returned nil cmd")
	}
	if filepath.Base(cmd.Path) != "nano" {
		t.Errorf("cmd.Path base = %q, want %q", filepath.Base(cmd.Path), "nano")
	}
	if len(cmd.Args) == 0 {
		t.Fatal("cmd.Args is empty")
	}
	if cmd.Args[len(cmd.Args)-1] != testFilePath {
		t.Errorf("last arg = %q, want %q", cmd.Args[len(cmd.Args)-1], testFilePath)
	}
	if cmd.Stdin != nil {
		t.Errorf("cmd.Stdin = %v, want nil", cmd.Stdin)
	}
	if cmd.Stdout != nil {
		t.Errorf("cmd.Stdout = %v, want nil", cmd.Stdout)
	}
	if cmd.Stderr != nil {
		t.Errorf("cmd.Stderr = %v, want nil", cmd.Stderr)
	}
}

// TestCommand_CodeWait verifies --wait is appended for VS Code and not duplicated.
func TestCommand_CodeWait(t *testing.T) {
	t.Run("wait appended", func(t *testing.T) {
		t.Setenv("EDITOR", "code")

		cmd, err := editor.Command(testFilePath)
		if err != nil {
			t.Fatalf("Command returned unexpected error: %v", err)
		}

		waitCount := 0
		for _, arg := range cmd.Args {
			if arg == "--wait" {
				waitCount++
			}
		}
		if waitCount != 1 {
			t.Errorf("--wait count = %d, want 1; args: %v", waitCount, cmd.Args)
		}
		if cmd.Args[len(cmd.Args)-1] != testFilePath {
			t.Errorf("last arg = %q, want path %q", cmd.Args[len(cmd.Args)-1], testFilePath)
		}
	})

	t.Run("wait not duplicated", func(t *testing.T) {
		t.Setenv("EDITOR", "code --wait")

		cmd, err := editor.Command(testFilePath)
		if err != nil {
			t.Fatalf("Command returned unexpected error: %v", err)
		}

		waitCount := 0
		for _, arg := range cmd.Args {
			if arg == "--wait" {
				waitCount++
			}
		}
		if waitCount != 1 {
			t.Errorf("--wait count = %d, want 1 (no duplication); args: %v", waitCount, cmd.Args)
		}
	})
}

// TestCommand_EmptyEditorFallback verifies empty EDITOR falls back to vi.
func TestCommand_EmptyEditorFallback(t *testing.T) {
	t.Setenv("EDITOR", "")
	// Use a temp dir with a fake vi so exec.LookPath succeeds.
	dir := t.TempDir()
	vi := filepath.Join(dir, "vi")
	if err := os.WriteFile(vi, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatalf("write vi stub: %v", err)
	}
	t.Setenv("PATH", dir)

	cmd, err := editor.Command(testFilePath)
	if err != nil {
		t.Fatalf("Command returned unexpected error: %v", err)
	}
	if filepath.Base(cmd.Path) != "vi" {
		t.Errorf("cmd.Path base = %q, want %q", filepath.Base(cmd.Path), "vi")
	}
}

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
