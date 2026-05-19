# Architecture: `clinban init`

## Existing Components (verified)

| Component | File:line | Responsibility |
|---|---|---|
| `PersistentPreRun` | `cmd/clinban/root.go:38–49` | Runs before every subcommand: discovers project root, loads config, builds store |
| `findProjectRoot()` | `cmd/clinban/root.go:64–87` | Walks up from CWD looking for `.clinban`; falls back to CWD if not found |
| `config.Load()` | `internal/config/config.go:36–70` | Reads `.clinban` TOML; returns defaults when file is absent |
| `config.defaults()` | `internal/config/config.go:73–78` | Returns `TicketsDir = projectRoot`, `ArchiveDir = projectRoot/archive` — **to be changed** |
| `Config` struct | `internal/config/config.go:20–25` | Holds `TicketsDir` and `ArchiveDir` as resolved absolute paths |

## Proposed Changes

| Change | Replaces/extends | Rationale |
|---|---|---|
| `cmd/clinban/init.go` — new `initCmd` | Extends CLI command set | New user-facing command |
| `initCmd` defines its own `PersistentPreRun` (no-op) | Overrides `rootCmd.PersistentPreRun` for this subcommand | `findProjectRoot()` walks up to find an existing `.clinban`; `init` must write to CWD unconditionally |
| `config.defaults()` updated to `tickets/` and `tickets/archive/` | Replaces current defaults (`projectRoot`, `projectRoot/archive`) | Pre-release; new defaults match `init`'s intentional layout |
| `config_test.go` — update `TestLoad_AbsentFile`, `TestLoad_PartialConfig_TicketsDirOnly` | Reflect new defaults | Tests assert on specific default paths |

## Integration Contracts

| Dependency | Protocol | Format | Failure mode | Owner |
|---|---|---|---|---|
| `.clinban` file | `os.WriteFile` | TOML (`tickets_dir`, `archive_dir` as relative paths) | File already exists → non-0 exit (or `--force` skip if partial) | `initCmd` |
| `tickets_dir` directory | `os.Mkdir` | Filesystem directory | Dir already exists → non-0 exit (or `--force` skip if partial) | `initCmd` |
| `archive_dir` directory | `os.Mkdir` | Filesystem directory | Dir already exists → non-0 exit (or `--force` skip if partial) | `initCmd` |

## NFRs

| Category | Requirement | Target | Status |
|---|---|---|---|
| Security | Configured paths must not escape CWD | Validate `--tickets-dir` and `--archive-dir` stay within CWD (or are absolute and user-intentional) | Open — Tech Lead to decide containment check scope |
| Security | No silent overwrites | Any pre-existing artifact blocks write without `--force` | Addressed by design |
| Resilience | Partial-write failure | If a `mkdir` fails after `.clinban` is written, state is inconsistent; no rollback required — user re-runs with `--force` | Accepted trade-off |
| Operability | Clear output | Each created artifact reported on stdout; each blocking artifact reported on stderr | Requirement |
| Operability | Exit codes | `0` = success (≥1 artifact created); `1` = error or all-exist-with-force | Requirement |

## ADRs

### ADR-1: `initCmd` defines its own `PersistentPreRun` to bypass root startup

**Status:** `accepted`
**Decision:** `initCmd` defines a no-op `PersistentPreRun`, overriding `rootCmd`'s for this subcommand only.
**Context:** `rootCmd.PersistentPreRun` calls `findProjectRoot()`, which walks up the directory tree looking for an existing `.clinban`. This is wrong for `init`: the command must write to CWD unconditionally, regardless of what ancestors contain. The package-level `st` (store) and `cfg` vars remain nil for `init`; that is safe since `init` does not use them.
**Alternatives:**
| Option | Rejected because |
|---|---|
| Guard inside `rootCmd.PersistentPreRun` | Spreads `init`-specific concern into shared startup; fragile as commands grow |
| Use `PreRun` instead of `PersistentPreRun` on `initCmd` | Parent's `PersistentPreRun` still runs before `PreRun`; does not achieve bypass |

