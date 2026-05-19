package editor

import (
	"fmt"
	"os"
	"os/exec"
)

// Open launches the configured editor for path and waits for it to exit.
//
// EDITOR is used when set; otherwise Open falls back to vi. The child process
// inherits stdin, stdout, and stderr so interactive editors behave normally.
// A non-zero editor exit status is returned as an error.
func Open(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor %q exited with error: %w", editor, err)
	}

	return nil
}
