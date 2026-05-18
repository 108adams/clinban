package editor

import (
	"fmt"
	"os"
	"os/exec"
)

// Open launches $EDITOR (fallback: "vi") with path as argument.
// Stdin, Stdout, and Stderr are inherited from the parent process.
// Returns error if the editor process exits non-zero.
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