**Rationale:** Cobra's documented override rule — a subcommand's `PersistentPreRun` replaces the parent's for that command — is the minimal, idiomatic solution. No changes to shared code required.
**Consequences:**
- `+` `init` operates on CWD regardless of ancestor `.clinban` files
- `+` No changes to `rootCmd` or `findProjectRoot()`
- `-` `st` and `cfg` are nil during `init`; callers of `init` internals must not assume they are set
- `!` If future commands also need to bypass startup, the pattern must be repeated or refactored

**Locks:** `initCmd` must not call any function that dereferences `st` or `cfg`.

---

### ADR-2: `config.Load` defaults change to `tickets/` and `tickets/archive/`

**Status:** `accepted`
**Decision:** Update `config.defaults()` to return `TicketsDir = projectRoot/tickets` and `ArchiveDir = projectRoot/tickets/archive`.
**Context:** The current defaults (`TicketsDir = projectRoot`) were a zero-config fallback where tickets live in the project root directory. `init` establishes an intentional layout with a dedicated subdirectory. Pre-release status means no backward-compatibility constraint applies. Keeping two different defaults (one in `config.Load`, one in `init`) would create a confusing split where the zero-config and init-config layouts differ silently.
**Alternatives:**
| Option | Rejected because |
|---|---|
| Keep `config.Load` defaults as-is; `init` writes a different layout | Zero-config and `init` paths diverge; a project without `.clinban` behaves differently from one created by `init` |
| Change defaults only in `init`, not `config.Load` | Requires `init` to write `.clinban` even when defaults are used, just to normalize behavior |

**Rationale:** A single canonical layout prevents divergence between `init`-created and hand-configured projects at this stage. Existing `config_test.go` tests for default paths must be updated.
**Consequences:**
- `+` Zero-config and `init`-config projects have the same directory structure
- `-` Breaks any existing projects relying on the old zero-config defaults (accepted: pre-release)
- `!` `config_test.go` tests asserting on default paths will fail until updated

**Locks:** All code that previously assumed `TicketsDir == projectRoot` by default must be reviewed for correctness under the new default.

---

### ADR-3: `--force` enables partial creation; all-exist is an error

**Status:** `accepted`
**Decision:** Without `--force`, any pre-existing artifact causes a non-0 exit listing all conflicts. With `--force`, only missing artifacts are created and each is reported. If all artifacts already exist, `--force` exits non-0 with "already fully initialized."
**Context:** `init` manages three artifacts: `.clinban`, `tickets/`, `tickets/archive/`. A project may be partially initialized (e.g., dirs exist but config is missing). `--force` is the repair path. "All exist with force" being an error prevents silent no-ops that would mask misconfiguration.
**Alternatives:**
| Option | Rejected because |
|---|---|
| `--force` is always a no-op success when all exist | Silent no-op masks already-initialized state; user gets no signal |
| `--force` overwrites existing artifacts (e.g., rewrite `.clinban`) | Destructive; destroys user customization of an existing config |

**Rationale:** Partial creation under `--force` makes `init` a safe repair tool. The "all exist" error gives the user explicit feedback that setup is complete and no action was taken.
**Consequences:**
- `+` `init` is safe to run on a fresh project
- `+` `--force` repairs partial setups without destroying existing config
- `-` Users must explicitly pass `--force` to repair; there is no auto-detect-and-fix
- `!` Determining "all exist" requires checking all three artifacts before any write

**Locks:** `init` must check all three artifacts for existence before writing any of them (pre-flight check). Write order after pre-flight: `tickets/` → `tickets/archive/` → `.clinban`.

## Open Questions

| Question | Owner | Blocking? |
|---|---|---|
| Should `--tickets-dir` / `--archive-dir` flags be validated to stay within CWD, or is an absolute path outside CWD acceptable (user intent)? | Tech Lead | No — default to no containment check; absolute paths are user-intentional |
| Should `init` create `archive_dir` as a relative path always (e.g., `tickets/archive`), or write the absolute path into `.clinban`? | Tech Lead | No — relative paths are idiomatic and portable |
| Should a failed `mkdir` after `.clinban` is written trigger any cleanup, or is "re-run with --force" the documented recovery? | Tech Lead | No — accepted: no rollback; re-run with `--force` |
