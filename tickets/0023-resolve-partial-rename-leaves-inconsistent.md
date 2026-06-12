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

Replace the current one-by-one rename loop with a two-phase batch built on hard links (`os.Link` + `os.Remove`, matching the existing single-file rename primitive). After a `Link`, the old and new names refer to the same underlying file.

**Phase 1 — Link:** call `os.Link(oldPath, destPath)` for every rename in the plan. On the first failure, remove every destination link created so far and exit 1. No source file has been touched — zero net change.

**Phase 2 — Remove:** call `os.Remove(oldPath)` for every rename, tracking which sources have been removed. On the first failure, roll back in this order, then exit 1:

1. **Restore** — for every source already removed in this phase, re-create it with `os.Link(destPath, oldPath)`. The destination still holds the file, so this brings back the original name.
2. **Clean up** — remove every destination link created in Phase 1.

After rollback every original name is present and no destination link remains: zero net change.

> Why restore is required: once `os.Remove(oldPath)` succeeds, the destination link is the *only* remaining name for that file. Removing destination links without restoring first would delete those tickets. Simply "removing all destination links" is only safe when Phase 2 fails on its very first item (nothing removed yet).

If rollback itself fails (a restore re-link or a destination removal cannot complete), report the original error plus every rollback error. The store may be inconsistent in this case — say so in the output. Attempt all rollback steps even if one fails.

## Error Messages

- Link failure: `resolve: link <basename>: <os error>`
- Remove failure: `resolve: remove <basename>: <os error>`
- Rollback, destination cleanup failure: `resolve: rollback: remove <basename>: <os error>`
- Rollback, source restore failure: `resolve: rollback: link <basename>: <os error>`

`<basename>` is the basename of the ticket file involved in the failed operation.

## Edge Cases

- Phase 1 fails at item N → clean up links 0..N-1, exit 1, zero net change
- Phase 2 fails on the first item (nothing removed yet) → clean up all destination links (no restore needed), exit 1, zero net change
- Phase 2 fails at item N>1 → restore the sources removed for items 0..N-1, then clean up all destination links, exit 1, zero net change
- Rollback fails for one or more sources or destinations → report original + all rollback errors; store may be inconsistent; this is a known limitation
- Concurrent resolve processes → undefined behaviour; no locking in scope
- SIGKILL between phases → undefined; explicitly out of scope

## Acceptance Criteria

- [ ] Given a 3-rename plan, when the 3rd Link fails, then no files are renamed and exit code is 1
- [ ] Given all Links succeed, when the 1st Remove fails, then all destination links are removed, every original file is untouched, and exit code is 1
- [ ] Given all Links succeed, when the 2nd Remove fails after the 1st succeeded, then every original ticket file is restored at its original path with identical content, no destination link remains, and exit code is 1
- [ ] Given a complete successful run, then each ticket file is at its new path with identical content and the old path is gone
- [ ] Given no conflicts, then "no conflicts found" is printed, exit 0, and no filesystem changes occur
- [ ] Given a successful run, `clinban lint` reports no duplicate ID errors
- [ ] Existing resolve integration tests pass unchanged

## Tests Required

- Phase-1 failure: pre-create a destination file between plan build and execution to force a Link failure on a specific item; assert no other files were renamed
- Phase-2 Remove failure on the first item: make a source directory read-only (`chmod 0555`) to block Remove; assert all destination links are cleaned up, exit code is 1
- Phase-2 Remove failure after a prior success (item N>1): force Remove to fail on a later item; assert already-removed sources are restored to their original paths and no destination link remains
- Note: `chmod 0555` does not block removal when the test runs as root; guard the permission-based tests with a skip when `os.Geteuid() == 0`
- Existing tests must pass without modification

## Out of Scope

- SIGKILL / process crash atomicity
- `--dry-run` flag (separate ticket)
- Concurrent invocation safety
- Rollback failure recovery beyond reporting the error
