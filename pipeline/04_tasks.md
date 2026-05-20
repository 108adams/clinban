# Developer Tasks
_Produced by: techlead-agent_
_Date: 2026-05-20_
_Status: draft_
_Input: pipeline/03_design.md_

## Task List

### TASK-001: go mod tidy and CI guard (Q-1)
- **Description:** Run `go mod tidy` to promote direct dependencies (`cobra`, `BurntSushi/toml`, `yaml.v3`, `x/term`) from `// indirect` to direct. Document the CI guard command in `CONTRIBUTING.md` or `AGENTS.md`.
- **Module(s):** `go.mod`, `go.sum`
- **Done criteria:**
  - [ ] `go.mod` lists `github.com/BurntSushi/toml`, `github.com/spf13/cobra`, `golang.org/x/term`, `gopkg.in/yaml.v3` without `// indirect`
  - [ ] `go.sum` updated to match
  - [ ] `go mod tidy && git diff --exit-code go.mod go.sum` exits 0 after the commit
  - [ ] CI guard command documented in `AGENTS.md` or `CONTRIBUTING.md`
  - [ ] `go test ./...` passes
- **Depends on:** none
- **Notes:** `github.com/inconshreveable/mousetrap`, `github.com/spf13/pflag`, and `golang.org/x/sys` are genuinely indirect and must stay marked as such.

---

### TASK-002: fsync in WriteTicket (D-1)
- **Description:** Add `tmp.Sync()` before `tmp.Close()` in `store.WriteTicket`. After `os.Rename` succeeds, open the parent directory and call `Sync()` on it to make the directory entry durable. Close the directory fd unconditionally.
- **Module(s):** `internal/store/write.go`
- **Done criteria:**
  - [ ] `tmp.Sync()` is called after `tmp.Chmod` and before `tmp.Close()`; its error is wrapped and returned
  - [ ] After `os.Rename` succeeds, parent directory is opened, `Sync()`'d, and closed (sync error is intentionally swallowed — best-effort; close error is also swallowed)
  - [ ] `go test ./internal/store/...` passes
  - [ ] `go vet ./internal/store/...` clean
- **Depends on:** none
- **Notes:** The parent-dir sync error is conventionally swallowed (same behavior as `git` and `postgres`); wrap and return the `tmp.Sync()` error because that one is in the critical write path. Existing `WriteTicket` tests still pass with no changes required.

---

### TASK-003: Atomic move with collision tests (D-2 + T-3)
- **Description:** Replace the `os.Stat(dest)` + `os.Rename(src, dest)` pattern in both `MoveToArchive` and `MoveToActive` with `os.Link(src, dest)` + `os.Remove(src)`. `os.Link` fails atomically with `EEXIST` if dest exists. Add `TestMoveToArchiveRefusesExistingDestination` and `TestMoveToActiveRefusesExistingDestination` to the store test file.
- **Module(s):** `internal/store/move.go`, `internal/store/store_test.go` (or adjacent test file)
- **Done criteria:**
  - [ ] `os.Stat` + `os.Rename` pattern removed from both functions
  - [ ] `os.Link(src, dest)` used; on `errors.Is(err, fs.ErrExist)`, returns the same "destination already exists" message as before
  - [ ] `os.Remove(src)` follows a successful `os.Link`; its error is wrapped and returned
  - [ ] `TestMoveToArchiveRefusesExistingDestination` exists and passes: pre-create dest, call `MoveToArchive`, assert error is non-nil and source file still exists
  - [ ] `TestMoveToActiveRefusesExistingDestination` exists and passes: same pattern for active direction
  - [ ] `go test ./internal/store/...` passes
  - [ ] `go vet ./internal/store/...` clean
- **Depends on:** none
- **Notes:** Import `io/fs` for `fs.ErrExist`. The existing `MoveToArchiveCreatesDir`, `PreservesFilename`, and `MoveToActive` tests must continue to pass unchanged.

---

### TASK-004: Unit tests for internal/editor (T-1a)
- **Description:** Create `internal/editor/editor_test.go` with three test functions covering the success path, the failure path, and the vi fallback path.
- **Module(s):** `internal/editor/editor_test.go` (new file)
- **Done criteria:**
  - [ ] `TestEditorSuccess`: sets `EDITOR=/bin/true`, calls `Open` with a temp file path, asserts `nil` return
  - [ ] `TestEditorFailure`: sets `EDITOR=/bin/false`, calls `Open`, asserts non-nil error containing `"exit status"`
  - [ ] `TestEditorFallback`: sets `EDITOR=""` and `PATH` to a temp dir (no `vi`), calls `Open`, asserts non-nil error containing `"executable file not found"`
  - [ ] All three tests use `t.Setenv` for environment isolation
  - [ ] `go test ./internal/editor/...` passes
  - [ ] `go vet ./internal/editor/...` clean
