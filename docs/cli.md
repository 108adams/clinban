---
title: CLI Reference
kind: reference
scope: cli
summary: Documents Clinban commands, expected outputs, and exit-code conventions.
updated: 2026-05-19
links:
  - ticket-schema
  - configuration
  - validation
  - storage
---

# CLI Reference

Clinban commands operate on the configured active and archive ticket directories.

Normal output goes to stdout. User-facing errors go to stderr, except lint violations, which are normal validation output and go to stdout. Exit code `0` means success; exit code `1` means an error or validation failure.

## `clinban new`

Creates a ticket interactively. Clinban renders a template with system fields, opens `$EDITOR` with fallback to `vi`, and writes the resulting ticket when the user provides a title.

If the title remains empty, the ticket is discarded.

If lint errors remain after creation, the ticket may still be written and the user is prompted to reopen the editor.

## `clinban new --no-interactive`

Creates a ticket from flags:

```text
clinban new --no-interactive --title "Fix login" --type bug --tags auth,backend --body "Details"
```

Required flags:

- `--title`
- `--type`

Optional flags:

- `--body`
- `--tags`

Lint errors block the write.

## `clinban register <path>`

Adopts an externally authored ticket file. Clinban parses the file, overwrites system-owned fields, lints the result, writes the canonical ticket file, and removes the source file after a successful write.

## `clinban list`

Lists active tickets, sorted by:

1. `in-progress`
2. `blocked`
3. `backlog`
4. `done`

Within each status, tickets are sorted by numeric ID.

Filters:

- `--status <value>`
- `--type <value>`
- `--tag <value>`

Multiple filters combine with AND logic.

## `clinban show <id>`

Prints one ticket in a human-readable format. It does not modify files. Archived tickets are shown with an `[archived]` marker.

## `clinban edit <id>`

Opens a ticket in `$EDITOR`. Clinban edits a temporary copy and replaces the live ticket only after parse and lint both pass.

If parse or lint fails, the user is prompted to reopen the editor. Declining exits with code `1` and leaves the original ticket unchanged.

## `clinban move <id> <status>`

Transitions a ticket status through the state machine. Invalid transitions are rejected with a list of valid next statuses.

Moving a done archived ticket back to `backlog` writes the updated ticket into the active directory before removing the archived copy.

## `clinban archive [id]`

With an ID, archives one ticket. The ticket must be `done`.

Without an ID, lists all active `done` tickets and prompts for confirmation before archiving them.

## `clinban lint [id]`

Validates one ticket or all tickets. With no argument, active and archived tickets are checked.

Lint exits silently with code `0` when no errors are found.

## `clinban completion <shell>`

Clinban uses Cobra, so it can generate shell completion scripts.

Supported shells:

- `bash`
- `zsh`
- `fish`
- `powershell`

Examples:

```bash
clinban completion bash
clinban completion zsh
clinban completion fish
clinban completion powershell
```

Each shell-specific subcommand prints the completion script to stdout. Use the shell subcommand's help for installation instructions:

```bash
clinban completion zsh --help
```
