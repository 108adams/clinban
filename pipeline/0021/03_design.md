# Design: Terminal UI Foundation (ticket 0021)

_Created: 2026-06-14_
_Input: `pipeline/0021/02_architecture.md` (accepted, cross-reviewed) + `tickets/0021-choose-terminal-ui-foundation.md`_

This is the implementation design for the phase-two TUI spike. It translates the architecture into
module boundaries, interface contracts, a test strategy, and coding standards. Task breakdown lives in
`pipeline/0021/04_tasks.md`.

## Resolved open questions (from architecture)

| Question | Resolution |
|----------|------------|
| Home for shared board ordering | Extract `internal/board` (`StatusOrder` + `Less`, ticket-only dep); `cmd/clinban/list.go` + TUI sort `[]store.Record` via `board.Less` |
| Shared parse+lint edit gate | Extract `lint.ValidateForCommit`; adopted by `edit`, `register`, and the TUI edit gate |
| Full keymap (spike) | `j/↓ k/↑` select · `ctrl+d/ctrl+u` scroll preview · `e` edit · `>` advance status · `r` reload · `?` help · `q`/`ctrl+c`/`esc` quit. Richer UX (palette/modal, filtering) deferred |
| Chroma highlighting / archived view / interactive filtering | Deferred to follow-up tickets (per architecture) |

## Module structure

### 1. `internal/board` (new package — board display order)

Single responsibility: the canonical board ordering, shared by the `list` command and the TUI.
Depends on `internal/ticket` **only** — a display-order package should not depend on the filesystem
record type, so it sorts by ticket fields and callers adapt their own `[]store.Record`.

```
board.go
  // StatusOrder ranks statuses for board display (lower sorts earlier).
  var StatusOrder = map[ticket.Status]int{
      StatusInProgress: 0, StatusBlocked: 1, StatusBacklog: 2, StatusDone: 3,
  }
  // Less reports whether ticket a sorts before b: board order, then ascending numeric ID.
  func Less(a, b *ticket.Ticket) bool
doc.go
```

Imports `internal/ticket` only (not `store`). Callers sort their own slices, e.g.
`sort.SliceStable(recs, func(i, j int) bool { return board.Less(recs[i].Ticket, recs[j].Ticket) })`.
`cmd/clinban/list.go` keeps a thin `sortRecords` that delegates to `board.Less`; `internal/tui` does the
same when feeding the list. No cycle (`store` does not import `board`). Filtering (`applyFilters`) stays
in `cmd/clinban/list.go` — not needed by the spike, deferred.

### 2. `internal/editor` (extend — single-source the invocation)

```
editor.go
  // Command builds the editor *exec.Cmd for path with stdio UNSET.
  // Callers wire stdio: Open sets os.Std*; tea.ExecProcess wires the tty.
  func Command(path string) (*exec.Cmd, error)
  // Open re-implemented on Command (sets os.Stdin/Stdout/Stderr, then Run).
  func Open(path string) error
```

The existing unexported `command()` resolver (editor.go:34) is reused by `Command`. **`Command` must
not set `Stdin/Stdout/Stderr`** — that is the caller's job. This is the contract that lets the same cmd
serve both the blocking CLI `Open` and Bubble Tea's `tea.ExecProcess`.

### 3. `internal/lint` (extend — shared commit validation)

```
lint.go (or commit.go in package lint)
  // ValidateForCommit parses raw, assigns id, and lints with filename + allIDs.
  // Returns (nil,nil,parseErr) on parse failure; (t, lintErrs, nil) otherwise.
  func ValidateForCommit(raw []byte, id, filename string, allIDs []string) (*ticket.Ticket, []LintError, error)
```

Kernel sequence: `ticket.Parse(raw)` → `t.ID = id` → `Lint(t, filename, allIDs)`. Callers keep their
distinct surrounding logic:
- `cmd/clinban/edit.go`: `id = base(livePath)[:4]`, `filename = base`; reopen loop + atomic write unchanged.
- `cmd/clinban/register.go`: `id = fmt("%04d", nextID)`, `filename = finalFilename`; timestamps + path-containment unchanged.
- `internal/tui`: `id = base(livePath)[:4]`, `filename = base` (same as edit).

### 4. `internal/tui` (new package — the Bubble Tea model)

```
model.go    Model + New(st) + Init/Update/View
keys.go     keyMap (key.Binding set) + ShortHelp/FullHelp
commands.go tea.Cmd factories — the only place that touches the store (I/O seam)
messages.go message types returned by the commands
view.go     Lip Gloss layout helpers (may fold into model.go for the spike)
doc.go
```

`Model` fields (indicative):
```
type Model struct {
    st        *store.Store
    records   []store.Record   // current snapshot
    list      list.Model       // bubbles/list (left pane), fed board-sorted records
    preview   viewport.Model   // bubbles/viewport (right pane), original file bytes
    keys      keyMap
    help      help.Model
    width, height int
    err       error            // load/error state — board does not render a half-list
    status    string           // transient status/lint message line
    scratch   string           // scratch path during an in-flight edit ("" otherwise)
    editLive  string           // live path of the in-flight edit, fresh-resolved at beginEdit
}
```

