# Tasks: Terminal UI Foundation (ticket 0021)

_Created: 2026-06-14_
_Input: `pipeline/0021/03_design.md` + `pipeline/0021/02_architecture.md`_

Spike for the phase-two TUI. 8 tasks, each ≤1 day, TDD. Run all Go commands with
`GOCACHE=/tmp/clinban-gocache`; gate each task on `go test ./... && go vet ./...` + `gofmt -w`.
One commit per task. T1/T2/T3 are independent refactors and may land in any order.
Each behavior-changing task lands its own docs in the same commit (relevant `docs/` page,
`docs/index.md` when navigation changes, `docs/log.md`, and a `cmd/clinban/schema.md` relevance
decision), per AGENTS.md "Documentation Rules"/"Definition of Done" — docs are **not** deferred to T8.
T8 is a final verification + reconciliation pass, not the first time docs are written.

## Dependency graph

```
T1 ─┐
T2 ─┼─► T4 ─► T5 ─┐
T3 ─┘     └─► T6 ─┼─► T8
          └─► T7 ─┘     (T7 also needs T1, T3)
```

| # | Deliverable | Depends on |
|---|-------------|-----------|
| T1 | `editor.Command` extraction | — |
| T2 | `internal/board` ordering | — |
| T3 | `lint.ValidateForCommit` extraction | — |
| T4 | TUI scaffold + deps + static board + non-mutating keymap (`r`/`?`) + docs | T2 |
| T5 | Preview pane | T4 |
| T6 | Status advance | T4 |
| T7 | Editor handoff + commit gate | T1, T3, T4 |
| T8 | Verification + docs reconciliation | T4–T7 |

---

## T1 — Extract `editor.Command`

**Deliverable:** `editor.Command(path string) (*exec.Cmd, error)` exported, returning a cmd with
`Stdin/Stdout/Stderr` **unset**; `editor.Open` re-implemented on top (sets `os.Std*`, then `Run`). The
unexported `command()` resolver is reused, not duplicated.

**Files:** `internal/editor/editor.go` (+ `editor_test.go`).

**Tests (write first, confirm FAIL):**
- `Command` with `EDITOR=nano` → name `nano`, args end with `path`, stdio fields nil.
- `Command` with `EDITOR=code` → `--wait` appended (and not duplicated when already present).
- `Command` with empty `EDITOR` → falls back to `vi`.
- Existing editor tests still pass (Open behavior unchanged).

**Done:** `go test ./internal/editor` green; `Open` produces identical behavior; no new deps.

---

## T2 — Extract `internal/board` ordering

**Deliverable:** new `internal/board` package (depends on `internal/ticket` **only**) with `StatusOrder`
(map) and `Less(a, b *ticket.Ticket) bool`. Refactor `cmd/clinban/list.go` so its `sortRecords`
delegates to `board.Less` via `sort.SliceStable`. `applyFilters` stays in `list.go`.

**Files:** `internal/board/board.go`, `internal/board/doc.go`, `internal/board/board_test.go`;
edit `cmd/clinban/list.go`.

**Tests (write first):**
- `board.Less`: mixed statuses + IDs → in-progress→blocked→backlog→done, ID ascending within group.
- Existing `cmd/clinban` list tests still pass (output order unchanged).

**Done:** `go test ./...` green; `clinban list` output byte-identical to before; no import cycle.

---

## T3 — Extract `lint.ValidateForCommit`

**Deliverable:** `lint.ValidateForCommit(raw []byte, id, filename string, allIDs []string)
(*ticket.Ticket, []LintError, error)` — parse → set ID → lint. Refactor `cmd/clinban/edit.go` and
`cmd/clinban/register.go` to call it for their parse+lint kernel. Edit's reopen loop + atomic write
and register's timestamps + path-containment stay unchanged.

**Files:** `internal/lint/lint.go` (or new `internal/lint/commit.go`) + `lint_test.go`; edit
`cmd/clinban/edit.go`, `cmd/clinban/register.go`.

**Tests (write first):**
- parse failure → `(nil, nil, err)`.
- lint violation (e.g. invalid status) → `(t, errs, nil)`.
- clean ticket → `(t, emptyErrs, nil)`.
- Existing edit + register tests still pass.

**Done:** `go test ./...` green; edit/register behavior unchanged.

---

## T4 — TUI scaffold + Charm deps + static two-pane board

