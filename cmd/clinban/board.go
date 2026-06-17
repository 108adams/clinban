package main

import (
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/108adams/clinban/internal/tui"
)

// boardCmd launches the interactive two-pane board TUI. It is a thin entry
// point: the package-level store (st), initialised by rootCmd's
// PersistentPreRun, is handed to the tui model, which is the sole owner of all
// board behavior.
var boardCmd = &cobra.Command{
	Use:   "board",
	Short: "Open the interactive board TUI",
	Long: `Open the interactive two-pane board.

The left pane lists active tickets in board order (in-progress, blocked,
backlog, done); the right pane previews the selected ticket's raw source.

Keys: j/k or arrows select, ctrl+d/ctrl+u scroll the preview, e edits the
selected ticket in $EDITOR, > advances its status, r reloads, ? toggles the
help bar, and q/ctrl+c/esc quit.`,
	Args: cobra.NoArgs,
	RunE: runBoard,
}

func init() {
	rootCmd.AddCommand(boardCmd)
}

func runBoard(_ *cobra.Command, _ []string) error {
	_, err := tea.NewProgram(tui.New(st)).Run()
	return err
}