- `New(st *store.Store) Model` — constructs the model; does not perform I/O.
- `Init() tea.Cmd` — returns `loadTickets(st)`.
- `Update(msg) (Model, tea.Cmd)` — pure dispatch over key/window/result messages; no blocking I/O.
- `View() string` — left list `│` divider `│` right preview, with a bottom status/help line, sized to `width/height`.

`board` is consumed when feeding the list: records are sorted with `board.Less` via `sort.SliceStable`
before building list items.

### 5. `cmd/clinban/board.go` (new command — thin entry)

```
boardCmd = &cobra.Command{ Use: "board", Short: ..., Args: cobra.NoArgs, RunE: runBoard }
func init() { rootCmd.AddCommand(boardCmd) }
func runBoard(cmd, args) error {
    m := tui.New(st)                       // st set by rootCmd PersistentPreRun
    _, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
    return err
}
```

No new bootstrap — rides `rootCmd.PersistentPreRun` (root.go:48) for `st`/`cfg`.

## Interface contracts (= unit-test boundaries)

### tea.Cmd seam (`internal/tui/commands.go`)
All store access is funnelled through these. They run off the Update path and return a message.
Tested over a temp-dir `*store.Store` with fixture files — real filesystem, no mocks.

| Command | Reads/Writes | Returns msg |
|---------|--------------|-------------|
| `loadTickets(st)` | `st.ListActive()` | `ticketsLoadedMsg{records, err}` |
| `loadPreview(path)` | `os.ReadFile(path)` (original bytes) | `previewLoadedMsg{content, err}` |
| `advanceStatus(st, id)` | `FindByID`→`ReadTicket`(fresh)→`fsm.NextStatus`→set status+`Updated`→`WriteTicket` | `statusAdvancedMsg{err, noForward bool}` |
| `beginEdit(st, id)` | `FindByID(id)` (fresh livePath) → read live bytes → same-dir `os.CreateTemp` scratch copy → `editor.Command(scratch)` | `editReadyMsg{scratch, livePath, cmd}` or `editBeginFailedMsg{err}` (any partial scratch removed) |
| `commitEdit(st, scratch, livePath, filename)` | read scratch → `lint.ValidateForCommit(raw, base(livePath)[:4], filename, st.AllIDs())`; clean → `WriteTicket(livePath)` (refresh `Updated`) | `editCommittedMsg{lintErrs, parseOrIOErr}` — see failure modes below |

### Edit handoff sequence (`e` key) — no blocking I/O in `Update`
1. `Update` on `e`: return `beginEdit(st, selectedID)` (a `tea.Cmd`). **No filesystem work happens in
   `Update`** — the fresh livePath resolution, scratch copy, and `editor.Command` construction all run
   inside `beginEdit`.
2. `Update` on `editBeginFailedMsg{err}`: surface the error; nothing to clean (beginEdit already removed
   any partial scratch).
3. `Update` on `editReadyMsg{scratch, livePath, cmd}`: store `scratch`/`editLive` on the model; return
   `tea.ExecProcess(cmd, fn)` where `fn(err)` yields `editFinishedMsg{err}`.
4. `Update` on `editFinishedMsg{err}`: if the editor itself errored, remove scratch + clear state +
   surface the error; otherwise return `commitEdit(st, scratch, editLive, base(editLive))`.
5. `Update` on `editCommittedMsg`: always remove scratch and clear `scratch`/`editLive`, then:
   - `parseOrIOErr != nil` (parse failure, scratch read failure, or `AllIDs` scan failure) → error
     state, original untouched — reported distinctly from lint output.
   - `len(lintErrs) > 0` → show lint errors in the status line, original untouched (user may press `e`
     again — an in-UI reopen, never a stdin prompt).
   - success → return `loadTickets(st)` to reload.

**commitEdit failure modes:** scratch read error, `ticket.Parse` error, and `st.AllIDs()` error all
leave the live file untouched and are reported via `editCommittedMsg.parseOrIOErr` (separate from
`lintErrs`), satisfying the project's parse-vs-lint distinction. The scratch is removed on every outcome.

**Scratch cleanup policy:** the scratch is a same-directory `os.CreateTemp(".clinban-edit-*.md")` file,
removed in every terminal handler (`editBeginFailed`, `editFinished` error, `editCommitted`). A scratch
left behind by a hard crash is harmless: its `.`-prefixed name never matches the managed
`^[0-9]{4}-…\.md` convention (scan.go:13), so `ListActive` ignores it — matching the CLI `edit`
best-effort cleanup (edit.go:66).

### Reload & selection stability
After any reload (`loadTickets` following a status advance or edit commit), selection is preserved by
**ticket ID**, not list index: locate the previously selected ID in the freshly sorted list and
re-select it; if that ID is gone (e.g. moved out of the active set), clamp to the nearest valid index.
This keeps the cursor on the ticket the operator was acting on — important because a status advance
re-sorts the ticket into a different group.

