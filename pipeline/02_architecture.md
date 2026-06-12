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
| Add exported `store.RenameOp{OldPath, NewBase string}` to `internal/store/move.go` | New store-owned DTO | Cross-package API: `planResolve` (in `cmd/clinban`) cannot pass its unexported `resolveRename` (resolve.go:39–42) into the store; the store must own the input type |
| Change `planResolve` to return `[]store.RenameOp` | `planResolve:70` (currently returns `[]resolveRename`) | Removes the package-private DTO; planning output feeds the store API directly with no conversion layer |
| Add `BatchRenameWithinDir(ops []store.RenameOp) ([]string, *BatchError)` to `internal/store/move.go` | Extends move.go alongside `RenameWithinDir` | Lifts the two-phase link+remove pattern to batch level with data-loss-safe rollback; keeps all filesystem mutation in the store layer. Returns new paths so the CLI can print `renamed: old -> new` (resolve.go:65) |
| Add exported `store.BatchError` (typed) | New | Store stays package-scoped in its own messages but carries enough structure (op kind, basename, cause, rollback errors, inconsistent flag) for the CLI to format the exact `resolve:`-prefixed messages the ticket requires |
| Replace per-file loop in `runResolve` with a single `st.BatchRenameWithinDir(plan)` call | `runResolve:60–67` | CLI becomes a thin caller; rollback logic stays out of the command handler. CLI maps `*BatchError` → `resolve:` messages |

`RenameWithinDir` remains unchanged — it is the correct primitive for single-file callers.

## Integration Contracts

| Dependency | Protocol | Format | Failure mode | Owner |
|------------|---------|--------|-------------|-------|
| `os.Link` | POSIX syscall | `(oldpath, newpath string)` | `ErrExist` if dest exists; `EXDEV` impossible — see contract below; other errors propagate | stdlib |
| `os.Remove` | POSIX syscall | `(path string)` | Propagates as error; triggers rollback | stdlib |
| `errors.Join` | Go 1.20+ stdlib | `(errs ...error) error` | N/A | stdlib (Go 1.25 confirmed) |

**Same-directory / same-filesystem contract (makes `EXDEV` impossible — enforced, not assumed):**
`BatchRenameWithinDir` validates each `op.NewBase == filepath.Base(op.NewBase)` (rejects path separators), exactly as `RenameWithinDir` already does (move.go:61–63). Each destination is `filepath.Join(filepath.Dir(op.OldPath), op.NewBase)` — same directory as the source by construction. Therefore source and destination always share a filesystem and `os.Link` cannot return `EXDEV`. Validation runs in a pre-flight pass before any mutation; a bad basename fails the whole batch with zero filesystem change. This also preserves the local-filesystem safety property in `docs/security.md:17–24`: a future caller cannot use the batch API to create links outside the source directory.

## Execution Model & Rollback Algorithm

> **Correctness note:** the ticket's Phase-2 cleanup ("remove all destination links", 0023 lines 29 & 42) is **data-loss-unsafe as written** and is corrected here. After `os.Remove(oldPath)` succeeds, the hard-linked `destPath` is the *only* remaining name for that inode; removing it during rollback deletes the ticket. The corrected Phase-2 rollback **re-links already-removed sources before removing destinations**. See "Requirement Corrections" below.

`BatchRenameWithinDir(ops)` runs three passes. `dest_i = filepath.Join(filepath.Dir(ops[i].OldPath), ops[i].NewBase)`.

**Pre-flight (no mutation):** validate every `NewBase` is basename-only. On any failure → return `BatchError{Op: Validate}`, zero filesystem change.

**Phase 1 — Link:** for each op, `os.Link(OldPath_i, dest_i)`. Track `linked` = dests created so far.
- On failure at item k → roll back Phase 1: `os.Remove` every dest in `linked` (best-effort, collect errors). No source touched. Return `BatchError{Op: Link, Base: base_k, Err, Rollback}`.

**Phase 2 — Remove:** for each op, `os.Remove(OldPath_i)`. Track `removed` = indices whose source is gone.
- On failure at item k → roll back Phase 2 (data-loss-safe):
  1. **Re-link** every already-removed source: `os.Link(dest_j, OldPath_j)` for `j ∈ removed`. This restores the original name from the surviving destination link. (best-effort, collect errors)
  2. **Remove** every Phase-1 destination link in `linked`. (best-effort, collect errors)
  3. Result: every `OldPath` exists, every `dest` is gone → zero net change.
  - Return `BatchError{Op: Remove, Base: base_k, Err, Rollback, Inconsistent: len(Rollback)>0}`.

**Success:** return `newPaths` (the `dest_i`, in plan order) and `nil`.

