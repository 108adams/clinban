---
title: Board TUI
kind: reference
scope: cli
summary: Documents the interactive two-pane board TUI launched by `clinban board`.
updated: 2026-06-16
links:
  - cli
  - architecture
  - storage
---

# Board TUI

`clinban board` opens an interactive two-pane terminal UI over the active
tickets. It is built on the Charm stack (Bubble Tea, Bubbles, Lip Gloss) and
lives in `internal/tui`. The TUI is a **pure consumer**: it reads tickets and
acts on them through the same store and workflow rules as the CLI, and never
becomes a second source of ticket truth.

## Layout

- **Left pane** — active tickets in board order (`in-progress`, `blocked`,
  `backlog`, `done`; numeric ID within each group), the same order as
  [`clinban list`](cli.md). Each row shows the ticket ID, title, status, and
  type.
- **Right pane** — preview of the selected ticket.
- **Bottom** — a status/error line and a help bar.

## Keymap

| Key | Action |
|-----|--------|
| `j` / `↓` | move selection down |
| `k` / `↑` | move selection up |
| `ctrl+d` / `ctrl+u` | scroll the preview down / up |
| `e` | edit the selected ticket in `$EDITOR` |
| `>` | advance the selected ticket to its next status |
| `r` | reload the board from disk |
| `?` | toggle the help bar (short ↔ full) |
| `q` / `ctrl+c` / `esc` | quit |

## Behavior

- Tickets are loaded through the store's active-ticket scan. If any file fails
  to load or parse, the board shows an **error state** rather than a partial
  list — it never renders half a board.
- The right pane shows the selected ticket's raw file bytes verbatim (never a
  re-rendered or re-marshaled view); `ctrl+d` / `ctrl+u` scroll it. Changing the
  selection re-loads the preview for the newly selected ticket.
- `>` advances the selected ticket to its next workflow status (`backlog` →
  `in-progress` → `done`; `blocked` → `in-progress`). The status is re-read
  fresh from disk, written through the store, and the board reloads — the
  cursor stays on the acted-on ticket even though it re-sorts into another
  group. A ticket already at a terminal status reports "no further status" and
  is not modified.
- `e` opens the selected ticket in `$EDITOR`. The board edits a same-directory
  scratch copy (never the live file directly), then re-reads, parses, and lints
  it; only a clean ticket is committed back through the store. Parse/IO errors
  and lint violations are surfaced in the status line (reported distinctly) and
  the original file is left untouched — there is no stdin reopen prompt under
  the TUI. The terminal is released for the editor and restored on return.
- `r` re-reads the board from disk; navigation clamps at the first and last
  ticket.
- Terminal resize is handled; the panes re-fit to the new size.

## Boundaries

This is the terminal-UI **foundation**. The first release:

- shows **active tickets only** (archived tickets are not listed);
- renders the preview as **raw Markdown source** — the exact file bytes, never
  re-rendered or re-marshaled;
- mutates only through the store — status advance (`>`) and edit (`e`) both
  re-read fresh and write via the same path as the CLI, never from the
  in-memory snapshot.

See [Architecture](architecture.md) for the package boundary and the decision
record behind the Charm stack.
