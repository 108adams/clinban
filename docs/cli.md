---
title: CLI Reference
kind: reference
scope: cli
summary: Documents Clinban commands, expected outputs, and exit-code conventions.
updated: 2026-05-22
links:
  - ticket-schema
  - configuration
  - validation
  - storage
---

# CLI Reference

Clinban commands operate on the configured active and archive ticket directories.

Normal output goes to stdout. User-facing errors go to stderr, except lint violations, which are normal validation output and go to stdout. Exit code `0` means success; exit code `1` means an error or validation failure.

If an unrecognised command is given (e.g. `clinban view`), Clinban prints an "unknown command" error to stderr, displays the root help to stdout, and exits with code `1`.

## `clinban init`

Initializes a Clinban project in the current directory by creating four artifacts:

- `tickets/` — active ticket directory
- `tickets/archive/` — archive directory
- `.clinban` — TOML configuration file
- `SCHEMA.md` — human/LLM-readable reference for the ticket format, field constraints, status transitions, and agent operations

Each created artifact is reported on stdout (`created: tickets/`, etc.). If any artifact already exists without `--force`, the command exits `1` and lists both the conflicting artifacts and the missing ones on stderr, so the user can see exactly what needs to be created.

Optional flags:

- `--tickets-dir` (default: `tickets`) — directory for active tickets
- `--archive-dir` (default: `<tickets-dir>/archive`) — directory for archived tickets
- `--force` — create only missing artifacts; fails with `already fully initialized` if all already exist

## `clinban new`

Creates a ticket interactively. Clinban renders a template with system fields, opens `$EDITOR` with fallback to `vi`, and writes the resulting ticket when the user provides a title.

`$EDITOR` may include arguments. For common GUI editors that return before the file is saved unless instructed to wait, Clinban adds the editor's wait flag automatically.

Optional positional arguments are joined with spaces. If the joined string contains `#`, the part before `#` (trimmed) pre-fills the frontmatter title and the part after `#` (trimmed) pre-fills the body. If there is no `#`, the full string goes to the body only (no title pre-fill).

```text
# Pre-fill body only (no # separator)
clinban new investigate the memory leak in the worker pool

# Pre-fill title and body
clinban new fix login timeout on staging \# users see error after 30 seconds
```

The editor opens with the relevant fields already present.

Set `split_raw_new=false` in `.clinban` to disable splitting; all args then go to the body regardless of `#`.

If lint errors remain after editing, including an empty title, Clinban prompts
to reopen the editor before creating the managed ticket. Declining the prompt
exits with code `1` and leaves no ticket file behind.

The ticket is written only after the edited frontmatter parses and passes lint.

## `clinban new --no-interactive`

Creates a ticket from flags:

```text
clinban new --no-interactive --title "Fix login" --type bug --tags auth,backend --body "Details"
```

Required flags:

- `--title`
- `--type` — required unless `default_type` is set to a valid type in `.clinban`

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

`$EDITOR` handling is the same as `clinban new`, including support for editor arguments and automatic wait flags for common GUI editors.

If parse or lint fails, the user is prompted to reopen the editor. Declining exits with code `1` and leaves the original ticket unchanged.

## `clinban move <id> <status>`

Transitions a ticket status through the state machine. Invalid transitions are rejected with a list of valid next statuses.

Moving a done archived ticket back to `backlog` writes the updated ticket into the active directory before removing the archived copy.

## `clinban push <id>`

Advances a ticket one step forward through the workflow without specifying the target status:

| Current status | After push |
|---|---|
| `backlog` | `in-progress` |
| `in-progress` | `done` |
| `blocked` | `in-progress` |
| `done` | _(no change — exits `0` with a message)_ |

Exits `0` in all non-error cases, including when the ticket is already `done`.

## `clinban archive [id]`

With an ID, archives one ticket. The ticket must be `done`.

Without an ID, lists all active `done` tickets and prompts for confirmation before archiving them.

## `clinban remove <id>`

Permanently deletes the ticket file from disk (active or archive directory).

Exit codes:

- `0` — file removed; prints `removed: <filename>` to stdout
- `1` — ticket not found; prints `ticket not found` to stderr
- `1` — multiple files share the ID (collision); lists all colliding filenames on stderr and suggests running `clinban lint`

## `clinban lint [id]`

Validates one ticket or all tickets. With no argument, active and archived tickets are checked.

Lint exits silently with code `0` when no errors are found.

## `clinban config [key=value]`

Views or sets configuration values stored in `.clinban`.

**No arguments** — lists all known configuration keys. For keys explicitly set in `.clinban`, shows the stored value. For keys not set, shows the built-in default with a note:

```
tickets_dir = tickets         (not set in .clinban, default: tickets)
archive_dir = tickets/archive (not set in .clinban, default: tickets/archive)
default_type =                (not set in .clinban, no default)
split_raw_new = true          (not set in .clinban, default: true)
```

When a key is set in `.clinban`:

```
tickets_dir = mydir
```

**One `key=value` argument** — sets the key in `.clinban`. Creates the file if absent.

Valid keys:

| Key | Valid values | Default |
|---|---|---|
| `tickets_dir` | Any non-empty path | `tickets` |
| `archive_dir` | Any non-empty path | `<tickets_dir>/archive` |
| `default_type` | `bug`, `task`, `feature`, `spike`, or empty string to unset | _(none)_ |
| `split_raw_new` | `true`, `false` | `true` |

Exit codes:

- `0` — success
- `1` — unknown key, invalid value, or missing `=` in argument

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
