# Architecture: Terminal UI Foundation (ticket 0021)

_Created: 2026-06-14_

Spike/ADR ticket. The Charm-stack decision is already accepted in the ticket body and
ratified by `tickets/RANKING.md` (Tier 2). This document ratifies that decision, fixes the
component boundary, and resolves the ticket's open questions so downstream TUI implementation
tickets can be planned.

## Existing Components (verified)

| Component | File:line | Responsibility |
|-----------|-----------|----------------|
| `rootCmd` `PersistentPreRun` | cmd/clinban/root.go:48 | Finds project root, loads config, builds package-level `st`/`cfg` before any subcommand runs |
| command self-registration | cmd/clinban/register.go:30, list.go:51 | Each command registers via `init()` + `rootCmd.AddCommand` |
| `store.New(cfg)` | internal/store/store.go:30 | Constructs the Store from config |
| `Store.ListActive()` → `[]Record` | internal/store/store.go:249 | Parses active tickets; returns empty (never nil) slice; errors on any unreadable/unparseable file (store.go:234) |
| `store.Record{Ticket,Path,InArchive}` | internal/store/store.go:48 | Pairs parsed ticket with its on-disk path |
| `Store.ReadTicket` / `WriteTicket` | cmd/clinban/move.go:61,98 | Read/atomic-write a single ticket file |
| `Store.FindByID` | internal/store/scan.go:118 | Locate a ticket file by 4-digit ID |
| `ticket.Ticket` / `ticket.Parse` | internal/ticket/ticket.go:22,70 | In-memory ticket model + Markdown+frontmatter parse |
| `fsm.ValidateTransition` / `NextStatus` | internal/fsm/fsm.go:25,45 | Workflow rules; pure functions |
| `lint.Lint(t, filename, allIDs)` | internal/lint/lint.go:55 | Schema validation; returns `[]LintError` |
| `editor.Open(path)` | internal/editor/editor.go:16 | Launches `$EDITOR` (fallback `vi`) as a blocking child inheriting stdio |
| `editor.command(path)` (unexported) | internal/editor/editor.go:34 | Resolves editor name+args incl. `--wait` for GUI editors |
| board ordering: `statusOrder` / `sortRecords` / `applyFilters` | cmd/clinban/list.go:19,127,85 | Display order (in-progress → blocked → backlog → done) + filters — currently package `main`, unreachable from other packages |

All filesystem mutation and ticket-truth logic is encapsulated in `internal/`. The CLI layer is a
thin coordinator. The TUI must preserve this: it is a new human interface, not a new source of truth.

## Proposed Changes

| Change | Replaces/extends | Rationale |
|--------|-----------------|-----------|
| New package `internal/tui` | New surface | Bubble Tea model (Model/Update/View), keymap, in-memory `[]store.Record` snapshot. Pure consumer of config/store/fsm/lint/editor |
| New command `cmd/clinban/board.go` (`clinban board`) | New surface, self-registers | Thin entry: reuse package-level `st`/`cfg` (already initialised by `root.go:48`), build the tui model, run `tea.Program` |
| Extract `editor.Command(path) (*exec.Cmd, error)` | Promotes unexported `editor.command()` + assembly (editor.go:16–52) | Single-source the editor invocation; `Open` re-implements on top; TUI hands the `*exec.Cmd` to `tea.ExecProcess` |
| Make board ordering reachable | Extracts `statusOrder`/`sortRecords` out of `cmd/clinban/list.go` | Avoid a 4th divergent copy of board order when the TUI needs the same sort; home is a Tech Lead call |
| Add Charm dependencies | go.mod | `bubbletea/v2`, `bubbles/v2`, `lipgloss/v2`. Confirm canonical v2 import paths at implementation time |

No changes to `internal/store`, `internal/ticket`, `internal/fsm`, `internal/lint`, `internal/config`.
The TUI adds **no new ticket-mutation primitive and no new ID/status logic**: every managed-ticket
write goes through `Store.WriteTicket`, and edits are gated by parse+lint before commit (ADR-3). The
only transient file it creates is the edit scratch copy, reusing the same same-directory
`os.CreateTemp` pattern the `edit` command already uses (cmd/clinban/edit.go:61).

## Integration Contracts

