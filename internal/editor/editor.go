package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Command builds the editor *exec.Cmd for path with Stdin, Stdout, and Stderr
// intentionally unset. Callers wire stdio as appropriate: Open sets os.Std*
// for blocking terminal use; tea.ExecProcess wires the tty for TUI use.
//
// EDITOR is used when set; otherwise Command falls back to vi. GUI editors
// (code, cursor, zed, etc.) receive --wait automatically unless already present.
func Command(path string) (*exec.Cmd, error) {
	name, args, err := command(path)
	if err != nil {
		return nil, err
	}
	return exec.Command(name, args...), nil
}

// Open launches the configured editor for path and waits for it to exit.
//
// EDITOR is used when set; otherwise Open falls back to vi. The child process
// inherits stdin, stdout, and stderr so interactive editors behave normally.
// A non-zero editor exit status is returned as an error.
func Open(path string) error {
	cmd, err := Command(path)
	if err != nil {
		return err
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	name := filepath.Base(cmd.Path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor %q exited with error: %w", name, err)
	}

	return nil
}

func command(path string) (string, []string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return "", nil, fmt.Errorf("editor command is empty")
	}

	name := parts[0]
	args := append([]string{}, parts[1:]...)
	if needsWaitFlag(name, args) {
		args = append(args, "--wait")
	}
	args = append(args, path)
	return name, args, nil
}

func needsWaitFlag(name string, args []string) bool {
	switch filepath.Base(name) {
	case "code", "code-insiders", "codium", "cursor", "zed", "subl", "sublime_text", "mate", "gedit":
		for _, arg := range args {
			if arg == "--wait" || arg == "-w" {
				return false
			}
		}
		return true
	default:
		return false
	}
}