- **Depends on:** none
- **Notes:** `/bin/true` and `/bin/false` are available on Linux and macOS; no stub scripts needed. Use `t.TempDir()` for the temp file argument to `Open`.

---

### TASK-005: Unit tests for internal/template (T-1b)
- **Description:** Create `internal/template/template_test.go` with two test functions verifying that `New` returns parseable bytes containing the expected ID and timestamp.
- **Module(s):** `internal/template/template_test.go` (new file)
- **Done criteria:**
  - [ ] `TestNewReturnsParseableTicket`: calls `New(1, time.Now())`, asserts non-empty bytes, passes bytes to `ticket.Parse` and asserts no error
  - [ ] `TestNewContainsIDAndTimestamp`: calls `New(1, fixedTime)` with a fixed `time.Time`, asserts the rendered bytes contain the string form of the ID and the timestamp in RFC3339 format
  - [ ] `go test ./internal/template/...` passes
  - [ ] `go vet ./internal/template/...` clean
- **Depends on:** none
- **Notes:** The parse/execute error branches inside `New` are not exercisable (the embedded template is always valid at compile time); do not attempt to test them. Import `clinban/internal/ticket` for the parse assertion.

---

### TASK-006: GOCOVERDIR wiring for cmd/clinban coverage (T-2)
- **Description:** Wire Go 1.20+ binary coverage so that subprocess tests in `cmd/clinban` contribute to the coverage report. The test binary must be built with `-cover` and `GOCOVERDIR` must be set in the environment when subprocesses are launched.
- **Module(s):** `cmd/clinban/` — `TestMain` function (add or update)
- **Done criteria:**
  - [ ] `TestMain` builds (or locates) the instrumented binary using `-cover`
  - [ ] `GOCOVERDIR` env var is set to a temp directory for all subprocess invocations in the test package
  - [ ] Running `go test -cover ./cmd/clinban/...` produces a coverage percentage substantially higher than 4.5%
  - [ ] `go test ./cmd/clinban/...` passes without `GOCOVERDIR` set (graceful degradation)
  - [ ] `go vet ./cmd/clinban/...` clean
- **Depends on:** none
- **Notes:** Refer to `cmd/go` docs for `GOCOVERDIR` and `go build -cover`. The existing subprocess harness helper should be updated to inject `GOCOVERDIR` into `exec.Command` env rather than rewriting individual tests.

---

### TASK-007: ExitError type and migration of show and move (Q-2)
- **Description:** Create `cmd/clinban/exit.go` with an `ExitError` struct. Update `Execute()` in `root.go` to detect `ExitError` via `errors.As` and call `os.Exit` with the carried code. Convert `show.go` (1 site) and `move.go` (3 sites) from `os.Exit(1)` to returning `ExitError`.
- **Module(s):** `cmd/clinban/exit.go` (new), `cmd/clinban/root.go`, `cmd/clinban/show.go`, `cmd/clinban/move.go`
- **Done criteria:**
  - [ ] `exit.go` defines `ExitError{Code int; Err error}` with `Error() string` and `Unwrap() error` methods
  - [ ] `root.go Execute()` calls `errors.As(err, &exitErr)` and delegates to `os.Exit(exitErr.Code)` when matched; otherwise exits with code 1
  - [ ] `show.go`: the `os.Exit(1)` after "ticket not found" is replaced with `return ExitError{Code: 1, Err: ...}`
  - [ ] `move.go`: all three `os.Exit(1)` calls are replaced with `return ExitError{Code: 1, Err: ...}`
  - [ ] Existing subprocess tests still pass (exit codes unchanged from user perspective)
  - [ ] `go test ./cmd/clinban/...` passes
  - [ ] `go vet ./cmd/clinban/...` clean
- **Depends on:** TASK-001 (go mod tidy ensures the build is clean before structural changes)
- **Notes:** The stderr printing before each `os.Exit` call should remain — only the `os.Exit` call itself is replaced. `PersistentPreRun` in `root.go` also calls `os.Exit(1)` on config-load failure; that site is out of scope for this task (it runs before `Execute()` returns).

