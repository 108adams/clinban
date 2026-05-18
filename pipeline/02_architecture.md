# Architecture: Clinban

## Existing Components (verified)

None. Project is greenfield. Only pipeline documents exist at time of writing.

---

## Proposed Components

| Component | Package path | Responsibility |
|---|---|---|
| CLI entry point | `cmd/clinban` | Cobra root command + subcommand wiring |
| Data model | `internal/ticket` | Ticket struct; YAML frontmatter marshal/unmarshal; schema is the external automata contract |
| File store | `internal/store` | Scan IDs; read/write/move ticket files; atomic write; directory layout; `Record` type pairs a parsed ticket with its path and archive flag |
| Lint engine | `internal/lint` | Validate Ticket struct in memory against all schema rules; no filesystem dependency |
| State machine | `internal/fsm` | Transition table; ValidateTransition(from, to); no model or store dependency |
| Editor | `internal/editor` | Detect $EDITOR env var; fall back to vi; launch process; wait for exit |
| Config | `internal/config` | Load .clinban TOML; resolve tickets_dir and archive_dir with defaults |
| Slug | `internal/slug` | Title → 5-word slug: lowercase, hyphens, strip non-alphanumeric |

**Dependency rule (enforced):** `internal/lint` and `internal/fsm` must not import `internal/store`.
`internal/ticket` must not import `internal/store`. This constraint ensures the schema contract
is self-contained and testable without a filesystem.

---

## Integration Contracts

| Dependency | Protocol | Format | Failure mode | Owner |
|---|---|---|---|---|
| Filesystem | POSIX file I/O | Markdown + YAML frontmatter | Atomic rename; abort on write error with stderr message | `internal/store` |
| $EDITOR | OS process exec (os/exec) | n/a | Fall back to vi if $EDITOR unset; propagate non-zero exit to caller | `internal/editor` |
| .clinban config file | File read | TOML | Absent → silent defaults; malformed → fatal error with message, exit 1 | `internal/config` |

---

## Ticket Schema (external contract)

This is the authoritative specification for all actors — human and automaton.

```yaml
---
id: "0042"                        # string; 4-digit zero-padded integer; set by Clinban; must match filename
status: "in-progress"             # string; one of: backlog, in-progress, blocked, done; set by Clinban on create
type: "bug"                       # string; one of: bug, task, feature, spike; required; set by author
title: "Fix login timeout on..."  # string; required; non-empty; set by author
tags: []                          # list of strings; optional; free-form
created: "2026-05-18T14:30:00Z"  # RFC3339 timestamp; set by Clinban on creation; never modified after
updated: "2026-05-18T15:00:00Z"  # RFC3339 timestamp; set by Clinban on every write
---
```

**Fields owned by Clinban:** `id`, `created`, `updated`. These are always overwritten on creation
and registration — external values are discarded.

**Fields owned by the author:** `type`, `title`, `tags`. Clinban updates `status` via `clinban move`.

**Filename convention:** `<id>-<slug>.md` — e.g. `0042-fix-login-timeout-on.md`.
Slug = first 5 words of title, lowercased, hyphens, non-alphanumeric stripped.
Filename is set at creation and never renamed on subsequent edits.

---

## NFRs

| Category | Requirement | Target | Status |
|---|---|---|---|
| Performance | ID scan (active + archive) | <500ms at 10,000 tickets | Open — Tech Lead to verify |
| Availability | Single binary; no uptime SLA | n/a | Accepted |
| Security | Path traversal prevention | Ticket IDs and filenames validated before path construction | Required |
| Resilience | Partial write prevention | Atomic write-temp-rename; temp file in same directory as target | Required |
| Scalability | Active ticket set size | Archive offloads done tickets; filesystem scan viable to thousands | Accepted |
| Observability | Exit codes | 0 = success, 1 = any error | Required |
| Observability | Output streams | Errors → stderr; all other output → stdout | Required |
| Data | Retention / backup | Git repository managed independently | Accepted by design |
| Operability | Distribution | `go install` or pre-built binary | Open — Tech Lead |
| Compliance | None | n/a | Accepted |

---

## ADRs

### ADR-1: CLI Framework — Cobra

**Status:** `accepted`
**Decision:** Use the Cobra library as the CLI framework for all command routing and flag parsing.
**Context:** Clinban has 8+ subcommands with distinct flag sets. A framework is needed to route
commands, parse flags, and generate consistent help text. The only realistic alternatives are
the stdlib `flag` package or `urfave/cli`.
**Alternatives:**
| Option | Rejected because |
|---|---|
| stdlib `flag` | No subcommand routing; manual help generation; significant boilerplate for every command |
| `urfave/cli` | Lower ecosystem adoption in Go tooling; Cobra is the industry standard (kubectl, gh, hugo, helm) |