### Edit concurrency (scope note)
`commitEdit` writes the user's freshly-edited **scratch content** to the live path; it does not write
from the in-memory board snapshot (the architecture's "never writes from the snapshot" lock holds). This
mirrors the CLI `edit` command, which also writes the edited scratch over the live file without a
pre-write re-read (edit.go:113–117). The narrow window where an external edit between `beginEdit` and
`commitEdit` could be overwritten is identical to the CLI's and accepted for a single-user tool;
content-level conflict detection (fingerprint compare) is a deferred enhancement, not spike scope. The
**status advance** path is different and does fresh-read-before-write (see Locks).

### Locks honoured (from ADRs — non-negotiable)
- `internal/tui` never calls `os.Link`/`os.Remove`/`os.Rename` on managed tickets, never recomputes
  IDs or transitions. All mutation via `store.WriteTicket` + `fsm`. (The only `os` write it performs is
  the edit scratch copy via `os.CreateTemp`, mirroring `edit.go`.)
- Preview content = original file bytes from `Record.Path`, never a re-marshaled `Ticket`.
- Status writes use a **fresh** `FindByID`+`ReadTicket`, never the in-memory snapshot.
- Edit fresh-resolves the live path via `FindByID(selectedID)` at `beginEdit` (not the snapshot's `Path`)
  and commits the user's edited scratch content (content-level conflict detection deferred — see Edit concurrency).
- No stdin prompts while the alt-screen is active.
- `editor.Command` returns a cmd with stdio unset.

## Test strategy

### Unit (no TTY — direct `Msg` injection into `Update`)
- Navigation: `j/k`/arrows move selection; clamps at first/last.
- `WindowSizeMsg`: stores `width/height`; layout recomputes; no panic at tiny sizes.
- `ticketsLoadedMsg{err}` → error state set, list not populated (no half-board).
- `ticketsLoadedMsg{records}` → list populated in board order.
- `previewLoadedMsg` → preview content set to exact bytes.
- `statusAdvancedMsg{noForward}` → status message, no reload; `{err}` → error surfaced; success → reload cmd issued.
- `editCommittedMsg{lintErrs}` → status shows errors, no reload; success → reload cmd issued.
- Quit keys → `tea.Quit`.
- Reload preserves selection by ticket ID: after a status advance that re-sorts the selected ticket, the cursor stays on that ID (not the old index); missing ID → clamped index.

### Store-backed command tests (temp-dir Store, real fs, no mocks)
- `advanceStatus`: backlog fixture → file on disk has `in-progress` + refreshed `Updated`; done fixture → `noForward`, file unchanged.
- `beginEdit`: success → scratch is a same-dir copy of the live bytes, livePath fresh-resolved; `editor.Command` error → `editBeginFailedMsg`, scratch removed (no orphan).
- `commitEdit` (three distinct outcomes): valid scratch → live file updated, `Updated` refreshed; **parse error** (bad frontmatter) → `parseOrIOErr` set, live file byte-identical, `WriteTicket` not reached; **lint violation** → `lintErrs` set (`parseOrIOErr` nil), live file byte-identical, `WriteTicket` not reached. Scratch removed in every case.
- `loadTickets`: unparseable file in dir → error returned (whole-board failure, matching `store.go:234`).

### Package-unit tests
- `board.Less`: mixed statuses + IDs → correct group order then ID-ascending; drives a stable `sort.SliceStable`.
- `lint.ValidateForCommit`: parse error → `(nil,nil,err)`; lint violation → `(t, errs, nil)`; clean → `(t, empty, nil)`.
- `editor.Command`: `$EDITOR` resolution, `--wait` appended for GUI editors, path last, **stdio nil**.

### Regression (must stay green)
- Existing `internal/editor`, `internal/lint`, and `cmd/clinban` list/edit/register tests after the
  three refactors (T1–T3). Behavior is unchanged; only call-sites move.

### Critical paths gated before ship (3)
1. **Edit commit gate** — invalid scratch never reaches the live file (data integrity).
2. **Status advance** — fresh-read → write; done → no-op.
3. **Load error** — error state, never a partial board.

### Manual (normal terminal — spike acceptance, ticket:168)
Resize, editor handoff, and a forced panic in `Update` each leave the terminal restored (cooked mode,
main screen) on exit. Confirm canonical Bubble Tea / Bubbles / Lip Gloss **v2 import paths** before
writing TUI code.

## Coding standards for this work
- TDD: stub → failing test (confirm FAIL) → implement (GREEN) → quality gate, per task.
- All Go commands with `GOCACHE=/tmp/clinban-gocache`; run `go test ./... && go vet ./...`; `gofmt -w` changed files.
- One commit per task (refactor/feature → tests → docs together).
- Update `docs/` + append `docs/log.md` in the same commit as the behavior change (new `clinban board`
  page in T8); use `/librarian` for the docs page.
- Do not commit generated binaries (`clinban`, `clinban.test`).
