package main_test

import (
	"os/exec"
	"strings"
	"testing"
)

// runBoard executes "clinban board [args...]" in workDir and returns stdout,
// stderr, and the exit code.
func runBoard(t *testing.T, bin, workDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(bin, append([]string{"board"}, args...)...)
	cmd.Dir = workDir
	cmd.Env = coverEnv()
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

// TestBoardHelp verifies the board command is registered and that
// "clinban board --help" prints usage and exits 0 without launching the TUI
// (cobra short-circuits on --help, so no TTY is required).
func TestBoardHelp(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	stdout, stderr, exitCode := runBoard(t, bin, root, "--help")
	if exitCode != 0 {
		t.Fatalf("board --help exit = %d, want 0\nstderr: %s", exitCode, stderr)
	}
	out := stdout + stderr
	for _, want := range []string{"board", "interactive"} {
		if !strings.Contains(out, want) {
			t.Errorf("board --help output missing %q\ngot:\n%s", want, out)
		}
	}
}

// TestBoardListedInRootHelp verifies the command is discoverable from the root
// help listing.
func TestBoardListedInRootHelp(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	root, _, _ := setupWorkDir(t)

	cmd := exec.Command(bin, "--help")
	cmd.Dir = root
	cmd.Env = coverEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("root --help failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "board") {
		t.Errorf("root --help does not list 'board'\ngot:\n%s", out)
	}
}