| Dependency | Protocol | Format | Failure mode | Owner |
|------------|----------|--------|-------------|-------|
| Bubble Tea runtime | in-process library | Model / Msg / Cmd | Panic in Update/View must restore terminal (exit alt-screen, raw mode off) before propagating; never leave the user's terminal corrupted | charmbracelet |
| `$EDITOR` child (via `tea.ExecProcess`) | child process; stdio inherited after Bubble Tea releases the terminal | `*exec.Cmd` from `editor.Command` | Non-zero exit → completion `Msg` carries the error → TUI shows in-UI error, keeps current snapshot; Bubble Tea restores the terminal on resume | editor pkg + Bubble Tea |
| `Store.ListActive` | in-process | `[]store.Record` | Read/parse error on any file → whole call errors (store.go:234) → TUI shows error state, does not partially populate the list | store |
| `fsm.ValidateTransition` / `NextStatus` | in-process pure fn | `ticket.Status` | Forbidden/terminal → error or `("",false)` → TUI shows a transient message, performs no write | fsm |
| `lint.Lint` (post-edit reload) | in-process | `[]LintError` | Invalid ticket after edit → surfaced as an in-UI error badge; **not** the CLI's stdin reopen prompt (incompatible with an alt-screen TUI) | lint |

`os.Link`/`os.Remove` and atomic writes stay entirely inside `internal/store`; the TUI never calls them.

## TUI Write & Edit Contracts

The TUI never writes from its in-memory snapshot. Both mutating actions read fresh from disk
immediately before writing, mirroring the CLI commands, so a stale snapshot can never clobber a
concurrent external edit (another `clinban` command or an external editor).

- **Status advance:** on the advance key, re-resolve the selected ticket via `FindByID` +
  `ReadTicket` (fresh bytes), compute the next status with `fsm.NextStatus`, refresh `Updated`, and
  write with `Store.WriteTicket` to the record's path — the same sequence as `runMove`
  (cmd/clinban/move.go:60–102). A `WriteTicket` error is shown in-UI; the in-memory snapshot is not
  mutated until the write succeeds. On success the board reloads via `ListActive`.
- **Edit:** per ADR-3 — scratch copy → `tea.ExecProcess` → parse+lint → commit via `Store.WriteTicket`
  on success only → reload. The original is preserved on any parse/lint failure.

Both actions reload from disk after a successful write rather than mutating the snapshot in place, so
the board always reflects exactly what is persisted.

## NFRs

| Category | Requirement | Target | Status |
|----------|------------|--------|--------|
| Testability | Model logic testable without a TTY | Navigation, selection, status-advance, and error states unit-testable via direct `Msg` injection (and/or `teatest`); no real terminal required for logic | Required — the core rationale for choosing Bubble Tea |
| Security | Attack surface | No new surface. Editor exec uses the same `$EDITOR` resolution as the CLI (editor.go:35) via `exec.Command` with an arg slice — no shell interpolation | No change |
| Resilience | Defined degradation for each failure mode | Editor non-zero exit, store read error, render panic → each restores the terminal and shows an error state | Required |
| Performance | Reload cost | On-demand re-scan; O(n) file reads per reload; single-user CLI — no target | Trivially satisfied |
| Observability | Error surfacing | Errors shown in the TUI status line; no new logging subsystem | Sufficient for a local interactive tool |
| Availability / Scalability / Compliance / Data / Operability | — | N/A for a local single-user CLI | N/A |

## ADRs

## ADR-1: Charm stack as the TUI foundation

**Status:** `accepted`
**Decision:** Build the TUI on Bubble Tea v2 (runtime), Bubbles v2 (list, viewport, help, key bindings), and Lip Gloss v2 (layout/styling).
**Context:** Ticket 0021 records this as a Proposed ADR; RANKING ratifies it as the gate for phase-two TUI work. The hard requirement is a testable, maintainable UI that grows from two panes into real workflows, not "draw two boxes."
**Alternatives:**
| Option | Rejected because |
|--------|-----------------|
| `jroimartin/gocui` / `awesome-gocui/gocui` | Mutable-view/callback model; weak maintenance signal; poor testability as workflows grow |
| `rivo/tview` | Widget/application-oriented; pushes UI policy into widget state; weaker behavior testability — credible fallback only if the Bubble Tea spike fails |
| `gdamore/tcell` / raw ANSI | Too low-level; forces a local app loop, focus, scrolling, and input handling |
**Rationale:** The Model/Update/View loop maps cleanly onto Clinban's existing message-free domain calls and keeps UI state explicit and unit-testable without a terminal. Bubbles supplies the boring primitives (list, viewport, help, keybindings).
**Consequences:**
- `+` UI behavior testable without a TTY
- `+` Layout/styling stays declarative and separate from domain ops
- `-` New dependency family; v2 import paths must be confirmed at implementation time
- `!` Some Bubbles components may be too opinionated and need local wrappers over time
**Locks:** The TUI runtime is Bubble Tea v2. Implementation tickets do not re-open the framework choice unless the spike demonstrably fails.

