---
title: Documentation Log
kind: log
scope: docs
summary: Records chronological maintenance activity for the Clinban documentation wiki.
updated: 2026-06-16
links:
  - index
  - schema
  - product
  - architecture
---

# Documentation Log

## [2026-06-14] feature | version flag and release workflow (ticket 0020)

- Source: `tickets/0020-version-command.md`, `.github/workflows/release.yml`
- Updated: `docs/cli.md`, `docs/log.md`
- Notes: Cobra `Version` field wired to `main.version` (default `"dev"`, injected via `-ldflags` on release builds). Added `.github/workflows/release.yml`: triggers on `v*` tag push, cross-compiles for linux/amd64, darwin/arm64, windows/amd64 on ubuntu-latest with `CGO_ENABLED=0`, uploads per-binary `.sha256` checksums and release binaries via `softprops/action-gh-release@v2` with auto-generated release notes. Documented `--version` flag in `docs/cli.md` with output examples for release and local builds.

## [2026-06-10] feature | clinban resolve command

- Source: `tickets/0022-add-conflict-resolution-command.md`, `cmd/clinban/resolve.go`, `internal/store/scan.go`, `internal/store/move.go`
- Updated: `docs/cli.md`, `docs/storage.md`, `docs/validation.md`, `cmd/clinban/schema.md`, `docs/log.md`
- Notes: Documented duplicate ticket ID repair with `clinban resolve`, including oldest-created retention, filename-only renumbering, archive preservation, and lint's detection role.

## [2026-05-22] feature | clinban init emits .claude/skills/tickets/SKILL.md (TASK-001)

- Source: `cmd/clinban/init.go`, `cmd/clinban/skills/tickets/SKILL.md`, `cmd/clinban/init_test.go`
- Updated: `docs/cli.md`, `docs/log.md`
- Notes: `clinban init` now creates a fifth artifact, `.claude/skills/tickets/SKILL.md`, at `.claude/skills/tickets/` relative to the project root. The file is the LLM agent skill for ticket lifecycle operations, embedded from `cmd/clinban/skills/tickets/SKILL.md` (copied verbatim from `.claude/skills/tickets/SKILL.md`). Pre-flight, `--force`, fully-initialized guard, and reporting strings all include the new artifact consistently with the existing four.

## [2026-05-22] docs | Document # title/body split feature for clinban new

- Source: `cmd/clinban/new.go`, `internal/config/config.go`
- Updated: `cmd/clinban/new.go` (Long string), `docs/cli.md`, `docs/configuration.md`, `docs/log.md`
- Notes: Documented the `#` title/body splitting behaviour introduced in TASK-003–005. `clinban new` now pre-fills the frontmatter title from the text before `#` and the body from the text after `#` when positional args are given. Updated `clinban new` Long help text to describe splitting, show a `\#`-escaped example, and note the `split_raw_new=false` opt-out. Added `split_raw_new` to the valid-keys table and config output example in `docs/cli.md`, and added a row for `split_raw_new` to the Fields table in `docs/configuration.md`.

## [2026-05-21] refactor | Remove id: from frontmatter; ID derived from filename

- Source: `internal/ticket/ticket.go`, `internal/store/write.go`, `internal/lint/rules.go`, `internal/template/new.md`, `cmd/clinban/new.go`
- Updated: `cmd/clinban/schema.md`, `docs/ticket-schema.md`, `docs/log.md`
- Notes: Removed `id:` field from YAML frontmatter. The ticket ID is now derived exclusively from the filename's 4-digit prefix (e.g. `0042` in `0042-slug.md`). `ticket.Parse` no longer reads `id:` from frontmatter; `store.ReadTicket` injects the ID from the filename after parsing. `ticket.Marshal` no longer emits an `id:` line. The `ruleIDMatchesFilename` lint rule and its `leadingDigits` helper were removed as dead code — since the ID is always set from the filename, the check can never fire. Updated both schema reference documents to omit `id:` from example frontmatter blocks, field tables, and ownership descriptions; added a "System-Derived Fields" section to `docs/ticket-schema.md` explaining the filename-derived ID.

## [2026-05-21] update | Document /tickets skill as product deliverable

- Source: `.claude/skills/tickets/SKILL.md`, conversation
- Updated: `docs/product.md`, `README.md`
- Notes: Added "LLM Interface" section to product.md and a bullet to README; establishes the skill as a co-deliverable with the CLI.

## [2026-05-19] scaffold | Wiki core

