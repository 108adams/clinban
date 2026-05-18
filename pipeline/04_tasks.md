# Tasks: Clinban

Input: `pipeline/03_design.md`, `pipeline/01_requirements.md`
Each task is one developer-day or less. Done criteria are explicit and testable.

---

## Foundation Layer

Tasks 1–8 establish internal packages. Task 1 must complete first.
Tasks 2–8 are independent of each other and may be worked in parallel after Task 1.

---

### T-01 · Project scaffold

**Deliverable:** Working Go module with all directories created and empty packages that compile.

**Steps:**
1. `go mod init clinban`
2. `go get github.com/spf13/cobra gopkg.in/yaml.v3 github.com/BurntSushi/toml golang.org/x/term`
3. Create directory tree: `cmd/clinban/`, `internal/config/`, `internal/ticket/`, `internal/store/`, `internal/lint/`, `internal/fsm/`, `internal/editor/`, `internal/slug/`, `internal/template/`
4. Add a stub `package` declaration in each directory.
5. Add stub `main.go` in `cmd/clinban/`.

**Done when:** `go build ./...` passes with no errors.
**Depends on:** nothing

---

### T-02 · `internal/ticket` — data model and schema contract

**Deliverable:** `Ticket` struct, `Status`/`Type` types and constants, `Parse`, `Marshal`.

**Key files:** `internal/ticket/ticket.go`, `status.go`, `tickettype.go`

**Implementation notes:**
- `Parse`: split content on `---\n` fence; YAML-decode the frontmatter block; remainder is `Body`.
- `Marshal`: encode frontmatter to YAML, wrap in `---` fences, append body.
- `Tags` serialises as `tags: []` when empty (use `yaml:",flow"` or explicit initialisation).
- Round-trip invariant: `Marshal(Parse(b))` must equal `b` for any valid ticket bytes.

**Done when:**
- `TestParseRoundTrip` passes: parse a fixture ticket, marshal it back, parse again; both parsed tickets have equal field values and body.
- `TestParseMalformed` passes: non-YAML frontmatter returns a non-nil error.
- `TestParseEmptyBody` passes: ticket with no body parses without error; `Body == ""`.

**Depends on:** T-01

---

### T-03 · `internal/fsm` — state machine

**Deliverable:** Transition table and `ValidateTransition`.

**Key file:** `internal/fsm/fsm.go`

**Implementation notes:**
- Use a `map[ticket.Status][]ticket.Status` for the valid-transitions table.
- `ValidateTransition` returns `nil` for valid transitions.
- For invalid transitions, the error message must list valid next statuses:
  `"cannot transition from \"blocked\" to \"done\"; valid transitions: in-progress"`

**Done when:**
- All 6 valid transitions return `nil`.
- All 10 invalid transitions return a non-nil error containing valid next statuses.
- Self-transitions (e.g. `backlog → backlog`) return an error (not in the valid table).

**Depends on:** T-01

---

### T-04 · `internal/slug` — title-to-slug

**Deliverable:** `Slugify(title string) string`.

**Key file:** `internal/slug/slug.go`

**Implementation notes:**
- Split on whitespace, take first 5 tokens.
- Lowercase each token; strip all characters that are not `[a-z0-9]`; join with `-`.
- A token that becomes empty after stripping is skipped (does not count toward the 5-word limit).

**Done when:**
- `Slugify("Fix login timeout on staging")` → `"fix-login-timeout-on-staging"`
- `Slugify("One two")` → `"one-two"` (fewer than 5 words)
- `Slugify("Hello, World! (urgent)")` → `"hello-world-urgent"`
- `Slugify("")` → `""`

**Depends on:** T-01

---

### T-05 · `internal/config` — configuration loading

**Deliverable:** `Config` struct and `Load`.

**Key file:** `internal/config/config.go`

**Implementation notes:**
- Use `github.com/BurntSushi/toml` for decoding.
- If `.clinban` is absent: return a `Config` with `TicketsDir = projectRoot` and `ArchiveDir = filepath.Join(projectRoot, "archive")`.
- If `.clinban` exists but is malformed: return error with the TOML parse error message.
- Partial config is valid: unset fields fall back to defaults.

**Done when:**
- Absent file returns a config with correct defaults, no error.
- Malformed TOML returns a non-nil error.
- Valid TOML with `tickets_dir = "tasks"` sets `TicketsDir` to the resolved absolute path; `ArchiveDir` defaults to `tasks/archive`.

**Depends on:** T-01

---

### T-06 · `internal/lint` — schema validation engine

**Deliverable:** `LintError`, `Lint`, and all 7 rule functions.

**Key files:** `internal/lint/lint.go`, `rules.go`

