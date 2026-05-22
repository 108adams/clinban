package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed schema.md
var schemaMD string

//go:embed skills/tickets/SKILL.md
var skillMD string

type initFlags struct {
	ticketsDir string
	archiveDir string
	force      bool
}

func newInitCmd() *cobra.Command {
	var flags initFlags
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a Clinban project in the current directory",
		Long: `Create the tickets directory, archive directory, and .clinban config file
in the current working directory.

Use --force to create only missing artifacts when some already exist.`,
		SilenceUsage: true,
		// Override root PersistentPreRun — init must not call findProjectRoot() or build the store.
		PersistentPreRun: func(cmd *cobra.Command, args []string) {},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(flags)
		},
	}
	cmd.Flags().StringVar(&flags.ticketsDir, "tickets-dir", "tickets", "Directory for active tickets")
	cmd.Flags().StringVar(&flags.archiveDir, "archive-dir", "", "Directory for archived tickets (default: <tickets-dir>/archive)")
	cmd.Flags().BoolVar(&flags.force, "force", false, "Create only missing artifacts; fail if all already exist")
	return cmd
}

func init() {
	rootCmd.AddCommand(newInitCmd())
}

func runInit(flags initFlags) error {
	// Step 1: derive archiveDir from ticketsDir if not set.
	if flags.archiveDir == "" {
		flags.archiveDir = filepath.Join(flags.ticketsDir, "archive")
	}

	// Step 2: get current working directory.
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("init: get working directory: %w", err)
	}

	// Step 3: resolve absolute paths.
	absTickets := flags.ticketsDir
	if !filepath.IsAbs(absTickets) {
		absTickets = filepath.Join(cwd, flags.ticketsDir)
	}
	absArchive := flags.archiveDir
	if !filepath.IsAbs(absArchive) {
		absArchive = filepath.Join(cwd, flags.archiveDir)
	}
	absConfig := filepath.Join(cwd, ".clinban")

	// Step 4: pre-flight — stat all five and record existence.
	absSchema := filepath.Join(cwd, "SCHEMA.md")
	absSkillFile := filepath.Join(cwd, ".claude", "skills", "tickets", "SKILL.md")

	_, errTickets := os.Stat(absTickets)
	ticketsExists := errTickets == nil
	_, errArchive := os.Stat(absArchive)
	archiveExists := errArchive == nil
	_, errConfig := os.Stat(absConfig)
	configExists := errConfig == nil
	_, errSchema := os.Stat(absSchema)
	schemaExists := errSchema == nil
	_, errSkill := os.Stat(absSkillFile)
	skillFileExists := errSkill == nil

	// Step 5: without --force, fail if any artifact exists.
	if !flags.force {
		if ticketsExists || archiveExists || configExists || schemaExists || skillFileExists {
			if ticketsExists {
				fmt.Fprintln(os.Stderr, "already exists: tickets/")
			}
			if archiveExists {
				fmt.Fprintln(os.Stderr, "already exists: tickets/archive/")
			}
			if configExists {
				fmt.Fprintln(os.Stderr, "already exists: .clinban")
			}
			if schemaExists {
				fmt.Fprintln(os.Stderr, "already exists: SCHEMA.md")
			}
			if skillFileExists {
				fmt.Fprintln(os.Stderr, "already exists: .claude/skills/tickets/SKILL.md")
			}
			if !ticketsExists {
				fmt.Fprintln(os.Stderr, "missing: tickets/")
			}
			if !archiveExists {
				fmt.Fprintln(os.Stderr, "missing: tickets/archive/")
			}
			if !configExists {
				fmt.Fprintln(os.Stderr, "missing: .clinban")
			}
			if !schemaExists {
				fmt.Fprintln(os.Stderr, "missing: SCHEMA.md")
			}
			if !skillFileExists {
				fmt.Fprintln(os.Stderr, "missing: .claude/skills/tickets/SKILL.md")
			}
			fmt.Fprintln(os.Stderr, "re-run with --force to create missing items")
			return fmt.Errorf("init: project already partially or fully initialized")
		}
	}

	// Step 6: with --force, fail if all artifacts exist.
	if flags.force && ticketsExists && archiveExists && configExists && schemaExists && skillFileExists {
		fmt.Fprintln(os.Stderr, "already fully initialized")
		return fmt.Errorf("init: already fully initialized")
	}

	// Step 7: create missing artifacts in order.
	if !ticketsExists {
		if err := os.Mkdir(absTickets, 0o755); err != nil {
			return fmt.Errorf("init: create tickets dir: %w", err)
		}
		fmt.Println("created: tickets/")
	}

	if !archiveExists {
		if err := os.Mkdir(absArchive, 0o755); err != nil {
			return fmt.Errorf("init: create archive dir: %w", err)
		}
		fmt.Println("created: tickets/archive/")
	}

	if !configExists {
		content := fmt.Sprintf("tickets_dir = %q\narchive_dir = %q\n", flags.ticketsDir, flags.archiveDir)
		if err := os.WriteFile(absConfig, []byte(content), 0o600); err != nil {
			return fmt.Errorf("init: write config: %w", err)
		}
		fmt.Println("created: .clinban")
	}

	if !schemaExists {
		if err := os.WriteFile(absSchema, []byte(schemaMD), 0o644); err != nil {
			return fmt.Errorf("init: write schema: %w", err)
		}
		fmt.Println("created: SCHEMA.md")
	}

	if !skillFileExists {
		if err := os.MkdirAll(filepath.Dir(absSkillFile), 0o755); err != nil {
			return fmt.Errorf("init: create skill dir: %w", err)
		}
		if err := os.WriteFile(absSkillFile, []byte(skillMD), 0o644); err != nil {
			return fmt.Errorf("init: write skill file: %w", err)
		}
		fmt.Println("created: .claude/skills/tickets/SKILL.md")
	}

	return nil
}