- Source: `wiki.md`, user discussion
- Updated: `docs/schema.md`, `docs/index.md`, `docs/log.md`
- Notes: Created the lightweight documentation wiki foundation and frontmatter conventions.

## [2026-05-19] ingest | Product overview

- Source: `pipeline/00_vision.md`
- Updated: `docs/product.md`, `docs/index.md`, `docs/log.md`, `docs/documentation.md`
- Notes: Distilled product-level vision into current wiki form and retired the imported source file.

## [2026-05-19] ingest | Pipeline reference migration

- Source: `pipeline/01_requirements.md`, `pipeline/02_architecture.md`, `pipeline/03_design.md`, `pipeline/04_tasks.md`, `pipeline/05_review.md`
- Updated: `docs/cli.md`, `docs/ticket-schema.md`, `docs/configuration.md`, `docs/validation.md`, `docs/storage.md`, `docs/security.md`, `docs/architecture.md`, `docs/development.md`, `docs/adr/0001-cli-framework.md`, `docs/adr/0002-package-decomposition.md`, `docs/adr/0003-atomic-file-writes.md`, `docs/index.md`, `docs/log.md`
- Notes: Distilled remaining pipeline knowledge into current wiki pages and retired the imported pipeline sources.

## [2026-05-19] lint | Wiki health check

- Source: full wiki scan
- Updated: `docs/index.md`, `docs/schema.md`
- Notes: Fixed orphan `development.md` (missing from index); expanded scope examples in schema to match pages in use.

## [2026-05-19] update | clinban init command and default directory layout

- Source: implementation (`cmd/clinban/init.go`, `internal/config/config.go`)
- Updated: `docs/cli.md`, `docs/configuration.md`, `docs/log.md`
- Notes: Documented `clinban init`; updated default directory layout from project root to `tickets/` and `tickets/archive/`.

## [2026-05-19] update | Shell completion documentation

- Source: Cobra-generated `clinban completion --help`
- Updated: `docs/cli.md`, `docs/log.md`
- Notes: Documented shell completion generation for bash, zsh, fish, and powershell.

## [2026-05-21] feature | clinban init emits SCHEMA.md (ticket 0001)

- Source: `cmd/clinban/schema.md`, `cmd/clinban/init.go`, `cmd/clinban/init_test.go`
- Updated: `docs/cli.md`, `docs/log.md`
- Notes: `clinban init` now creates a fourth artifact, `SCHEMA.md`, at the project root. The file is a static Markdown schema reference for LLM agents and human contributors; it covers ticket format, field constraints, status transitions, and step-by-step agent operations.

## [2026-05-21] update | clinban push, new body args, default_type, init improvements

- Source: commits `9aa4134`–`c41d3fb` (tickets 0001–0003, 0005–0006, 0008)
- Updated: `docs/cli.md`, `docs/configuration.md`, `docs/log.md`, `CHANGELOG`
- Notes: Documented `clinban push`, body-arg behavior for `clinban new`, `default_type` config field, SCHEMA.md init artifact, partial-init missing-items output, and inline type hint in the ticket template.

## [2026-05-21] update | End-to-end testing strategy

- Source: maintainer discussion
- Updated: `docs/development.md`, `docs/log.md`
- Notes: Added E2E testing guidance for black-box CLI scenarios, fake editors, PTY use, subprocess coverage, and when to consider testscript.

## [2026-05-21] fix | Unknown command shows help (ticket 0007)

- Source: `cmd/clinban/root.go`, `cmd/clinban/root_test.go`
- Updated: `docs/cli.md`, `docs/log.md`
- Notes: `clinban <unknown>` now prints an "unknown command" error to stderr, displays root help to stdout, and exits 1 instead of silently failing.

## [2026-05-21] fix | Interactive new waits for GUI editors

- Source: `internal/editor/editor.go`, `internal/editor/editor_test.go`, `cmd/clinban/new_test.go`
- Updated: `docs/cli.md`, `docs/log.md`
- Notes: Documented editor command arguments and automatic wait flags for common GUI editors so interactive `new` and `edit` do not read unchanged temp files before save.

## [2026-05-21] feature | clinban remove command

- Source: `cmd/clinban/remove.go`, `cmd/clinban/remove_test.go`, `internal/store/scan.go`, `internal/store/store.go`, `internal/store/store_test.go`
- Updated: `docs/cli.md`, `docs/log.md`
- Notes: Added `clinban remove <id>` which deletes a ticket file from disk. Exits 1 with "ticket not found" when the ID doesn't exist. Exits 1 with a list of colliding filenames and a lint suggestion when multiple files share the same ID. Added `FindAllByID` to the store for collision detection and `Remove` as a thin `os.Remove` wrapper.

