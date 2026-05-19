# Design: Clinban

Input: `pipeline/02_architecture.md`
Output: module structure, interface contracts, test strategy for developer use.

---

## Directory Layout

```
clinban/
  cmd/clinban/
    main.go         ← entry point; calls root.Execute()
    root.go         ← cobra root command; PersistentPreRun loads config
    new.go          ← clinban new (interactive + --no-interactive)
    list.go         ← clinban list
    show.go         ← clinban show (read-only)
    edit.go         ← clinban edit (editor-based)
    move.go         ← clinban move
    archive.go      ← clinban archive
    lint.go         ← clinban lint
    register.go     ← clinban register
  internal/
    config/
      config.go
    ticket/
      ticket.go     ← Ticket struct, Parse, Marshal
      status.go     ← Status type and constants
      tickettype.go ← Type type and constants
    store/
      store.go      ← Store struct, New
      scan.go       ← NextID, FindByID, AllIDs, ListActive
      write.go      ← WriteTicket (atomic), ReadTicket
      move.go       ← MoveToArchive, MoveToActive
    lint/
      lint.go       ← LintError, Lint
      rules.go      ← one function per rule
    fsm/
      fsm.go        ← transitions table, ValidateTransition
    editor/
      editor.go     ← Open
    slug/
      slug.go       ← Slugify
    template/
      template.go   ← embedded template access via embed.FS
      new.md        ← the ticket template file (embedded)
  go.mod
  go.sum
```

---

## Package Interfaces

### `internal/ticket`

```go
type Status string
const (
    StatusBacklog    Status = "backlog"
    StatusInProgress Status = "in-progress"
    StatusBlocked    Status = "blocked"
    StatusDone       Status = "done"
)
func (s Status) Valid() bool

type Type string
const (
    TypeBug     Type = "bug"
    TypeTask    Type = "task"
    TypeFeature Type = "feature"
    TypeSpike   Type = "spike"
)
func (t Type) Valid() bool

type Ticket struct {
    ID      string    `yaml:"id"`
    Status  Status    `yaml:"status"`
    Type    Type      `yaml:"type"`
    Title   string    `yaml:"title"`
    Tags    []string  `yaml:"tags"`
    Created time.Time `yaml:"created"`
    Updated time.Time `yaml:"updated"`
    Body    string    // markdown body; not part of YAML frontmatter
}

// Parse splits on --- fences, decodes YAML frontmatter, captures body.
// Returns error if frontmatter is missing or malformed YAML.
func Parse(content []byte) (*Ticket, error)

// Marshal serialises the ticket back to --- fenced YAML + body.
func Marshal(t *Ticket) ([]byte, error)
```

**Constraints:**
- `Parse` and `Marshal` are semantic inverses: `parse → marshal → parse` yields equal field values and body. Byte-level equality is not guaranteed (yaml.v3 may reformat whitespace, quoting, or key order).
- Tags field serialises as `tags: []` (not omitted) when empty, for schema consistency. The `yaml:"tags"` tag (no `omitempty`) ensures the field is always emitted.

---

### `internal/fsm`

```go
// ValidateTransition returns nil if the transition from→to is in the
// valid transitions table. Returns a descriptive error listing valid
// next statuses if the transition is forbidden.
func ValidateTransition(from, to ticket.Status) error
```

**Valid transitions (all others are errors):**

| From | To |
|---|---|
| backlog | in-progress |
| backlog | blocked |
| in-progress | blocked |
| in-progress | done |
| blocked | in-progress |
| done | backlog |

---

### `internal/slug`

```go
// Slugify returns the first 5 words of title, lowercased,
// joined with hyphens, all non-alphanumeric characters stripped.
// Short titles (< 5 words) use all words.
func Slugify(title string) string
```

---

### `internal/config`

```go
type Config struct {
    TicketsDir string `toml:"tickets_dir"`
    ArchiveDir string `toml:"archive_dir"`
}

// Load reads .clinban from projectRoot.
// If the file is absent, returns defaults silently.
// If the file exists but is malformed TOML, returns error.
// Defaults: TicketsDir = projectRoot, ArchiveDir = projectRoot/archive.
func Load(projectRoot string) (*Config, error)
```

---

### `internal/lint`

```go
type LintError struct {
    File    string // filename only, not full path
    Field   string
    Message string
}

// String returns the canonical one-line format:
// "0042-fix-login-timeout.md: field 'type': invalid value"
func (e LintError) String() string

// Lint runs all 7 rules against the ticket.
// filename is the base filename (for rule 4 and error output).
// allIDs is the full list of IDs across active + archive (for rule 7).
// Returns an empty slice (never nil) when the ticket is valid.
func Lint(t *ticket.Ticket, filename string, allIDs []string) []LintError
```

**Rules (executed in order):**

| # | Field | Check |
|---|---|---|
| 1 | all required | id, status, title, type, created, updated are non-zero |
| 2 | status | value is one of the valid Status constants |
| 3 | type | value is one of the valid Type constants |
| 4 | id | numeric portion matches the id field value |
| 5 | created, updated | are non-zero `time.Time` values; a zero value indicates the source string was unparseable by yaml.v3 |
| 6 | tags | if present, all elements are non-empty strings (YAML decode coerces type; lint verifies no zero-value entries) |
| 7 | id | is unique across all IDs in allIDs |

---

### `internal/editor`

```go
// Open launches $EDITOR (fallback: "vi") with path as argument.
// Stdin, Stdout, and Stderr are inherited from the parent process.
// Returns error if the editor process exits non-zero.
func Open(path string) error
```

