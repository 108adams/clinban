# Implementation Design
_Produced by: techlead-agent_
_Date: 2026-05-20_
_Status: draft_
_Input: pipeline/quality-audit.md (no 02_architecture.md — hardening sprint only)_

## Scope note

This is a hardening sprint. Module boundaries are stable and unchanged. Every
change is a targeted fix or addition within existing packages. No new packages
are introduced.

---

## Module Structure

### internal/store — write.go (D-1)

**Files:**
- `internal/store/write.go` — add fsync of temp file and parent directory to `WriteTicket`

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|---------------|
| `WriteTicket` | `(t *ticket.Ticket, path string) → error` | Serialize ticket to path atomically; after this change, also fsync-durable |

**Interface contract:**
- Accepts: a valid `*ticket.Ticket` and an absolute target path
- Returns: `nil` on success; wrapped error on marshal failure, temp-file I/O failure, chmod failure, sync failure, or rename failure
- Errors: all wrapped with prefix `"store: write ticket: ..."`
- Change: `tmp.Sync()` is called after `tmp.Chmod` and before `tmp.Close()`; after `os.Rename` succeeds, the parent directory is opened and `Sync()`'d to make the directory entry durable; the directory fd is always closed

**Implementation notes:**
```
// After chmod, before close:
if err := tmp.Sync(); err != nil { ... }

// After os.Rename succeeds:
dir, err := os.Open(filepath.Dir(path))
if err == nil {
    _ = dir.Sync()
    _ = dir.Close()
}
```

---

### internal/store — move.go (D-2)

**Files:**
- `internal/store/move.go` — replace TOCTOU `os.Stat` + `os.Rename` pattern with `os.Link` + `os.Remove`

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|---------------|
| `MoveToArchive` | `(path string) → (string, error)` | Move ticket file to archive atomically, refusing collision |
| `MoveToActive` | `(path string) → (string, error)` | Move ticket file to active dir atomically, refusing collision |

**Interface contract:**
- Accepts: source path of an existing ticket file
- Returns: new path on success; error if destination already exists or any I/O fails
- Errors: `"store: move to archive: link: ..."` / `"store: move to active: link: ..."` on collision or I/O failure; `"store: move to archive: remove: ..."` / `"store: move to active: remove: ..."` on source removal failure
- Change: `os.Stat(dest)` + `os.Rename(src, dest)` is replaced with `os.Link(src, dest)` followed by `os.Remove(src)`; `os.Link` returns `EEXIST` atomically if dest exists, which becomes the collision error; no silent overwrite is possible

**Implementation notes:**
- `os.Link` is POSIX-standard and available on all target platforms (Linux, macOS)
- If `os.Link` succeeds but `os.Remove` fails, the source file still exists alongside the new hard link; the caller gets an error and can retry
- The existing error message text "destination already exists: ..." is preserved by checking `errors.Is(err, fs.ErrExist)` after `os.Link`

---

### internal/lint — rules.go (D-3)

**Files:**
- `internal/lint/rules.go` — delete `ruleTimestampsNonZero`; update `ruleRequiredFields` to emit the precise timestamp message for zero `Created`/`Updated`

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|---------------|
| `ruleRequiredFields` | `(t *ticket.Ticket, filename string, _ []string) → []LintError` | Check all required fields; now also emits precise RFC3339 message for zero timestamps |
| `ruleTimestampsNonZero` | _deleted_ | Was rule 5; merged into rule 1 |

**Interface contract:**
- Accepts: parsed ticket, filename, allIDs slice (unused by this rule)
- Returns: one `LintError` per missing or zero-timestamp field; never two errors for the same field
- Errors: for `Created` zero value: `Message: "zero timestamp; value was not parseable as RFC3339"`; for `Updated` zero value: same message; replaces the previous `"required field missing"` message for timestamp fields
- Change: `ruleTimestampsNonZero` is removed from the rule list in `lint.go`; `ruleRequiredFields` replaces its `t.Created.IsZero()` branch with the precise message; same for `t.Updated`

**Implementation notes:**
- The rule registration slice in `lint.go` (or wherever rules are wired) must have `ruleTimestampsNonZero` removed; confirm the variable name at wiring site before deleting
- Lint tests that assert on timestamp error messages must be updated to expect exactly one error per zero field with the precise message

