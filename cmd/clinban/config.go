package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/108adams/clinban/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config [key=value]",
	Short: "View or set configuration values",
	Long: `View or set Clinban configuration values stored in .clinban.

With no arguments, all known configuration keys are listed. Keys that are
explicitly set in .clinban show their value. Keys that are not set show the
built-in default with a note.

With one argument of the form key=value, the key is set in .clinban. The file
is created if it does not exist. Exits 1 on unknown key or invalid value.

Known keys:
  tickets_dir  — path to the active tickets directory (default: tickets)
  archive_dir  — path to the archive directory (default: tickets/archive)
  default_type — default ticket type: bug, task, feature, or spike (no default)`,
	Args: cobra.RangeArgs(0, 1),
	RunE: runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
}

// runConfig implements the "clinban config" command.
func runConfig(_ *cobra.Command, args []string) error {
	root := findProjectRoot()

	if len(args) == 0 {
		return runConfigList(root)
	}
	return runConfigSet(root, args[0])
}

// runConfigList prints all known config keys with their values and notes.
func runConfigList(root string) error {
	entries, err := config.Entries(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return ExitError{Code: 1, Err: err}
	}

	for _, e := range entries {
		if e.IsSet {
			fmt.Fprintf(os.Stdout, "%s = %s\n", e.Key, e.Value)
		} else if e.Default == "" {
			fmt.Fprintf(os.Stdout, "%s = %s\t(not set in .clinban, no default)\n", e.Key, e.Value)
		} else {
			fmt.Fprintf(os.Stdout, "%s = %s\t(not set in .clinban, default: %s)\n", e.Key, e.Value, e.Default)
		}
	}
	return nil
}

// runConfigSet parses a "key=value" argument and sets the key in .clinban.
func runConfigSet(root, arg string) error {
	idx := strings.IndexByte(arg, '=')
	if idx < 0 {
		fmt.Fprintf(os.Stderr, "invalid argument %q: expected key=value\n", arg)
		return ExitError{Code: 1, Err: fmt.Errorf("invalid argument: expected key=value")}
	}

	key := arg[:idx]
	value := arg[idx+1:]

	if err := config.SetKey(root, key, value); err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return ExitError{Code: 1, Err: err}
	}
	return nil
}