**Rationale:** Cobra is the de facto standard for Go CLIs. Consistent help output, shell completion,
and flag validation require zero custom code. Any Go contributor will recognise the pattern immediately.
**Consequences:**
- `+` Consistent help, flag validation, and error formatting across all commands
- `+` Shell completion generation available at no cost
- `-` Adds one external dependency
- `!` Cobra v1 and v2 have incompatible APIs — version must be pinned at project init
**Locks:** All commands are implemented as `*cobra.Command`. No alternative CLI dispatch pattern
is permitted anywhere in the codebase.

---

### ADR-2: Package Decomposition — Separate model, store, and lint

**Status:** `accepted`
**Decision:** Implement `internal/ticket` (data model), `internal/store` (file I/O), and
`internal/lint` (validation) as distinct packages, with an enforced rule that lint and fsm
must not import store.
**Context:** The ticket schema is the external contract for automata, which write files directly
and use lint as their only validation layer. If lint depends on the filesystem, the contract
becomes implicit and untestable without a real file tree. Separating the packages makes the
schema self-contained.
**Alternatives:**
| Option | Rejected because |
|---|---|
| Single `ticket` package owning model + I/O + lint | Entangles schema validation with filesystem; lint becomes untestable without real files; violates automata contract clarity |
| `ticket` + `store` merged, `lint` separate | Better separation but still conflates ID generation (a store concern) with data modelling |

**Rationale:** The no-import rule (lint has no store dependency) makes the Go struct in
`internal/ticket` the canonical schema representation. Lint rules are unit-testable with
in-memory structs alone. Automata can reason about the contract without understanding the
filesystem layout.
**Consequences:**
- `+` Lint rules are unit-testable without any filesystem setup
- `+` Schema contract is fully self-contained in `internal/ticket`
- `+` Two-phase validation contract is explicit: parse errors (`ticket.Parse()`) are categorically distinct from schema violations (`lint.Lint()`); lint cannot run on a ticket that failed to parse
- `-` More packages to navigate for changes that touch model + I/O together
- `!` The no-import rule requires active enforcement (tooling or CI check)
**Locks:** `internal/lint` and `internal/fsm` must not import `internal/store`. `internal/ticket`
must not import `internal/store`. This constraint applies to all packages added in future.

---

### ADR-3: Atomic File Writes — Write-temp-then-rename in target directory

**Status:** `accepted`
**Decision:** All ticket file writes use write-to-temp-file-then-rename, with the temp file
created in the same directory as the final target path.
**Context:** Clinban is the single source of truth for task state in the repository. A partial
write (due to crash or interrupt) would silently corrupt a ticket visible to git and other
processes. Automata may access the ticket directory concurrently with Clinban writes.
**Alternatives:**
| Option | Rejected because |
|---|---|
| Direct `os.WriteFile` to target path | Not atomic; interrupted write leaves a corrupt file visible to all readers |
| Write to system /tmp then rename to target | Cross-device rename is not atomic on POSIX; /tmp is frequently on a different filesystem than the project |

**Rationale:** `os.Rename` on the same filesystem is guaranteed atomic by the POSIX specification.
Writing the temp file in the same directory as the target ensures the rename is always
same-filesystem. This is the canonical safe-write pattern for Unix file-based storage.
**Consequences:**
- `+` No partial writes are ever visible in the ticket directory or to git
- `-` The same-filesystem constraint means temp files must never be created in /tmp
- `!` If the process crashes between the write and the rename, a temp file is left in the
  ticket directory; cleanup strategy deferred to v2
**Locks:** `internal/store` must use temp-then-rename for every ticket file write.
Direct `os.WriteFile` to a final ticket path is forbidden.

---

## Open Questions

| Question | Owner | Blocking? |
|---|---|---|
| ~~YAML library choice~~ | Resolved | `gopkg.in/yaml.v3` (T-01) |
| ~~TOML library choice~~ | Resolved | `github.com/BurntSushi/toml` (T-01) |
| ~~Template placeholder format~~ | Resolved | `embed.FS` with `text/template` (T-17) |
| ~~Terminal width detection~~ | Resolved | `golang.org/x/term` (T-01, T-12) |
| ~~Test strategy~~ | Resolved | Unit + filesystem integration tests (T-08, T-17) |
| Cobra version to pin (v1 vs v2) | Tech Lead | No — decide at `go mod init` |
| Shell completion: generate and install by default, or opt-in? | Tech Lead | No |
| Distribution method (`go install`, pre-built binary, package manager) | Tech Lead | No |
| Temp file cleanup on crash during atomic write | Tech Lead | No — low priority for v1 |
