package editor_test

import (
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