**Implementation notes:**
- Each rule is a private function: `func ruleRequiredFields(t, filename) []LintError`, etc.
- `Lint` calls all rules in order, concatenates results.
- Returns `[]LintError{}` (empty, never nil) when valid.
- Rule 4 (ID matches filename): extract leading digits from filename, compare with `t.ID`.
- Rule 7 (uniqueness): iterate `allIDs`, count occurrences of `t.ID`; flag if count > 1.
- `LintError.String()` format: `"<filename>: field '<field>': <message>"`

**Done when:**
- Each of the 7 rules has at least one test that produces a `LintError` and one that produces none.
- A fully valid ticket with two unique IDs returns an empty slice.
- A ticket missing `title` and with a duplicate ID produces exactly 2 errors.

**Depends on:** T-02

---

### T-07 · `internal/editor` — editor integration

**Deliverable:** `Open(path string) error`.

**Key file:** `internal/editor/editor.go`

**Implementation notes:**
- Resolve editor: `os.Getenv("EDITOR")`; if empty, use `"vi"`.
- Use `exec.Command(editor, path)` with `Stdin`, `Stdout`, `Stderr` set to `os.Stdin/out/err`.
- Return error if `cmd.Run()` returns non-nil (editor exited non-zero).

**Done when:**
- Package compiles.
- Manual verification deferred to T-17: the interactive `clinban new` integration test.

**Depends on:** T-01

---

### T-08 · `internal/store` — file store

**Deliverable:** Full `Store` implementation with atomic writes.

**Key files:** `internal/store/store.go`, `scan.go`, `write.go`, `move.go`

**Implementation notes:**
- `NextID`: read all filenames in both dirs matching `[0-9]{4}-*.md`; parse numeric prefix; return max+1. Return 1 if no matches.
- `FindByID`: match files where the numeric prefix equals the given ID string. TicketsDir first, then ArchiveDir. Return `ErrNotFound` if absent.
- `WriteTicket`: call `ticket.Marshal`, write to `path + ".tmp"` in the same directory, `os.Rename` to final path. Does **not** set `t.Updated`; the caller sets it before calling `WriteTicket`.
- `MoveToArchive`: `os.MkdirAll(ArchiveDir)` then `os.Rename`.
- `MoveToActive`: `os.Rename` from ArchiveDir into TicketsDir. Used only for archive-to-active transitions (e.g. `done → backlog` reopen); **not** for adopting external files.
- `ListActive`: read all `*.md` files in TicketsDir, parse each, return `[]Record`.
- `ListArchive`: read all `*.md` files in ArchiveDir, parse each, return `[]Record`.
- `AllIDs`: collect numeric prefix from all `*.md` files in both directories.
- Define `type Record struct { Ticket *ticket.Ticket; Path string; InArchive bool }` in `store.go`.
- Define `var ErrNotFound = errors.New("ticket not found")` in `store.go`.

**Done when:**
- `TestNextIDEmpty` passes: empty directory returns 1.
- `TestNextIDWithTickets` passes: directory with `0003-*.md` as highest returns 4.
- `TestFindByID` passes: finds in TicketsDir and ArchiveDir; missing returns `ErrNotFound`.
- `TestWriteTicketAtomic` passes: temp file does not exist after successful write; file contains correct content.
- `TestMoveToArchiveCreatesDir` passes: archive dir is created if absent.
- `TestListArchive` passes: tickets in ArchiveDir are returned as `Record` values with `InArchive == true`.

**Depends on:** T-02

---

## CLI Layer

Task 9 must complete before Tasks 10–17. Tasks 10–17 are independent of each other.

---

### T-09 · `cmd/clinban` scaffold

**Deliverable:** Cobra root command; config loading; `clinban --help` works.

**Key files:** `cmd/clinban/main.go`, `root.go`

**Implementation notes:**
- `main.go`: call `root.Execute()`, exit 1 on error.
- `root.go`: define `rootCmd`; add `PersistentPreRun` that:
  - Finds project root (walk up from cwd looking for `.clinban`; fall back to cwd).
  - Calls `config.Load(projectRoot)`.
  - Constructs `store.New(cfg)` and stores it on a package-level variable accessible to subcommands.
- All subcommand files register themselves via `init()` calling `rootCmd.AddCommand(...)`.

**Done when:** `go run ./cmd/clinban --help` prints usage without error.

**Depends on:** T-05, T-08

---

### T-10 · `clinban lint` command

**Deliverable:** `clinban lint` and `clinban lint <id>` implemented.

**Key file:** `cmd/clinban/lint.go`