## ADR-2: TUI is a pure consumer in `internal/tui`; `clinban board` is a thin entry

**Status:** `accepted`
**Decision:** Add `internal/tui` (the model) and `cmd/clinban/board.go` (the `clinban board` command). The TUI consumes config/store/fsm/lint/editor exactly as the CLI handlers do and introduces no ticket-truth logic.
**Context:** The ticket mandates "keep the TUI out of domain packages; the UI is a consumer, not a new source of ticket truth." All mutation and ID/status rules already live in `internal/`.
**Alternatives:**
| Option | Rejected because |
|--------|-----------------|
| Put TUI logic in package `main` | Untestable from other packages; mixes UI with command wiring |
| Let the TUI write/move files directly | Duplicates store logic; breaks the single-source-of-truth boundary |
**Rationale:** Minimal new surface. `root.go:48` already builds `st`/`cfg` for every subcommand, so `board` needs no new bootstrap. A separate package keeps the model unit-testable.
**Consequences:**
- `+` Domain boundary preserved; model unit-testable in isolation
- `+` `clinban board` rides the existing config/store init for free
- `-` The board-order contract (in-progress → blocked → backlog → done) must not diverge between CLI and TUI; sharing `statusOrder`/`sortRecords` (currently `cmd/clinban/list.go:19,127`) is the clean way. The extraction home is a Tech Lead call (recommend `internal/board`); the spike may re-derive the order inline if extraction is deferred
**Locks:** `internal/tui` may not call `os.Link`/`os.Remove`/`os.Rename` or recompute IDs/transitions. All mutation goes through `internal/store` and `internal/fsm`. Invocation is the subcommand `clinban board`.

## ADR-3: Editor handoff via extracted `editor.Command` + `tea.ExecProcess`, edited through a scratch copy with a parse+lint commit gate

**Status:** `accepted`
**Decision:** Extract the editor invocation into `editor.Command(path) (*exec.Cmd, error)`; `editor.Open` re-implements on top of it. To edit, the TUI copies the selected ticket to a same-directory scratch file (the existing `edit` pattern, edit.go:61), runs `editor.Command(scratchPath)` through `tea.ExecProcess` — which releases the terminal, runs the child with inherited stdio, restores the terminal, and emits a completion `Msg`. On that `Msg` the TUI re-reads the scratch file, parses, and lints; **only on success** does it commit via `Store.WriteTicket` to the live path (refreshing `Updated`) and reload. On parse/lint failure the original file is left untouched and the errors are shown in-UI.
**Context:** `editor.Open` (editor.go:16) is a blocking child that owns stdio; Bubble Tea owns the terminal (raw mode + alt-screen) — the runtime must suspend before the editor runs and resume after. Separately, the CLI `edit` command (edit.go:55–119) never edits the live file directly: it edits a scratch copy and commits only after parse+lint pass, preserving the original on invalid edits. The TUI must keep that integrity guarantee; the ticket mandates the TUI be a consumer, not a new source of ticket truth (tickets/0021-choose-terminal-ui-foundation.md:166).
**Alternatives:**
| Option | Rejected because |
|--------|-----------------|
| Edit the live file directly, surface lint errors after the fact | Bypasses `Store.WriteTicket`, lets invalid/partial saves persist, and makes the next `ListActive` fail the whole board load (any unparseable file errors the scan, store.go:234) — a data-integrity regression vs. the CLI |
| Wrap blocking `editor.Open` in manual `ReleaseTerminal`/`RestoreTerminal` | Puts terminal juggling and a blocking call inside the update flow; bypasses the idiomatic completion-message path |
| Quit the TUI, run editor, relaunch | Loses UI state; jarring UX |
**Rationale:** `tea.ExecProcess` is the idiomatic v2 suspend/resume path and needs an `*exec.Cmd`, so `editor.Command` single-sources the `$EDITOR`/`--wait` resolution. Driving it against a scratch copy and committing only on parse+lint success reuses the established `edit` safety model without the stdin reopen prompt (edit.go:124) — which cannot run under the alt-screen; an in-UI error/reopen action replaces the `[y/N]` prompt.
**Consequences:**
- `+` One source of truth for editor invocation; idiomatic terminal release/restore
- `+` Existing parse+lint commit gate preserved; invalid edits never reach the live file; all managed-ticket writes go through `Store.WriteTicket`
- `+` Completion `Msg` is the single place to re-read, lint, commit, and reload
- `-` The TUI carries scratch-path state and replaces the stdin `[y/N]` reopen loop with an in-UI reopen action
- `!` The scratch-copy + lint-gate logic now exists in two places (cmd `edit` and the TUI) unless extracted into a shared helper — see Open Questions
**Locks:** Editor invocation is single-sourced through `editor.Command`; live-file decisions stay out of the `editor` package. The TUI edits only a scratch copy and commits exclusively via `Store.WriteTicket` after parse+lint pass. The TUI must not prompt on stdin while the alt-screen is active. Post-edit reload goes through `store.ListActive`.

