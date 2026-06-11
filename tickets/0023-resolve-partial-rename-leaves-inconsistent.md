---
title: 'resolve: partial rename leaves inconsistent state on execution failure'
status: backlog
type: bug
tags: [cli, resolve, robustness]
created: 2026-06-11T10:50:20.630521563+02:00
updated: 2026-06-11T10:50:20.630521563+02:00
---

## Problem

`runResolve` applies renames one by one with no rollback. If rename N succeeds but rename N+1 fails (disk full, permission error, etc.), renames 1..N are already committed to disk and the remaining duplicates are not resolved. The store is left in a state that is harder to repair than the original collision: some IDs were reassigned, others were not, and re-running `resolve` allocates from a new maxID, drifting further from the original layout.

## Desired Outcome

`clinban resolve` is all-or-nothing. After the command exits, the store is in one of two states:

- All planned renames applied (exit 0), or
- Zero renames applied (exit 1)

SIGKILL between phases is explicitly out of scope.

## Execution Model

Replace the current one-by-one rename loop with a two-phase batch:

**Phase 1 — Link:** call `os.Link(oldPath, destPath)` for every rename in the plan. On the first failure, remove all destination links created so far and exit 1. No source file has been touched.

**Phase 2 — Remove:** call `os.Remove(oldPath)` for every rename. On the first failure, remove all destination links (full phase-1 cleanup) and exit 1. Zero net change to the store.

If rollback itself fails (cannot remove a destination link), report both the original error and the rollback error. The store may be inconsistent in this case — document it in the error message. Attempt all rollback removals even if one fails.

## Error Messages

- Link failure: `resolve: link <basename>: <os error>`
- Remove failure: `resolve: remove <basename>: <os error>`
- Rollback failure (additional): `resolve: rollback: remove <basename>: <os error>`

## Edge Cases

- Phase 1 fails at item N → clean up links 0..N-1, exit 1, zero net change
- Phase 2 fails at item N → clean up all destination links (full phase-1 cleanup), exit 1, zero net change
- Rollback fails for one or more destinations → report original + rollback errors; store may be inconsistent; this is a known limitation
- Concurrent resolve processes → undefined behaviour; no locking in scope
- SIGKILL between phases → undefined; explicitly out of scope

## Acceptance Criteria

- [ ] Given a 3-rename plan, when the 3rd Link fails, then no files are renamed and exit code is 1
- [ ] Given all Links succeed, when the 1st Remove fails, then all destination links are removed and exit code is 1
- [ ] Given a complete successful run, then each ticket file is at its new path with identical content and the old path is gone
- [ ] Given no conflicts, then "no conflicts found" is printed, exit 0, and no filesystem changes occur
- [ ] Given a successful run, `clinban lint` reports no duplicate ID errors
- [ ] Existing resolve integration tests pass unchanged

## Tests Required

- Phase-1 failure: pre-create a destination file between plan build and execution to force a Link failure on a specific item; assert no other files were renamed
- Phase-2 Remove failure: make a source directory read-only (`chmod 0555`) to block Remove; assert all destination links are cleaned up and exit code is 1
- Existing tests must pass without modification

## Out of Scope

- SIGKILL / process crash atomicity
- `--dry-run` flag (separate ticket)
- Concurrent invocation safety
- Rollback failure recovery beyond reporting the error