**Behaviour:**
- No argument: load `store.AllIDs()`; call `store.ListActive()` and `store.ListArchive()` to iterate all tickets; run `lint.Lint` on each using the `Record.Path` filename.
- Single argument: `store.FindByID(id)`, load ticket, run `lint.Lint`.
- Unknown ID: print `"ticket not found"` to stderr, exit 1.
- Errors: print each `LintError.String()` to stdout, exit 1.
- No errors: exit 0 silently.

**Done when:** Acceptance criteria §7 lint from `01_requirements.md` are met.

**Depends on:** T-06, T-08, T-09

---

### T-11 · `clinban new --no-interactive`

**Deliverable:** Non-interactive ticket creation via flags.

**Key file:** `cmd/clinban/new.go` (the `--no-interactive` path)

**Flags:** `--title` (required), `--type` (required), `--body` (optional), `--tags` (optional, comma-separated).

**Behaviour:**
- Validate `--title` and `--type` non-empty; missing → stderr + exit 1.
- Assign next ID; build `Ticket` struct with `Created = Updated = time.Now()`, `Status = backlog`.
- Run `lint.Lint`; if errors → print to stderr, exit 1.
- Write ticket via `store.WriteTicket` to `store.TicketPath(id, slug.Slugify(title))`.
- On success: print `"created: <filename>"` to stdout, exit 0.

**Done when:** Acceptance criteria §7 `new --no-interactive` from `01_requirements.md` are met.

**Depends on:** T-04, T-06, T-08, T-09

---

### T-12 · `clinban list`

**Deliverable:** `clinban list` with optional filters.

**Key file:** `cmd/clinban/list.go`

**Flags:** `--status`, `--type`, `--tag` (all optional, combinable).

**Behaviour:**
- Load all active tickets via `store.ListActive()`.
- Apply filters (AND logic for multiple flags).
- Sort: `in-progress` first, `blocked` second, `backlog` third, `done` last; within each group, ascending by ID.
- If empty after filtering: print `"No active tickets"` to stdout, exit 0.
- Print one line per ticket: `<id>  <status>  <type>  <title>` — truncate title to fit terminal width (use `golang.org/x/term` to get width; default 80 if not a terminal).

**Done when:** Acceptance criteria §7 list from `01_requirements.md` are met.

**Depends on:** T-08, T-09

---

### T-13 · `clinban move`

**Deliverable:** `clinban move <id> <status>` with FSM enforcement.

**Key file:** `cmd/clinban/move.go`

**Behaviour:**
- Resolve ticket by ID; unknown → `"ticket not found"` + exit 1.
- Parse target status; invalid value → print valid status list + exit 1.
- If current == target: exit 0 silently.
- `fsm.ValidateTransition(current, target)`; invalid → print error with valid next statuses + exit 1.
- Update `Status`, call `store.WriteTicket`.
- Print `"<id> moved to <status>"`, exit 0.
- Special case: `done → backlog` calls `store.MoveToActive` if ticket is in archive, then updates status.

**Done when:** Acceptance criteria §7 move from `01_requirements.md` are met.

**Depends on:** T-03, T-08, T-09

---

### T-14 · `clinban archive`

**Deliverable:** `clinban archive` (bulk) and `clinban archive <id>` (single).

**Key file:** `cmd/clinban/archive.go`

**Behaviour — single (`clinban archive <id>`):**
- Resolve ticket; unknown → `"ticket not found"` + exit 1.
- Status not `done` → `"ticket must be in 'done' status to archive"` + exit 1.
- `store.MoveToArchive(path)`; print `"archived: <filename>"`, exit 0.

**Behaviour — bulk (`clinban archive`):**
- `store.ListActive()` (returns `[]Record`); filter `r.Ticket.Status == done`.
- None found → `"No done tickets to archive"`, exit 0.
- List filenames; prompt `"Archive N ticket(s)? [y/N] "`.
- Read one character from stdin; if `y` or `Y`: `store.MoveToArchive(r.Path)` for each record, print count, exit 0. Otherwise exit 0.

**Done when:** Acceptance criteria §7 archive from `01_requirements.md` are met.

**Depends on:** T-08, T-09

---

### T-15 · `clinban register`

**Deliverable:** `clinban register <path>` for automata file adoption.

**Key file:** `cmd/clinban/register.go`

**Behaviour:**
- Read file at `<path>`; not found → `"file not found"` + exit 1.
- Parse as ticket (YAML frontmatter); parse error → print error + exit 1.
- Overwrite `ID`, `Created`, `Updated` with system-assigned values (`time.Now()`).
- `store.AllIDs()`, run `lint.Lint`.
- Lint errors → print each to stderr, exit 1. Do not move the file.
- No errors → compute `finalPath = store.TicketPath(id, slug.Slugify(t.Title))`; validate `finalPath` is within `store.TicketsDir` (path containment check — prevents path traversal); `t.Updated = time.Now()`; `store.WriteTicket(t, finalPath)`; delete the source file at `<path>` if it differs from `finalPath`; print `"registered: <filename>"`, exit 0.

