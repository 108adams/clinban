# Architecture: Resolve Batch Atomicity (ticket 0023)

_Created: 2026-06-11_

## Existing Components (verified)

| Component | File:line | Responsibility |
|-----------|-----------|----------------|
| `runResolve` | cmd/clinban/resolve.go:44 | CLI entry point — builds plan, currently calls `RenameWithinDir` per-file in a loop |
| `planResolve` | cmd/clinban/resolve.go:70 | Computes rename plan; pre-flight: parse duplicates, check destinations |
| `RenameWithinDir` | internal/store/move.go:60 | Per-file atomic `os.Link + os.Remove`; same-directory only |
| `MoveToArchive` | internal/store/move.go:16 | Per-file atomic `os.Link + os.Remove`; active → archive |
| `MoveToActive` | internal/store/move.go:40 | Per-file atomic `os.Link + os.Remove`; archive → active |

All filesystem mutation in the codebase is encapsulated in `internal/store/`. The CLI layer never calls `os.Link` or `os.Remove` directly.

## Proposed Changes

| Change | Replaces/extends | Rationale |
|--------|-----------------|-----------|
| Add `BatchRenameWithinDir(ops []RenameOp) error` to `internal/store/move.go` | Extends move.go alongside `RenameWithinDir` | Lifts the two-phase link+remove pattern to batch level with rollback; keeps all filesystem mutation in the store layer |
| Replace per-file loop in `runResolve` with a single `st.BatchRenameWithinDir(plan)` call | `runResolve:60–67` | CLI becomes a thin caller; rollback logic stays out of the command handler |

`RenameWithinDir` remains unchanged — it is the correct primitive for single-file callers.

## Integration Contracts

| Dependency | Protocol | Format | Failure mode | Owner |
|------------|---------|--------|-------------|-------|
| `os.Link` | POSIX syscall | `(oldpath, newpath string)` | `ErrExist` if dest exists; `EXDEV` impossible (same-dir guarantee); other errors propagate | stdlib |
| `os.Remove` | POSIX syscall | `(path string)` | Propagates as error; triggers rollback | stdlib |
| `errors.Join` | Go 1.20+ stdlib | `(errs ...error) error` | N/A | stdlib (Go 1.25 confirmed) |

`os.Link` same-filesystem constraint is satisfied by design: `RenameWithinDir` and `BatchRenameWithinDir` both operate within a single directory.

## NFRs

| Category | Requirement | Target | Status |
|----------|------------|--------|--------|
| Correctness | Store unchanged on any error exit | Zero partial renames observable after exit 1 | Required — this is the bug |
| Resilience | SIGKILL between phases | Undefined / out of scope (BA confirmed) | Accepted limitation |
| Resilience | Rollback on Remove failure | Best-effort: attempt all rollback removals; report all failures via `errors.Join` | Required |
| Observability | Error reporting | `resolve: link <basename>: <err>`, `resolve: remove <basename>: <err>`, `resolve: rollback: remove <basename>: <err>` | Required |
| Performance | Throughput | O(2n) syscalls for n renames; no target — single-user CLI | Trivially satisfied |
| Security | Attack surface | No new surface; same filesystem permissions as existing rename ops | No change |

All other NFR categories (availability, scalability, compliance, operability, data retention) are N/A for a local single-user CLI.

## ADRs

## ADR-1: Batch rename and rollback responsibility — store layer

**Status:** `accepted`

**Decision:** Add `BatchRenameWithinDir` to `internal/store/move.go`; the CLI calls it with the computed plan and receives a single error.

**Context:** Ticket 0023 requires all-or-nothing semantics for a batch of within-directory renames. The rollback logic (remove created destination links on failure) is filesystem error recovery. Every other filesystem mutation in the codebase lives in `internal/store/`; the CLI never calls `os.Link` or `os.Remove` directly.

**Alternatives:**

| Option | Rejected because |
|--------|-----------------|
| Two-phase loop in `runResolve` (CLI layer) | Breaks the established boundary: CLI would be the first caller of `os.Link`/`os.Remove` directly; rollback logic mixed into command handler |
| Reuse `RenameWithinDir` per-file with manual rollback in CLI | Same boundary violation; also duplicates rollback logic outside the store |

**Rationale:** The store already owns `MoveToArchive`, `MoveToActive`, and `RenameWithinDir` — all using the same `os.Link + os.Remove` idiom. `BatchRenameWithinDir` is a natural extension of this family. Keeping the rollback inside the store means the CLI stays a thin coordinator: build plan, call store, print results.

**Consequences:**
- `+` CLI command handler remains free of low-level filesystem error recovery
- `+` Future commands needing batch renames can reuse `BatchRenameWithinDir`
- `+` Consistent with established store boundary
- `-` New exported type (`RenameOp` or equivalent) in the store package
- `!` Rollback is best-effort: if `os.Remove(destLink)` fails during rollback, both old and new paths exist; this is a known limitation documented in the ticket

**Locks:**
- `internal/store/move.go` is the only location where batch rename and rollback logic may live
- `runResolve` must not call `os.Link` or `os.Remove` directly
- Multi-error for rollback failures must use `errors.Join` (Go 1.20+, confirmed available)
- `BatchRenameWithinDir` must attempt all rollback removals even when one fails (best-effort policy from BA)

## Open Questions

| Question | Owner | Blocking? |
|----------|-------|-----------|
| Name of the input type: `RenameOp`, `BatchRenameItem`, or plain `{OldPath, NewBase string}` struct? | Tech Lead | No |
| Should `BatchRenameWithinDir` validate that all ops share the same directory, or is same-directory a caller precondition? | Tech Lead | No |
| Error message format when rollback produces multiple errors: join with `\n` or `;`? | Tech Lead | No |
