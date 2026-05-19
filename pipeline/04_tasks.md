# Developer Tasks
_Produced by: techlead-agent_
_Date: 2026-05-19_
_Status: draft_
_Input: pipeline/03_design.md_

## Task List

---

## T-01: Update config.defaults() and fix config_test.go

**Status:** todo
**Depends on:** none
**Files:** `internal/config/config.go`, `internal/config/config_test.go`

### What to do

In `config.go`, change the `defaults()` function so that `TicketsDir` is `filepath.Join(projectRoot, "tickets")` and `ArchiveDir` is `filepath.Join(projectRoot, "tickets", "archive")`. The function signature and callers are unchanged; only the two string values inside the returned struct change.

In `config_test.go`, update every test that asserts on the old default paths. Three tests are affected:

- `TestLoad_AbsentFile`: change the expected `TicketsDir` from `dir` to `filepath.Join(dir, "tickets")` and the expected `ArchiveDir` from `filepath.Join(dir, "archive")` to `filepath.Join(dir, "tickets", "archive")`.
- `TestLoad_EmptyTOML`: same changes as `TestLoad_AbsentFile`.
- `TestLoad_PartialConfig_ArchiveDirOnly`: this test sets only `archive_dir` in the config file, so `TicketsDir` falls back to the default. Change the expected `wantTicketsDir` from `dir` to `filepath.Join(dir, "tickets")`.

No other tests in `config_test.go` assert on default paths and no changes to those tests are needed.

### Done criteria
- [ ] `defaults()` returns `projectRoot/tickets` for `TicketsDir` and `projectRoot/tickets/archive` for `ArchiveDir`
- [ ] `GOCACHE=/tmp/go-trello-gocache go test ./internal/config/...` passes with no failures
- [ ] `go vet ./internal/config/...` reports no issues

---

## T-02: Migrate CLI integration test infrastructure for new default layout

**Status:** todo
**Depends on:** T-01
**Files:** `cmd/clinban/lint_test.go`, `cmd/clinban/list_test.go`, `cmd/clinban/new_test.go`, `cmd/clinban/move_test.go`, `cmd/clinban/show_test.go`, `cmd/clinban/archive_test.go`, `cmd/clinban/edit_test.go`, `cmd/clinban/register_test.go`

### What to do

Add a `setupWorkDir` helper function to `cmd/clinban/lint_test.go`, alongside the existing `buildBinary`, `writeTicket`, and `projectRoot` helpers. The function must:

1. Call `t.TempDir()` to create `root`
2. Call `os.MkdirAll(filepath.Join(root, "tickets", "archive"), 0o755)` to create both subdirectories in one call; call `t.Fatal` on error
3. Return `root`, `filepath.Join(root, "tickets")`, and `filepath.Join(root, "tickets", "archive")`

```go
func setupWorkDir(t *testing.T) (root, ticketsDir, archiveDir string) {
    t.Helper()
    root = t.TempDir()
    ticketsDir = filepath.Join(root, "tickets")
    archiveDir = filepath.Join(ticketsDir, "archive")
    if err := os.MkdirAll(archiveDir, 0o755); err != nil {
        t.Fatalf("setupWorkDir: %v", err)
    }
    return root, ticketsDir, archiveDir
}
```

Then update every test in all the listed files that currently does `dir := t.TempDir()` followed by ticket writes and archive path assertions. The migration pattern is:

- Replace `dir := t.TempDir()` with `root, ticketsDir, archiveDir := setupWorkDir(t)`
- Replace `writeTicket(t, dir, ...)` with `writeTicket(t, ticketsDir, ...)`
- Replace `cmd.Dir = dir` (or equivalent working directory assignment) with `cmd.Dir = root`
- Replace `filepath.Join(dir, "archive", filename)` with `filepath.Join(archiveDir, filename)`

Work through each test file systematically. Some test files may use a shared `dir` variable across multiple tests via a package-level setup or a `runXxx` helper; trace every usage of `dir` to ensure the correct replacement is applied. The `cmd.Dir` field must always be `root` so that `findProjectRoot()` in the binary resolves the project root correctly, and `config.defaults()` then derives `tickets/` and `tickets/archive/` under it.

### Done criteria
- [ ] `setupWorkDir` is defined in `cmd/clinban/lint_test.go` and returns all three paths as guaranteed-existing directories
- [ ] No test in `cmd/clinban/` uses `dir := t.TempDir()` as the sole directory for both ticket writes and the command working directory
- [ ] `GOCACHE=/tmp/go-trello-gocache go test ./cmd/clinban/...` passes with no failures (excluding any tests for the not-yet-implemented `init` command)
- [ ] `go vet ./cmd/clinban/...` reports no issues

---

## T-03: Write init_test.go (RED)

**Status:** todo
**Depends on:** T-02
**Files:** `cmd/clinban/init_test.go` (new file)

### What to do

Create `cmd/clinban/init_test.go` with five integration test functions. All tests compile and run the binary produced by `buildBinary(t)`, run `clinban init` (with varying flags and preconditions), and assert on exit code, stdout, and stderr.

Each test must follow the same structural pattern as other integration tests in the package: build the binary once, set up a temp working directory, run the command with `exec.Command`, capture stdout/stderr/exit code, and assert.

The five tests are:

**TestInitFreshDirectory** — Create a clean temp directory (just the root, no subdirectories). Run `clinban init` with no flags. Assert exit code is 0. Assert stdout contains `"created: tickets/"`, `"created: tickets/archive/"`, and `"created: .clinban"`. Optionally verify that all three artifacts now exist on disk.