## [2026-05-21] fix | Interactive new validates before writing

- Source: `cmd/clinban/new.go`, `cmd/clinban/new_test.go`
- Updated: `docs/cli.md`, `docs/log.md`
- Notes: Empty titles and other lint failures now trigger the re-open prompt before a managed ticket file is created; declining the prompt exits 1 and leaves no invalid ticket behind.

## [2026-05-21] feature | clinban config command

- Source: `internal/config/config.go`, `cmd/clinban/config.go`, `cmd/clinban/config_test.go`
- Updated: `docs/cli.md`, `docs/configuration.md`, `docs/log.md`
- Notes: Added `clinban config` subcommand. No-args mode lists all three known keys with values, defaults, and set/not-set notes. Single `key=value` arg sets the key in `.clinban`, creating the file if absent, with validation for unknown keys, invalid `default_type` values, and empty path values.

## [2026-05-21] update | Schema cleanup — title first, states comment in template (ticket 0016)

- Source: `internal/ticket/ticket.go`, `internal/template/new.md`
- Updated: `docs/ticket-schema.md`, `cmd/clinban/schema.md`, `docs/log.md`
- Notes: Reordered `frontmatter` struct so `title` serialises as the first YAML field. Added `# states: backlog, in-progress, blocked, done` comment below the `status` field in the new-ticket template, mirroring the existing types hint. Updated example frontmatter blocks in both schema reference docs to reflect the new field order.

## [2026-05-22] fix | title field now double-quoted in new ticket frontmatter

- Source: ticket 0018
- Changed: `internal/template/template.go` — `yamlstr` func now uses `yaml.Node` with `DoubleQuotedStyle`
- Notes: Plain-string titles were rendered unquoted (e.g. `title: my title`); now consistently `title: "my title"` like all other frontmatter fields.

## [2026-05-20] update | GitHub migration — module path and CI

- Source: migration commit `6a0cd17`
- Updated: `docs/development.md`, `docs/log.md`
- Notes: Added CI section documenting GitHub Actions workflow; no other wiki pages had stale GitLab or old module path references.

## [2026-06-16] feature | clinban board TUI scaffold (ticket 0021, T4)

- Source: `internal/tui/*`, `cmd/clinban/board.go`
- Updated: `docs/clinban-board.md` (new), `docs/cli.md`, `docs/index.md`, `docs/log.md`
- Notes: Added the interactive two-pane board TUI on the Charm stack (Bubble Tea/Bubbles/Lip Gloss v2, `charm.land/...` import paths). First slice is the scaffold: active-ticket list in board order, raw-source preview placeholder, non-mutating keymap (`j`/`k`, `r` reload, `?` help, quit), resize handling, and a whole-board error state. Editing, status advance, and preview scrolling land in T5–T7 and will be documented as they arrive. `cmd/clinban/schema.md` checked — unchanged (board adds no ticket-frontmatter or generated-SCHEMA guidance).

## [2026-06-16] feature | board preview pane (ticket 0021, T5)

- Source: `internal/tui/commands.go`, `internal/tui/messages.go`, `internal/tui/model.go`, `internal/tui/keys.go`
- Updated: `docs/clinban-board.md`, `docs/log.md`
- Notes: The right pane now previews the selected ticket's raw file bytes (`os.ReadFile` on `Record.Path`, never a re-marshaled Ticket — ADR-4). Selection changes re-load the preview; `ctrl+d`/`ctrl+u` scroll it. Added a test guarding that non-canonical frontmatter is shown verbatim, not normalized.

## [2026-06-16] feature | board status advance (ticket 0021, T6)

- Source: `internal/tui/commands.go`, `internal/tui/messages.go`, `internal/tui/model.go`, `internal/tui/keys.go`
- Updated: `docs/clinban-board.md`, `docs/log.md`
- Notes: `>` advances the selected ticket to its next status. The status is re-read fresh from disk (`FindByID`+`ReadTicket`), advanced via `fsm.NextStatus`, and written via `store.WriteTicket` — never from the in-memory snapshot, mirroring `clinban move`. A terminal ticket reports "no further status" (no write); errors leave the file unchanged. After a successful advance the board reloads and the cursor stays on the acted-on ticket by ticket ID, even though it re-sorts into another group (shared `withReload`/`applyPendingSelection` helpers, now used by every reload).