---

### internal/slug — slug.go (D-4)

**Files:**
- `internal/slug/slug.go` — add fallback return value `"ticket"` when `Slugify` produces an empty string

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|---------------|
| `Slugify` | `(title string) → string` | Convert title to a filesystem-safe slug; now returns `"ticket"` instead of `""` for all-non-ASCII / all-punctuation input |

**Interface contract:**
- Accepts: any string
- Returns: a non-empty slug string; `"ticket"` when no ASCII-alphanumeric characters survive stripping
- Errors: none (pure function)
- Change: one line added at the end of the function: `if result == ""; return "ticket"`

---

### internal/editor — editor_test.go (T-1a)

**Files:**
- `internal/editor/editor_test.go` — new file; 3 test functions

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|---------------|
| `TestEditorSuccess` | `(t *testing.T)` | EDITOR=/bin/true → Open returns nil |
| `TestEditorFailure` | `(t *testing.T)` | EDITOR=/bin/false → Open returns error containing "exit status" |
| `TestEditorFallback` | `(t *testing.T)` | EDITOR="" + nonexistent path → error contains "executable file not found" (proves vi fallback path is reached) |

**Interface contract (test perspective):**
- Each test sets `t.Setenv("EDITOR", ...)` to isolate environment
- Tests create a `t.TempDir()` file to pass as the path argument
- `TestEditorFallback` overrides `PATH` so `vi` is not found, proving the fallback branch executes

---

### internal/template — template_test.go (T-1b)

**Files:**
- `internal/template/template_test.go` — new file; 2 test functions

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|---------------|
| `TestNewReturnsParseableTicket` | `(t *testing.T)` | `New(1, someTime)` returns non-empty bytes that `ticket.Parse` accepts without error |
| `TestNewContainsIDAndTimestamp` | `(t *testing.T)` | rendered bytes contain the ID (`"0001"` or `"1"`) and timestamp in expected RFC3339 format |

**Interface contract (test perspective):**
- Parse/execute error branches are defensive (embedded template is always valid at compile time) and are not directly exercisable; they are not tested
- Tests use a fixed `time.Time` value for deterministic assertions

---

### cmd/clinban — exit.go (Q-2)

**Files:**
- `cmd/clinban/exit.go` — new file; defines `ExitError` type

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|---------------|
| `ExitError` | `struct{ Code int; Err error }` | Typed error carrying a process exit code |
| `(e ExitError) Error()` | `() → string` | Implement `error`; delegates to `e.Err.Error()` |
| `(e ExitError) Unwrap()` | `() → error` | Implement `errors.Unwrap`; returns `e.Err` |

**Interface contract:**
- `ExitError.Code` is the intended `os.Exit` code (1 for user-visible failures)
- `Execute()` in `root.go` checks `errors.As(err, &exitErr)` after `rootCmd.Execute()` and calls `os.Exit(exitErr.Code)` when matched; otherwise exits with code 1
- Initial migration scope: `show.go` (1 `os.Exit` site) and `move.go` (3 `os.Exit` sites) are converted to `return ExitError{Code: 1, Err: err}` pattern

**Migration pattern for show.go:**
```go
// Before:
fmt.Fprintln(os.Stderr, "ticket not found")
os.Exit(1)

// After:
fmt.Fprintln(os.Stderr, "ticket not found")
return ExitError{Code: 1, Err: fmt.Errorf("ticket not found")}
```

---

### cmd/clinban — GOCOVERDIR wiring (T-2)

**Files:**
- `cmd/clinban/main_test.go` or equivalent TestMain file — add `-cover` build flag and `GOCOVERDIR` env var to subprocess invocations

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|---------------|
| `TestMain` | `(m *testing.M)` | Build the test binary with `-cover`; set `GOCOVERDIR` when spawning subprocess tests |

**Interface contract:**
- `GOCOVERDIR` must point to a directory that exists for the duration of the test run
- Coverage from subprocess execution is aggregated into the same coverage report as in-process tests
- No individual test function signatures change

---

### cmd/clinban — list.go (Q-4)

**Files:**
- `cmd/clinban/list.go` — fix `formatRecord` to use `utf8.RuneCountInString(prefix)` instead of `len(prefix)`

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|---------------|
| `formatRecord` | `(r store.Record, width int) → string` | Format one ticket row for terminal display; after fix, prefix length is computed in runes not bytes |