Per-op state tracked: `linked` (dests created), `removed` (sources deleted). These two slices are sufficient to drive a correct rollback from either phase.

## NFRs

| Category | Requirement | Target | Status |
|----------|------------|--------|--------|
| Correctness | Store unchanged on any error exit | Zero partial renames observable after exit 1, **including after partial Phase-2 success** (rollback re-links removed sources — no ticket deletion) | Required — this is the bug |
| Resilience | SIGKILL between phases | Undefined / out of scope (BA confirmed) | Accepted limitation |
| Resilience | Rollback on Remove failure | Best-effort: attempt all re-links and all dest removals even if one fails; collect every failure | Required |
| Observability | Error reporting (CLI owns format) | `resolve: link <basename>: <err>`, `resolve: remove <basename>: <err>`, `resolve: rollback: remove <basename>: <err>`, **`resolve: rollback: link <basename>: <err>`** (re-link failure — new variant) | Required |
| Maintainability | Error ownership | Store returns typed `*BatchError` (op kind, basename, cause, `[]Rollback`, `Inconsistent`); CLI maps it to the `resolve:`-prefixed strings. Store never embeds the `resolve` command name. | Required — resolves Challenge on error ownership |
| Performance | Throughput | O(2n) syscalls for n renames (worst-case rollback adds ≤2n more); no target — single-user CLI | Trivially satisfied |
| Security | Attack surface | No new surface. Basename-only validation (see same-directory contract) prevents links escaping the source dir; preserves `docs/security.md:17–24` local-fs safety | No change |

All other NFR categories (availability, scalability, compliance, operability, data retention) are N/A for a local single-user CLI.

**Testability note (for the tasks stage):** the Phase-2 remove-failure test relies on `chmod 0555` to block `os.Remove`. This does **not** block removal when the test process runs as root (root bypasses directory write permission). The test must `t.Skip` under `os.Geteuid() == 0`, or use an injectable failure hook. Rollback-failure reporting (multiple `resolve: rollback: ...` lines) also needs explicit test coverage (ticket 0023 lines 31–37).

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
- `+` Typed `*BatchError` keeps `resolve:` message formatting in the CLI while the store stays package-scoped
- `-` New exported types (`store.RenameOp`, `store.BatchError`) in the store package; `planResolve` now couples to `store.RenameOp` (accepted — avoids a pointless conversion layer)
- `!` Rollback is best-effort: if a re-link or dest-removal fails during Phase-2 rollback, the store may be inconsistent (`BatchError.Inconsistent = true`) and all such failures are reported. Only this residual case is the documented limitation — the common partial-Phase-2 case is now fully recovered, not left half-applied.

**Locks:**
- `internal/store/move.go` is the only location where batch rename and rollback logic may live
- `runResolve` must not call `os.Link` or `os.Remove` directly
- Phase-2 rollback MUST re-link already-removed sources before removing destination links (data-loss-safe; see Execution Model)
- `BatchRenameWithinDir` must attempt all rollback steps even when one fails (best-effort policy from BA)
- Store returns typed `*BatchError`; the `resolve:` string prefix is owned by `cmd/clinban`, never by `internal/store`
- `NewBase` must be basename-only (validated, mirrors `RenameWithinDir`)

## Requirement Corrections (flagged to ticket 0023)

The architecture corrects two gaps in the ticket's execution model. **These should be reconciled back into the ticket via `/ba` before the acceptance criteria are treated as final:**

1. **Phase-2 rollback (ticket lines 29, 42–43):** "remove all destination links" is data-loss-unsafe once any source has been removed in Phase 2 — it deletes the only surviving link to the inode. Corrected: re-link removed sources from their destinations, *then* remove destinations.
2. **Error messages (ticket lines 33–37):** the corrected rollback can fail while re-linking, which has no message in the ticket. Added variant: `resolve: rollback: link <basename>: <err>`.

Acceptance criterion 0023 line 50 ("when the 1st Remove fails, then all destination links are removed") stays correct (first remove → `removed` is empty → no re-link needed). The new path that needs a test: **Remove fails at item N>1**, asserting items 0..N-1 are restored to their original names.

## Open Questions

| Question | Owner | Resolution |
|----------|-------|------------|
| Name of the input type | Tech Lead | **Resolved:** `store.RenameOp{OldPath, NewBase string}` (exported, store-owned) |
| Same-directory: validate or caller precondition? | Tech Lead | **Resolved:** store validates `NewBase` is basename-only; same-directory holds by construction (dest joined to source's dir). Documented contract above |
| Rollback multi-error format | Tech Lead | **Resolved:** store returns structured `[]Rollback` on `BatchError`; CLI emits one `resolve: rollback: ...` line per error (no `\n`/`;` joining at store layer) |
