# Design: clinban init
_Produced by: techlead-agent_
_Date: 2026-05-19_
_Status: draft_
_Input: pipeline/02_architecture.md_

## Module Structure

### internal/config — defaults update

**Files:**
- `internal/config/config.go` — update `defaults()` to return `tickets/` and `tickets/archive/` subdirectories

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|----------------|
| `defaults` | `(projectRoot string) → *Config` | Returns canonical default paths; **change**: `TicketsDir = filepath.Join(projectRoot, "tickets")`, `ArchiveDir = filepath.Join(projectRoot, "tickets", "archive")` |

**Interface contract:**
- Accepts: `projectRoot` — an absolute path to the project root directory
- Returns: `*Config` with `TicketsDir` and `ArchiveDir` set to absolute paths under `projectRoot`
- Errors: none (pure construction)

---

### internal/config — test update

**Files:**
- `internal/config/config_test.go` — update two tests that assert on the old default values

**Tests to update:**

| Test | Old assertion | New assertion |
|------|--------------|---------------|
| `TestLoad_AbsentFile` | `TicketsDir == dir`, `ArchiveDir == dir/archive` | `TicketsDir == dir/tickets`, `ArchiveDir == dir/tickets/archive` |
| `TestLoad_EmptyTOML` | `TicketsDir == dir`, `ArchiveDir == dir/archive` | `TicketsDir == dir/tickets`, `ArchiveDir == dir/tickets/archive` |
| `TestLoad_PartialConfig_ArchiveDirOnly` | `TicketsDir == dir` (default) | `TicketsDir == dir/tickets` (new default) |

Note: `TestLoad_PartialConfig_ArchiveDirOnly` sets only `archive_dir`; the `TicketsDir` falls back to the default, which will now be `dir/tickets`.

---

### cmd/clinban — shared test infrastructure

**Files:**
- `cmd/clinban/lint_test.go` — add `setupWorkDir` helper alongside existing `buildBinary`, `writeTicket`, `projectRoot`

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|----------------|
| `setupWorkDir` | `(t *testing.T) → (root, ticketsDir, archiveDir string)` | Creates a temp dir tree matching the new default layout; returns all three paths for test use |

**Interface contract:**
- Accepts: `*testing.T`
- Returns: `root` (the temp dir itself), `ticketsDir` (`root/tickets`), `archiveDir` (`root/tickets/archive`)
- Errors: calls `t.Fatal` on any `os.MkdirAll` failure; all returned paths are guaranteed to exist

**Implementation notes:**
1. Call `t.TempDir()` to get `root`
2. Call `os.MkdirAll(filepath.Join(root, "tickets", "archive"), 0o755)` to create both subdirs in one shot
3. Derive and return the three paths

---

### cmd/clinban — existing integration tests migration

**Files affected (all in `cmd/clinban/`):**
- `lint_test.go`
- `list_test.go`
- `new_test.go`
- `move_test.go`
- `show_test.go`
- `archive_test.go`
- `edit_test.go`
- `register_test.go`

**Migration pattern:**

Before:
```go
dir := t.TempDir()
writeTicket(t, dir, "some-file.md", content)
cmd.Dir = dir
archivePath := filepath.Join(dir, "archive", filename)
```

After:
```go
root, ticketsDir, archiveDir := setupWorkDir(t)
writeTicket(t, ticketsDir, "some-file.md", content)
cmd.Dir = root
archivePath := filepath.Join(archiveDir, filename)
```

The `cmd.Dir` must be set to `root` (not `ticketsDir`) so that `findProjectRoot()` and `config.defaults()` resolve paths from the project root, not the tickets subdirectory.

---

### cmd/clinban/init.go — new command

**Files:**
- `cmd/clinban/init.go` — implements `initCmd` and its `runInit` function; registers with `rootCmd` in `init()`

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|----------------|
| `initFlags` | struct | Holds parsed flag values for the init command |
| `newInitCmd` | `() → *cobra.Command` | Constructs and returns the configured `initCmd` with flags and `PersistentPreRun` override |
| `runInit` | `(flags initFlags) → error` | Implements the full init algorithm: resolve paths, pre-flight check, conditional creation, output |

**Flags struct:**
```go
type initFlags struct {
    ticketsDir string // --tickets-dir, default "tickets"
    archiveDir string // --archive-dir, default "" (derived at runtime)
    force      bool   // --force
}
```

**Interface contract for `runInit`:**
- Accepts: `initFlags` with flag values as provided by the user (relative or absolute strings)
- Returns: `error` — the caller (`RunE` on the command) prints to stderr and exits 1 on non-nil
- Errors:
  - Exits 1 if `os.Getwd()` fails
  - Exits 1 (via returned error) if any artifact exists and `--force` is not set
  - Exits 1 (via returned error) if all artifacts exist and `--force` is set
  - Exits 1 (via returned error) if `os.Mkdir` fails for a directory
  - Exits 1 (via returned error) if `os.WriteFile` fails for `.clinban`

**`runInit` algorithm (exact):**
1. If `flags.archiveDir == ""`, set `flags.archiveDir = filepath.Join(flags.ticketsDir, "archive")`
2. Get CWD via `os.Getwd()`; return error on failure
3. Resolve absolute paths using `filepath.IsAbs` check (same logic as `config.absPath`):
   - `absTickets = filepath.Join(cwd, flags.ticketsDir)` if relative
   - `absArchive = filepath.Join(cwd, flags.archiveDir)` if relative
   - `absConfig  = filepath.Join(cwd, ".clinban")`