**Interface contract:**
- Accepts: a `store.Record` and terminal width in columns
- Returns: a string of at most `width` runes
- Change: line 165 `prefixLen := len(prefix)` becomes `prefixLen := utf8.RuneCountInString(prefix)`; add `"unicode/utf8"` to imports

---

### go.mod (Q-1)

**Files:**
- `go.mod` — promote direct dependencies from `// indirect` to direct

**Interface contract:**
- After `go mod tidy`: `github.com/BurntSushi/toml`, `github.com/spf13/cobra`, `golang.org/x/term`, and `gopkg.in/yaml.v3` appear without `// indirect`
- `github.com/inconshreveable/mousetrap`, `github.com/spf13/pflag`, and `golang.org/x/sys` remain `// indirect`
- CI guard command: `go mod tidy && git diff --exit-code go.mod go.sum`

---

## Inter-Component Communication

| From | To | Method | Data |
|------|----|--------|------|
| `cmd/clinban/root.go Execute()` | `cmd/clinban/exit.go ExitError` | `errors.As` check | `ExitError{Code, Err}` |
| `cmd/clinban/show.go runShow` | `ExitError` | return value | exit code 1 + wrapped error |
| `cmd/clinban/move.go runMove` | `ExitError` | return value | exit code 1 + wrapped error |
| `internal/lint Run()` | `ruleRequiredFields` | direct function call | `*ticket.Ticket`, filename, allIDs |
| `internal/store WriteTicket` | OS filesystem | syscall | `tmp.Sync()`, parent dir `Sync()` |
| `internal/store MoveToArchive/MoveToActive` | OS filesystem | syscall | `os.Link`, `os.Remove` |

---

## Test Strategy

**Unit tests (per module):**
- `internal/store`: existing tests cover `WriteTicket`; add `TestMoveToArchiveRefusesExistingDestination` and `TestMoveToActiveRefusesExistingDestination` to pin D-2 fix
- `internal/lint`: update existing zero-timestamp test to assert exactly one error per field; verify `ruleTimestampsNonZero` is gone
- `internal/slug`: add `TestSlugifyAllNonASCII` asserting `Slugify("你好世界") == "ticket"`
- `internal/editor`: 3 new test functions in `editor_test.go`
- `internal/template`: 2 new test functions in `template_test.go`
- `cmd/clinban`: existing subprocess tests remain; add GOCOVERDIR wiring

**Critical paths (must be tested before first ship):**
1. `store.MoveToArchive` and `store.MoveToActive` with a pre-existing destination file — `TestMoveToArchiveRefusesExistingDestination` and `TestMoveToActiveRefusesExistingDestination` must pass and demonstrate `os.Link`-based atomicity
2. `lint.Run` on a ticket with zero `Created` and `Updated` — after rule merge, exactly one `LintError` per field (not two); message is `"zero timestamp; value was not parseable as RFC3339"`
3. `editor.Open` with `EDITOR=/bin/true` returns `nil`; with `EDITOR=/bin/false` returns a non-nil error containing `"exit status"`

**Integration tests:**
- Existing `cmd/clinban` subprocess tests cover end-to-end command behavior; no new integration tests required beyond GOCOVERDIR wiring to make coverage metrics accurate

---

## Resolved Architecture Questions

| Question (from audit) | Decision | Rationale |
|-----------------------|----------|-----------|
| Use `os.Link`+`os.Remove` vs `unix.Renameat2(RENAME_NOREPLACE)` for atomic move | `os.Link`+`os.Remove` | POSIX-standard, no CGo, no build constraints needed; `Renameat2` is Linux-only |
| Merge D-3 timestamp rule into rule 1 or add a skip guard in rule 5 | Delete rule 5, emit precise message from rule 1 | One rule, one message per field; simpler, eliminates the duplicate entirely |
| ExitError scope — migrate all 22 `os.Exit` sites now or incrementally | Incremental: `show.go` (1 site) and `move.go` (3 sites) in this sprint | Reduces risk; remaining commands can be migrated per subsequent sprint |
| GOCOVERDIR wiring location | `TestMain` in `cmd/clinban` test package | Go 1.20+ standard pattern; no individual test changes required |
