package main_test

import (
	"os/exec"
	"strings"
	"testing"
)

// Constants for root command tests.
const (
	unknownCmd = "view"
)

// runRoot executes the clinban binary with the given args in workDir and
// returns stdout, stderr, and the exit code.
func runRoot(t *testing.T, bin, workDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
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

// TestUnknownCommand verifies that invoking clinban with an unknown command
// prints an error to stderr, prints help to stdout, and exits with code 1.
func TestUnknownCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		args               []string
		wantExitCode       int
		wantStderrContains string
		wantStdoutContains string
	}{
		{
			name:               "unknown command exits 1 with error and help",
			args:               []string{unknownCmd},
			wantExitCode:       1,
			wantStderrContains: "unknown command",
			wantStdoutContains: "Usage:",
		},
		{
			name:               "no args exits 0 and shows help",
			args:               []string{},
			wantExitCode:       0,
			wantStdoutContains: "Usage:",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bin := buildBinary(t)
			root, _, _ := setupWorkDir(t)

			stdout, stderr, code := runRoot(t, bin, root, tc.args...)

			if code != tc.wantExitCode {
				t.Errorf("exit code = %d, want %d; stdout=%q stderr=%q",
					code, tc.wantExitCode, stdout, stderr)
			}
			if tc.wantStderrContains != "" && !strings.Contains(stderr, tc.wantStderrContains) {
				t.Errorf("stderr = %q, want to contain %q", stderr, tc.wantStderrContains)
			}
			if tc.wantStdoutContains != "" && !strings.Contains(stdout, tc.wantStdoutContains) {
				t.Errorf("stdout = %q, want to contain %q", stdout, tc.wantStdoutContains)
			}
		})
	}
}