**TestInitAlreadyExists_NoForce** — Manually create all three artifacts in the temp directory (`tickets/`, `tickets/archive/`, `.clinban`). Run `clinban init` with no flags. Assert exit code is 1. Assert stderr contains text identifying each of the three artifacts as already existing.

**TestInitAlreadyExists_WithForce** — Same setup as above (all three artifacts exist). Run `clinban init --force`. Assert exit code is 1. Assert stderr contains `"already fully initialized"`.

**TestInitPartial_DirsExist_NoConfig_Force** — Manually create `tickets/` and `tickets/archive/` but do not create `.clinban`. Run `clinban init --force`. Assert exit code is 0. Assert stdout contains `"created: .clinban"`. Assert stdout does NOT contain `"created: tickets/"` or `"created: tickets/archive/"`.

**TestInitPartial_ConfigExists_NoDirs_Force** — Manually create `.clinban` (with minimal valid content) but do not create `tickets/` or `tickets/archive/`. Run `clinban init --force`. Assert exit code is 0. Assert stdout contains `"created: tickets/"` and `"created: tickets/archive/"`. Assert stdout does NOT contain `"created: .clinban"`.

For the working directory in each test, use a plain `t.TempDir()` call (not `setupWorkDir`) because these tests are explicitly setting up their own preconditions and must control which directories exist. Set `cmd.Dir` to this temp root.

At the time this task is done, the tests will fail with something like `unknown command "init"` or build failure because `init.go` does not exist yet. The file must compile cleanly (no syntax errors), and the failure must be a runtime test assertion failure, not a compilation error.

### Done criteria
- [ ] `cmd/clinban/init_test.go` exists and compiles as part of `package main` in `cmd/clinban/`
- [ ] The file contains exactly the five test functions listed above
- [ ] `GOCACHE=/tmp/go-trello-gocache go test -run TestInit ./cmd/clinban/...` compiles and runs; tests fail (not crash during compilation)
- [ ] `go vet ./cmd/clinban/...` reports no issues

---

## T-04: Implement clinban init command (GREEN)

**Status:** todo
**Depends on:** T-03
**Files:** `cmd/clinban/init.go` (new file)

### What to do

Create `cmd/clinban/init.go` in `package main`. The file must define `initFlags`, the `newInitCmd` constructor, and the `runInit` function. Register `initCmd` with `rootCmd` via an `init()` function.

**Struct and constructor:**

Define `initFlags` with three fields: `ticketsDir string`, `archiveDir string`, `force bool`. In `newInitCmd()`, create a `*cobra.Command` with `Use: "init"`, attach a no-op `PersistentPreRun` to bypass root startup, bind the three flags (`--tickets-dir` defaulting to `"tickets"`, `--archive-dir` defaulting to `""`, `--force` defaulting to `false`), and set `RunE` to parse the flags into an `initFlags` value and call `runInit`.

**`runInit` implementation — follow this exact algorithm:**

1. If `flags.archiveDir` is empty, set it to `filepath.Join(flags.ticketsDir, "archive")`.
2. Call `os.Getwd()`. Return an error if it fails.
3. Resolve absolute paths. For `ticketsDir` and `archiveDir`: if the value is not already absolute (`!filepath.IsAbs(...)`), join it with cwd. The config path is always `filepath.Join(cwd, ".clinban")`.
4. Stat all three resolved absolute paths. Record which ones exist (store as a map or three booleans).
5. If `--force` is not set and any artifact exists: for each existing artifact, print `"already exists: <relative-name>"` to stderr (use the original flag value or display name like `"tickets/"`, `"tickets/archive/"`, `".clinban"`). Print `"re-run with --force to create missing items"` to stderr. Return a non-nil error.
6. If `--force` is set and all three artifacts exist: print `"already fully initialized"` to stderr. Return a non-nil error.
7. For each missing artifact in write order (`tickets/` first, then `tickets/archive/`, then `.clinban`):
   - For directories: call `os.Mkdir(absPath, 0o755)`. If it returns an error, return that error.
   - For `.clinban`: call `os.WriteFile` with content `fmt.Sprintf("tickets_dir = %q\narchive_dir = %q\n", flags.ticketsDir, flags.archiveDir)` and mode `0o600`. If it returns an error, return that error.
   - On success: print `"created: <relative-name>"` to stdout.

**Registration:**

```go
func init() {
    rootCmd.AddCommand(newInitCmd())
}
```

After implementing, verify:
- `clinban --help` output includes `init` in the list of available commands
- `clinban init --help` shows the three flags
- All five tests from T-03 pass

### Done criteria
- [ ] `cmd/clinban/init.go` exists and compiles as part of `package main`
- [ ] `GOCACHE=/tmp/go-trello-gocache go test -run TestInit ./cmd/clinban/...` — all five tests pass
- [ ] `GOCACHE=/tmp/go-trello-gocache go test ./...` — full test suite passes
- [ ] `go vet ./...` reports no issues
- [ ] `gofmt -l cmd/clinban/init.go` reports no diff (file is properly formatted)
- [ ] `clinban --help` lists `init` as an available command

---

## Dependency Order

```
T-01  (config defaults)
  └── T-02  (test infrastructure migration)
        └── T-03  (init_test.go — RED)
              └── T-04  (init.go — GREEN)
```

T-01 must be complete before T-02 because the migrated tests will run against the binary compiled from the updated codebase; if `config.defaults()` still returns the old paths, tests that assert on directory structure will fail. T-03 must be complete before T-04 so that the test file defines the contract that T-04 is verified against. Do not merge T-04 without all prior tasks passing.
