package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"clinban/internal/config"
	"clinban/internal/store"
)

// st is the package-level Store used by all subcommands. It is initialised by
// rootCmd's PersistentPreRun before any subcommand runs.
var st *store.Store

// cfg is the package-level Config loaded from the project root. Subcommands
// may read it directly when they need path information.
var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:   "clinban",
	Short: "Clinban — a kanban board backed by markdown files",
	Long: `Clinban manages kanban tickets as markdown files in your project directory.

Each ticket is a markdown file with YAML frontmatter that records its ID, status,
type, title, tags, and timestamps. Tickets live in a configured directory and are
archived when done.`,

	// SilenceUsage prevents Cobra from printing full usage on every error.
	SilenceUsage: true,

	// PersistentPreRun runs before every subcommand. It finds the project root,
	// loads configuration, and initialises the package-level store.
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		root := findProjectRoot()

		var err error
		cfg, err = config.Load(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "clinban: %v\n", err)
			os.Exit(1)
		}

		st = store.New(cfg)
	},
}

// Execute runs the root command. main calls this and exits 1 on error.
func Execute() error {
	return rootCmd.Execute()
}

// findProjectRoot walks up from the current working directory looking for a
// .clinban config file. Returns the directory that contains .clinban, or the
// current working directory if no .clinban is found in any ancestor.
func findProjectRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		// Cannot determine cwd — fall back to "." resolved by the OS.
		return "."
	}

	dir := cwd
	for {
		candidate := filepath.Join(dir, ".clinban")
		if _, err := os.Stat(candidate); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the filesystem root without finding .clinban.
			break
		}
		dir = parent
	}

	return cwd
}