---

### TASK-008: Merge zero-timestamp lint rule into ruleRequiredFields (D-3)
- **Description:** Delete `ruleTimestampsNonZero` from `internal/lint/rules.go` and from the rule registration slice. Update `ruleRequiredFields` to emit `"zero timestamp; value was not parseable as RFC3339"` instead of `"required field missing"` for the `Created` and `Updated` zero-value branches. Update any lint tests that assert on timestamp error messages.
- **Module(s):** `internal/lint/rules.go`, `internal/lint/lint.go` (rule registration), `internal/lint/lint_test.go` (or adjacent test file)
- **Done criteria:**
  - [ ] `ruleTimestampsNonZero` function deleted from `rules.go`
  - [ ] `ruleTimestampsNonZero` removed from the rule registration slice in `lint.go`
  - [ ] `ruleRequiredFields` emits `Message: "zero timestamp; value was not parseable as RFC3339"` for `Created` and `Updated` zero checks
  - [ ] A test asserts that a ticket with zero `Created` and `Updated` produces exactly one `LintError` per field (not two)
  - [ ] `go test ./internal/lint/...` passes
  - [ ] `go vet ./internal/lint/...` clean
- **Depends on:** none
- **Notes:** Confirm the rule registration slice location (likely `lint.go:var rules`) before removing the entry. The `ruleTagsNonEmpty` and `ruleIDUnique` rules are unaffected.

---

### TASK-009: Empty-slug fallback in Slugify (D-4)
- **Description:** Add a fallback in `slug.Slugify` so that when the result would be `""` (all-non-ASCII or all-punctuation input), `"ticket"` is returned instead. Add a corresponding test case.
- **Module(s):** `internal/slug/slug.go`, `internal/slug/slug_test.go`
- **Done criteria:**
  - [ ] `Slugify` returns `"ticket"` for input `"你好世界"` (all CJK)
  - [ ] `Slugify` returns `"ticket"` for input `"!!! ??? ..."` (all punctuation)
  - [ ] `Slugify` still returns `"hello-world"` for `"Hello World"` (regression check)
  - [ ] `Slugify` still returns `""` for `""` ... actually: `Slugify("")` returns `"ticket"` (empty input)
  - [ ] Test case added to `slug_test.go` for all-non-ASCII input
  - [ ] `go test ./internal/slug/...` passes
  - [ ] `go vet ./internal/slug/...` clean
- **Depends on:** none
- **Notes:** The one-line fix: after `return strings.Join(parts, "-")`, extract to a variable; if the variable is `""`, return `"ticket"`. Alternatively add `if result == "" { return "ticket" }` before the final return.

---

### TASK-010: Fix formatRecord rune math in list.go (Q-4)
- **Description:** Replace `prefixLen := len(prefix)` with `prefixLen := utf8.RuneCountInString(prefix)` on `list.go:165`. Add the `unicode/utf8` import.
- **Module(s):** `cmd/clinban/list.go`
- **Done criteria:**
  - [ ] Line 165 uses `utf8.RuneCountInString(prefix)` not `len(prefix)`
  - [ ] `"unicode/utf8"` added to imports
  - [ ] `go test ./cmd/clinban/...` passes
  - [ ] `go vet ./cmd/clinban/...` clean
  - [ ] `gofmt -l cmd/clinban/list.go` reports no diff
- **Depends on:** none
- **Notes:** The current behavior is correct for ASCII-only prefix content (id, status, type are all ASCII). This is a future-proofing change. No test for the rune-vs-byte distinction is required unless a non-ASCII prefix format is introduced.

---

## Dependency Order

Tasks 1–9 and 10 are nearly all independent. The single dependency is:

```
TASK-001 (go mod tidy)
    └── TASK-007 (ExitError + show/move migration)
```

Recommended execution order for a single developer (parallelizable pairs shown on the same line):

```
1. TASK-001                        # unblocks TASK-007; fast, commit immediately
2. TASK-002, TASK-003              # data integrity fixes; independent
3. TASK-004, TASK-005              # new test files; independent
4. TASK-006                        # coverage wiring; independent
5. TASK-007                        # depends on TASK-001
6. TASK-008, TASK-009, TASK-010    # P2 items; independent
```

All tasks except TASK-007 can be done in any order relative to each other.