---

### `internal/store`

```go
type Store struct {
    TicketsDir string
    ArchiveDir string
}

func New(cfg *config.Config) *Store

// NextID scans TicketsDir and ArchiveDir for the highest numeric
// filename prefix, returns that value + 1. Returns 1 if no tickets exist.
func (s *Store) NextID() (int, error)

// FindByID locates a ticket file by its 4-digit ID prefix.
// Searches TicketsDir first, then ArchiveDir.
// Returns the full path, whether it is in the archive, and any error.
// Returns ("", false, ErrNotFound) if the ID does not exist.
func (s *Store) FindByID(id string) (path string, inArchive bool, err error)

// AllIDs returns every ID string found in both directories.
// Used by lint for uniqueness checking.
func (s *Store) AllIDs() ([]string, error)

// Record pairs a parsed Ticket with its filesystem location.
type Record struct {
	Ticket    *ticket.Ticket
	Path      string
	InArchive bool
}

// ListActive returns all tickets in TicketsDir as Records.
// Only files matching [0-9]{4}-*.md are parsed; other .md files are skipped.
// Returns an empty slice (never nil) if the directory is empty.
func (s *Store) ListActive() ([]Record, error)

// ListArchive returns all tickets in ArchiveDir as Records.
// Only files matching [0-9]{4}-*.md are parsed; other .md files are skipped.
// Returns an empty slice (never nil) if the directory is empty.
func (s *Store) ListArchive() ([]Record, error)

// ReadTicket reads and parses the file at path.
func (s *Store) ReadTicket(path string) (*ticket.Ticket, error)

// WriteTicket serialises t and writes it atomically to path.
// Atomic: creates a temp file via os.CreateTemp in the same directory,
// writes, sets permissions to 0600, closes, then renames.
// Does not modify any fields on t; caller must set t.Updated before calling.
func (s *Store) WriteTicket(t *ticket.Ticket, path string) error

// TicketPath returns the canonical path for a ticket in TicketsDir.
// Format: <TicketsDir>/<id-zero-padded-4>-<slug>.md
func (s *Store) TicketPath(id int, slug string) string

// ActivePath returns the path a ticket would have in TicketsDir,
// preserving its existing filename. Used when moving a ticket back from archive.
func (s *Store) ActivePath(archivePath string) string

// MoveToArchive moves the file at path into ArchiveDir.
// Creates ArchiveDir if it does not exist.
// Returns an error if a file with the same name already exists in ArchiveDir.
// Returns the new path.
func (s *Store) MoveToArchive(path string) (string, error)

// MoveToActive moves the file at path from ArchiveDir into TicketsDir.
// Returns an error if a file with the same name already exists in TicketsDir.
// Returns the new path.
func (s *Store) MoveToActive(path string) (string, error)

// ErrNotFound is returned by FindByID when no ticket matches.
var ErrNotFound = errors.New("ticket not found")
```

---

### `internal/template`

```go
// New returns the content of the embedded ticket template,
// with id, created, and updated pre-filled.
func New(id int, now time.Time) ([]byte, error)
```

Template file (`internal/template/new.md`):

```markdown
---
id: "{{.ID}}"
status: "backlog"
type: "task"
title: ""
tags: []
created: "{{.Created}}"
updated: "{{.Updated}}"
---

```

---

### `cmd/clinban` — command conventions

- All errors print to `stderr`; all normal output prints to `stdout`.
- Exit code 0 on success, 1 on any error.
- `root.go` resolves the project root (directory containing `.clinban`, or cwd if absent),
  loads config, and stores `*store.Store` on cobra's context for all subcommands.
- `rootCmd` sets `SilenceErrors: true` and `SilenceUsage: true`. All error output is
  owned by the command handler or by `main`; Cobra never prints errors directly.
- Command handlers call `os.Exit(1)` directly after printing a user-facing error message.
  They return an `error` only for unexpected failures (I/O errors, etc.) that `main` should
  surface. This keeps the two paths distinct and prevents double-printing.
- Interactive prompts (`y/N`) read from `os.Stdin`; default on bare Enter is `N`.

---

## Test Strategy

### Critical paths — tests required before any code ships

1. **`internal/lint`** — every rule with a valid and an invalid input. This is the automata safety net.
2. **`internal/fsm`** — all 16 transition combinations asserted explicitly (6 valid, 10 invalid).
3. **`internal/ticket` Parse/Marshal** — round-trip test; silent field drop would corrupt every ticket.

### Unit tests (no filesystem)

| Package | What to test |
|---|---|
| `ticket` | Parse round-trip; Marshal output; malformed frontmatter returns error; empty body handled |
| `fsm` | All 16 combinations; error message includes valid next statuses |
| `lint` | Each rule isolated; clean ticket returns empty slice (not nil) |
| `slug` | Short title (<5 words); special chars; unicode; empty string |
| `config` | Absent file → defaults; malformed TOML → error; values override defaults |

### Integration tests (real filesystem via `t.TempDir()`)

| Package | What to test |
|---|---|
| `store` | NextID with 0, 1, N tickets; FindByID in active and archive; WriteTicket atomic (verify temp file gone); MoveToArchive creates dir if absent; MoveToArchive and MoveToActive return error if destination exists; ListActive skips non-ticket .md files |
| CLI | Happy path smoke test for each command using `--no-interactive` or scripted `$EDITOR` |

### Not unit-tested

`internal/editor` — spawns an OS process; verified in `clinban new` integration test by setting
`$EDITOR` to a shell script that writes a valid ticket and exits 0.
