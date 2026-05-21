package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/108adams/clinban/internal/config"
	"github.com/108adams/clinban/internal/store"
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

	// SilenceUsage prevents Cobra from printing usage on every error.
	// SilenceErrors prevents Cobra from printing returned errors — main owns that.
	SilenceUsage:  true,
	SilenceErrors: true,

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

// Execute runs the Clinban root command.
//
// It is separated from main so command tests can execute the CLI entry point
// without duplicating process setup. If the command returns an ExitError,
// Execute calls os.Exit with the carried code so that deferred cleanup in
// callers still runs. Any other error exits with code 1.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		var exitErr ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		fmt.Fprintf(os.Stderr, "clinban: %v\n\n", err)
		_ = rootCmd.Help()
		os.Exit(1)
	}
}

// findProjectRoot walks upward from the current directory looking for .clinban.
//
// If no config file is found in any ancestor, the current working directory is
// treated as the project root.
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