4. Pre-flight: stat all three; record which exist
5. Without `--force`: if any exist → print each to stderr as `"already exists: <name>"`, print `"re-run with --force to create missing items"`, return error
6. With `--force`: if all exist → print `"already fully initialized"` to stderr, return error
7. Write order for missing artifacts: `tickets/` → `tickets/archive/` → `.clinban`
8. For each artifact created, print to stdout: `"created: tickets/"`, `"created: tickets/archive/"`, `"created: .clinban"`
9. Use `os.Mkdir` (not `MkdirAll`) for directories
10. `.clinban` content written via `fmt.Fprintf` using relative (flag) values:
    ```
    tickets_dir = %q\narchive_dir = %q\n
    ```
    Written with `os.WriteFile(..., 0o600)`

**`PersistentPreRun` override:**
```go
PersistentPreRun: func(cmd *cobra.Command, args []string) {},
```
This overrides `rootCmd.PersistentPreRun` for the `init` subcommand only, bypassing `findProjectRoot()` and store initialisation. The package-level `st` and `cfg` vars remain nil; `runInit` must not reference them.

---

### cmd/clinban/init_test.go — new integration tests

**Files:**
- `cmd/clinban/init_test.go` — five integration test functions using the compiled binary

**Test functions:**

| Function | Scenario | Key assertions |
|----------|----------|----------------|
| `TestInitFreshDirectory` | Clean dir, no prior artifacts | Exit 0; stdout contains `"created: .clinban"`, `"created: tickets/"`, `"created: tickets/archive/"` |
| `TestInitAlreadyExists_NoForce` | All three artifacts exist, no `--force` | Exit 1; stderr contains each artifact name |
| `TestInitAlreadyExists_WithForce` | All three artifacts exist, `--force` | Exit 1; stderr contains `"already fully initialized"` |
| `TestInitPartial_DirsExist_NoConfig_Force` | `tickets/` and `tickets/archive/` exist, no `.clinban`, `--force` | Exit 0; stdout contains `"created: .clinban"`; no `"created: tickets/"` in stdout |
| `TestInitPartial_ConfigExists_NoDirs_Force` | `.clinban` exists, dirs absent, `--force` | Exit 0; stdout contains `"created: tickets/"` and `"created: tickets/archive/"`; no `"created: .clinban"` in stdout |

---

## Interface Contracts

| From | To | Method | Data |
|------|----|--------|------|
| `runInit` | filesystem | `os.Mkdir` | Creates `tickets/` directory at `absTickets` |
| `runInit` | filesystem | `os.Mkdir` | Creates `tickets/archive/` directory at `absArchive` |
| `runInit` | filesystem | `os.WriteFile` | Writes `.clinban` TOML at `absConfig` with relative `tickets_dir` and `archive_dir` values; mode `0o600` |
| `runInit` | stdout | `fmt.Fprintln` | `"created: <relative-name>"` for each artifact created |
| `runInit` | stderr | `fmt.Fprintln` | `"already exists: <name>"` for each conflicting artifact (no-force path) |
| `runInit` | stderr | `fmt.Fprintln` | `"re-run with --force to create missing items"` (no-force conflict hint) |
| `runInit` | stderr | `fmt.Fprintln` | `"already fully initialized"` (force + all-exist path) |
| `config.Load` | caller | return `*Config` | Returns `TicketsDir = projectRoot/tickets`, `ArchiveDir = projectRoot/tickets/archive` when `.clinban` is absent |
| `setupWorkDir` | test callers | return `(root, ticketsDir, archiveDir string)` | All three paths exist on disk; `ticketsDir = root/tickets`, `archiveDir = root/tickets/archive` |

---

## Test Strategy

**Unit tests (per module):**
- `internal/config`: `TestLoad_AbsentFile`, `TestLoad_EmptyTOML` — assert new default paths `tickets/` and `tickets/archive/`
- `internal/config`: `TestLoad_PartialConfig_ArchiveDirOnly` — assert `TicketsDir` defaults to `dir/tickets`
- `internal/config`: all other existing tests remain unchanged

**Critical paths (must pass before first ship):**
1. `clinban init` in a fresh directory creates all three artifacts and exits 0 (`TestInitFreshDirectory`)
2. `clinban init` with existing artifacts and no `--force` exits 1 and names each conflict on stderr (`TestInitAlreadyExists_NoForce`)
3. `clinban init --force` on a partially initialised directory creates only missing artifacts and exits 0 (`TestInitPartial_DirsExist_NoConfig_Force`, `TestInitPartial_ConfigExists_NoDirs_Force`)

**Integration tests:**
- All five `init_test.go` tests run the compiled binary against a temp filesystem, exercising the full CLI path including flag parsing, PersistentPreRun bypass, and artifact creation
- All migrated tests in `cmd/clinban/` (list, lint, new, move, show, archive, edit, register) must pass with the new `setupWorkDir` layout, confirming that other commands work correctly under the new default directory structure

---

## Open Questions Resolved

| Question (from 02_architecture.md) | Decision | Rationale |
|------------------------------------|----------|-----------|
| Should `--tickets-dir` / `--archive-dir` flags be validated to stay within CWD, or is an absolute path outside CWD acceptable? | No containment check. Absolute paths are treated as user-intentional. | Adds implementation complexity with minimal safety gain at this stage; absolute paths are an advanced user choice. |
| Should `init` write relative or absolute paths into `.clinban`? | Always write the relative (flag) values into `.clinban`. | Relative paths are portable across machines and user home directories; idiomatic for dotfile configs. |
| Should a failed `mkdir` after `.clinban` is written trigger cleanup / rollback? | No rollback. Document "re-run with `--force`" as the recovery path. | Rollback adds complexity; `--force` partial-creation already handles the repair case cleanly. |