**Deliverable:** `internal/tui` package (`model.go`, `keys.go`, `commands.go`, `messages.go`,
`doc.go`) and `cmd/clinban/board.go` (`clinban board`, `NoArgs`, self-registers). Renders a two-pane
Lip Gloss layout (left: bubbles `list` fed records sorted via `sort.SliceStable`+`board.Less`; right: empty viewport placeholder),
loads active tickets via `loadTickets(st)` → `ticketsLoadedMsg`, handles `WindowSizeMsg`. Scaffold
owns the **complete non-mutating keymap** so implementation matches the documented design (§"Full keymap"):
navigation (`j/↓` `k/↑`), `r` reload (re-issues `loadTickets`), `?` help toggle (bubbles `help.Model`
ShortHelp↔FullHelp, wired via `keys.go`), and quit (`q`/`ctrl+c`/`esc`). The mutating keys
(`ctrl+d/ctrl+u`, `e`, `>`) land in T5–T7. Add `bubbletea/v2`, `bubbles/v2`, `lipgloss/v2` to go.mod
(confirm canonical v2 import paths first).

**Files:** `internal/tui/*`, `cmd/clinban/board.go`, `go.mod`/`go.sum`; `docs/clinban-board.md` (new
page), `docs/cli.md`, `docs/index.md`, `docs/log.md`. Use `/librarian` for the docs page.

**Tests (write first, Msg injection — no TTY):**
- `ticketsLoadedMsg{records}` → list populated in board order.
- `ticketsLoadedMsg{err}` → `Model.err` set, list empty (no half-board).
- nav down/up moves selection, clamps at bounds.
- `r` key issues a `loadTickets` reload cmd.
- `?` key toggles help; `View` reflects the expanded/collapsed help state.
- `WindowSizeMsg` stores dims; `View` does not panic at small sizes.
- quit keys return `tea.Quit`.
- `clinban board` is registered on `rootCmd` and `clinban board --help` succeeds without launching the
  TUI (cobra short-circuits help — no TTY; follows the `cmd/clinban` binary integration-test pattern).

**Docs:** new `docs/clinban-board.md` (purpose, the full keymap incl. `r`/`?`, two-pane behavior,
boundaries); add `clinban board` to `docs/cli.md`; add the new page to `docs/index.md`; append
`docs/log.md`. Schema relevance: `cmd/clinban/schema.md` **unchanged** — board adds no frontmatter or
generated-`SCHEMA.md` guidance (record the check).

**Done:** `clinban board` launches, lists active tickets, navigates, reloads (`r`), toggles help (`?`),
quits cleanly (manual); model tests green; `go test ./... && go vet ./...` green; docs above landed in
this commit.

---

## T5 — Preview pane (original file bytes)

**Deliverable:** `loadPreview(path)` cmd (`os.ReadFile(Record.Path)`) → `previewLoadedMsg`; viewport
renders the raw bytes; selection change fires `loadPreview` for the newly selected record; `ctrl+d`/`ctrl+u`
scroll the preview.

**Files:** `internal/tui/commands.go`, `messages.go`, `model.go` (+ tests); `docs/clinban-board.md`,
`docs/log.md`.

**Tests (write first):**
- `previewLoadedMsg{content}` → viewport content equals exact bytes.
- selection change issues `loadPreview` for the new path.
- fixture with non-canonical frontmatter order → preview equals file bytes (NOT re-marshaled — guards ADR-4).

**Docs:** update `docs/clinban-board.md` (preview pane + `ctrl+d/ctrl+u` scroll); append `docs/log.md`.

**Done:** preview shows verbatim file content and scrolls; tests green; docs updated in this commit.

---

## T6 — Status advance (`>`)

**Deliverable:** `advanceStatus(st, id)` cmd — `FindByID`→`ReadTicket` (fresh) → `fsm.NextStatus`;
if a forward status exists, set it + refresh `Updated`, `WriteTicket` to the record path; else return
`noForward`. Bind `>`; success → `loadTickets` reload; `noForward`/error → status/error line, snapshot
unchanged until success. Reload preserves selection by **ticket ID** (re-select the same ID in the
re-sorted list; clamp if gone) — a shared helper used by every reload.

**Files:** `internal/tui/commands.go`, `messages.go`, `model.go`, `keys.go` (+ tests);
`docs/clinban-board.md`, `docs/log.md`.

**Tests (write first, temp-dir Store):**
- backlog fixture + `advanceStatus` → on-disk status `in-progress`, `Updated` refreshed.
- done fixture → `noForward`, file unchanged.
- write error path (e.g. read-only dir) → `statusAdvancedMsg{err}` surfaced, snapshot not mutated.
- `>` key issues `advanceStatus` for the selected id; success issues reload.
- after advancing the selected ticket (re-sorted into another group), the cursor stays on that ticket ID, not the old index; missing ID → clamped.

**Docs:** update `docs/clinban-board.md` (`>` advance behavior, selection-stable reload); append `docs/log.md`.

**Done:** pressing `>` advances the selected ticket, the board reflects the persisted state, and the
cursor stays on the acted-on ticket after the re-sort; tests green; docs updated in this commit.

---

## T7 — Editor handoff (`tea.ExecProcess`) + commit gate