**Done when:** Acceptance criteria §7 register from `01_requirements.md` are met.

**Depends on:** T-02, T-06, T-08, T-09

---

### T-16 · `clinban show`

**Deliverable:** `clinban show <id>` — print ticket to stdout in human-readable format.

**Key file:** `cmd/clinban/show.go`

**Behaviour:**
- Resolve ticket by ID (search active + archive via `store.FindByID`); unknown → `"ticket not found"` + exit 1.
- `store.ReadTicket(path)`.
- Print to stdout:
  - `ID:      <id>`
  - `Status:  <status>`
  - `Type:    <type>`
  - `Title:   <title>`
  - `Tags:    <tag1>, <tag2>` (omit line if Tags is empty)
  - `Created: <RFC3339>`
  - `Updated: <RFC3339>`
  - `[archived]` (only if `inArchive == true`)
  - blank line + body (if body is non-empty)
- Exit 0. No files are modified.

**Done when:** Acceptance criteria §7 show from `01_requirements.md` are met.

**Depends on:** T-08, T-09

---

### T-17 · `clinban new` (interactive)

**Deliverable:** Interactive ticket creation with editor, template, lint, and re-open loop.

**Key file:** `cmd/clinban/new.go` (the interactive path, same file as T-11)

**Implementation notes:**
- `internal/template`: embed `new.md` via `//go:embed new.md`; `New(id int, now time.Time)` executes `text/template` and returns bytes.
- Write rendered template to `os.CreateTemp(store.TicketsDir, ".clinban-*.md")` (temp in `TicketsDir`, not system temp — ensures same-filesystem rename per ADR-3).
- `editor.Open(tmpPath)`.
- Read file; if `title == ""` (unchanged): delete temp, print `"Ticket discarded."`, exit 0.
- Run `lint.Lint` on parsed ticket.
- Compute final filename: `store.TicketPath(id, slug.Slugify(title))`.
- Move temp file to `TicketsDir` (regardless of lint state): `os.Rename(tmpPath, finalPath)`.
- If lint errors: print each; prompt `"Re-open in editor? [y/N] "`; if `y`, `editor.Open(finalPath)`, re-lint; repeat.
- On clean lint or user declines re-open: print `"created: <filename>"`, exit 0.

**Integration test:** Set `EDITOR` to a shell script that appends a valid title and type to the template; assert ticket file appears in TicketsDir with correct content.

**Done when:** Acceptance criteria §7 `new` (interactive) from `01_requirements.md` are met; editor open/lint/re-open loop verified by integration test.

**Depends on:** T-04, T-06, T-07, T-08, T-09

---

### T-18 · `clinban edit`

**Deliverable:** `clinban edit <id>` — open ticket in editor; write back only when parse and lint pass.

**Key file:** `cmd/clinban/edit.go`

**Behaviour:**
- Resolve ticket by ID (search active + archive); unknown → `"ticket not found"` + exit 1.
- `editor.Open(path)` (opens the live ticket file directly).
- On editor close:
  - Re-read file; `ticket.Parse()`. If parse error: print error; prompt `"Re-open in editor? [y/N] "`; if `y` repeat from `editor.Open`; if `n` exit 1.
  - `store.AllIDs()`, `lint.Lint(t, filename, allIDs)`. If errors: print each; prompt `"Re-open in editor? [y/N] "`; if `y` repeat from `editor.Open`; if `n` exit 1.
  - `t.Updated = time.Now()`; `store.WriteTicket(t, path)`.
- Exit 0.

**Integration test:** Set `EDITOR` to a script that modifies the title field; assert `updated` timestamp changes and `clinban lint <id>` passes afterward.

**Done when:** Acceptance criteria §7 edit from `01_requirements.md` are met.

**Depends on:** T-06, T-07, T-08, T-09

---

## Dependency Graph

```
T-01
├── T-02
│   ├── T-06 ─── T-10, T-11, T-15, T-18
│   └── T-08 ─── T-09
│                  └── T-10, T-11, T-12, T-13, T-14, T-15, T-16, T-17, T-18
├── T-03 ──────── T-13
├── T-04 ──────── T-11, T-17
├── T-05 ──────── T-09
└── T-07 ──────── T-17, T-18
```

**Minimum critical path to first working command (`clinban lint`):**
T-01 → T-02 → T-06 → T-08 → T-09 → T-10