## ADR-4: Preview is raw Markdown source, never rendered; Chroma highlighting deferred

**Status:** `accepted`
**Decision:** The right preview pane shows the selected ticket's raw Markdown source verbatim in a scrolling viewport. "Verbatim" means the **original file bytes** (`os.ReadFile` on `Record.Path`), not a re-marshaled `Ticket` — `ticket.Marshal` normalizes and reorders frontmatter (ticket.go:128) and would not be verbatim. Markdown is **never rendered** (no Glamour, now or in the future). Syntax highlighting of the raw source, when added, uses Chroma (`github.com/alecthomas/chroma/v2`, Markdown lexer + terminal ANSI formatter). Highlighting is **deferred to a fast-follow** ticket, not the spike.
**Context:** The product owner wants the literal file contents visible (every `#`, `*`, backtick), not a re-rendered document. Rendering would hide structure and risk the "second Markdown product" the ticket warns against. Highlighting is a pure view-layer concern with zero coupling to store/fsm/editor.
**Alternatives:**
| Option | Rejected because |
|--------|-----------------|
| Glamour-rendered Markdown | Hides source; restructures the document; explicitly rejected by the PO |
| Chroma highlighting inside the spike | Conflates "foundation works" with "highlighting looks right"; adds a dependency the spike does not need to prove the foundation |
**Rationale:** Raw source matches the PO's intent and keeps the spike minimal. Chroma is the de-facto Go highlighter (Glamour itself uses it) and can be added later with no architectural impact.
**Consequences:**
- `+` Preview shows exactly what is on disk; spike stays small with no extra dependency
- `+` Highlighting can land later as a pure view change
- `-` Spike preview is uncolored plain text
- `!` Chroma emits ANSI; the viewport/Lip Gloss must measure width accounting for escape codes when highlighting lands (impl detail, not architecture)
**Locks:** Preview content is the original file bytes read from `Record.Path`, never re-marshaled `Ticket` content. The preview never renders/transforms Markdown structure. Glamour is permanently out of scope. The only future addition is source-level syntax highlighting via Chroma.

## Spike Acceptance (scope of this ticket)

The narrow spike (per the ticket's Implementation Spike Requirements) must:
load active tickets via existing config/store paths; render a two-pane board (`clinban board`) with
the preview pane showing the selected ticket's **original file bytes**; support `up/down` + `k/j`
selection and right-pane scrolling; handle terminal resize; exit on `q` and `ctrl+c`; show an error
state when tickets fail to load/parse; open `$EDITOR` for the selected ticket via `tea.ExecProcess`
**on a scratch copy** and commit only after parse+lint pass (original preserved on failure), then
reload; advance the selected ticket through one valid status transition (fresh read → `fsm.NextStatus`
→ `Store.WriteTicket` → reload). Active-only; raw uncolored preview; single-key status advance.

Verification (the ticket requires both a normal terminal and model/unit tests — tickets/0021-choose-terminal-ui-foundation.md:168):

- Confirm the canonical Bubble Tea / Bubbles / Lip Gloss **v2 import paths** before writing code.
- Model/unit tests without a real terminal for navigation, selection, status advance, edit
  commit/reject, and error states.
- Manual check in a **normal terminal**: resize, editor handoff, and a forced panic in `Update` each
  leave the terminal restored (cooked mode, main screen) on exit — terminal restoration is verified,
  not assumed.

## Open Questions

| Question | Owner | Blocking? |
|----------|-------|-----------|
| Extraction home for shared board ordering (`statusOrder`/`sortRecords`) — recommend `internal/board` | Tech Lead | No |
| Full keymap beyond the spike basics (which keys advance status, refresh, etc.) | Tech Lead | No |
| Extract the shared scratch-copy + parse+lint edit gate from `cmd/clinban/edit.go`, or duplicate it in the TUI? (adopting the gate itself is decided — ADR-3) | Tech Lead | No |
| Chroma syntax-highlighting timing | Resolved — fast-follow (first TUI implementation ticket) | No |
| Richer status-change UX (command palette / modal selector) | Deferred — future ticket | No |
| Interactive filtering vs. mirroring `clinban list` flags | Deferred — future ticket | No |
| Archived-ticket visibility in the TUI | Deferred — future ticket (first release is active-only) | No |