**Deliverable:** `e` key flow per design §"Edit handoff sequence" — **no blocking I/O in `Update`**:
`beginEdit(st, selectedID)` fresh-resolves livePath (`FindByID`), reads live bytes, creates the same-dir
`os.CreateTemp` scratch, builds `editor.Command(scratch)`, returns `editReadyMsg{scratch, livePath, cmd}`
or `editBeginFailedMsg{err}`. `Update` on `editReadyMsg` runs `tea.ExecProcess(cmd, …)` →
`editFinishedMsg` → `commitEdit(st, scratch, editLive, base)`. `commitEdit` sequence: read scratch
(read error → `parseOrIOErr`); then `allIDs, err := st.AllIDs()` — **`AllIDs` returns `([]string, error)`;
the scan error is collected here and surfaced as `parseOrIOErr`, original untouched, no write**; then
`lint.ValidateForCommit(raw, base(livePath)[:4], base, allIDs)`; clean → `WriteTicket(livePath)` (refresh
`Updated`) + reload. The three `parseOrIOErr` sources (scratch read, `st.AllIDs()` scan, `ticket.Parse`)
are reported distinctly from `lintErrs`, per design §"commitEdit failure modes". Scratch removed on every
terminal outcome.

**Files:** `internal/tui/commands.go`, `messages.go`, `model.go`, `keys.go` (+ tests);
`docs/clinban-board.md`, `docs/log.md`.

**Tests (write first, temp-dir Store):**
- `beginEdit`: success → scratch copies live bytes, livePath fresh-resolved; `editor.Command` error → `editBeginFailedMsg`, scratch removed.
- `commitEdit` valid scratch → live file updated + `Updated` refreshed + reload issued.
- `commitEdit` **scratch read failure** (scratch removed/unreadable) → `parseOrIOErr` set, live file byte-identical, no write.
- `commitEdit` **`st.AllIDs()` scan failure** (e.g. unreadable archive dir fixture) → `parseOrIOErr` set, live file byte-identical, `lint.ValidateForCommit` not reached, no write.
- `commitEdit` **parse error** (bad frontmatter) → `parseOrIOErr` set, live file byte-identical, no write.
- `commitEdit` **lint violation** → `lintErrs` set (`parseOrIOErr` nil), live file byte-identical, no write.
- scratch removed on every outcome (no orphan).

**Docs:** update `docs/clinban-board.md` (`e` edit handoff, parse/IO-vs-lint rejection, original-untouched guarantee); append `docs/log.md`.

**Done:** edit round-trips through `$EDITOR`; `Update` performs no filesystem I/O; `st.AllIDs()`/scratch/parse
failures are surfaced distinctly from lint and never write the live file; invalid edit rejected
without corrupting the original; terminal restored after handoff (verified in T8); tests green; docs updated in this commit.

---

## T8 — Verification + docs reconciliation

**Deliverable:** spike acceptance closed. (The `docs/clinban-board.md` page, `docs/cli.md`,
`docs/index.md`, and `docs/log.md` entries were authored incrementally in T4–T7; T8 verifies and
reconciles them, it does **not** write them for the first time.)
- Confirm canonical Bubble Tea / Bubbles / Lip Gloss **v2 import paths** in go.mod.
- Manual normal-terminal checks: resize, editor handoff, and a forced panic in `Update` each leave the
  terminal restored (cooked mode, main screen) on exit.
- `go test ./... && go vet ./...` green; `gofmt -w` clean.
- Docs reconciliation (use `/librarian`): confirm `docs/clinban-board.md` documents the **full landed
  keymap** (`j/k`, `ctrl+d/ctrl+u`, `e`, `>`, `r`, `?`, quit) and matches actual behavior; `docs/cli.md`
  lists `clinban board`; `docs/index.md` links the new page; `docs/log.md` has the full T4–T7 trail;
  reconfirm the `cmd/clinban/schema.md` relevance decision (no change expected).

**Files:** `docs/` (verification/reconciliation of pages landed in T4–T7); any cleanup.

**Done:** every Spike Acceptance item in `02_architecture.md` is satisfied and checked; docs verified
complete and consistent with landed behavior; ticket 0021 ready to move to done.

---

## ADR locks every task must honour
- TUI never calls `os.Link`/`os.Remove`/`os.Rename` on managed tickets; no ID/transition recompute. All
  mutation via `store.WriteTicket` + `fsm`. (Edit scratch via `os.CreateTemp` only, mirroring `edit.go`.)
- Preview = original file bytes from `Record.Path`, never a re-marshaled `Ticket`.
- Status/edit writes use a **fresh** disk read, never the in-memory snapshot.
- No stdin prompts under the alt-screen — invalid edits surface in-UI.
- `editor.Command` returns a cmd with stdio unset (caller wires it).
