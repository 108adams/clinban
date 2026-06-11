---
title: 'resolve: partial rename leaves inconsistent state on execution failure'
status: backlog
type: bug
tags: [cli, resolve, robustness]
created: 2026-06-11T10:50:20.630521563+02:00
updated: 2026-06-11T10:50:20.630521563+02:00
---

## Problem

`runResolve` applies renames one by one with no rollback. If rename N succeeds
but rename N+1 fails (disk full, permission error, etc.), the first N files have
already been moved and the remaining duplicates are not. The repo is left in a
state that is worse than before: some IDs were reassigned, others were not, and
re-running `resolve` sees the post-rename state and allocates from a new maxID,
drifting further from the original layout.

Reproduction: three tickets share ID 0003. First rename (0003-b → 0004-b)
succeeds. Second rename (0003-c → 0005-c) hits ENOSPC. Result: 0003-b is gone,
0004-b exists, 0003-c still collides with 0003-a — and the original collision
count went from 3 to 2, masking the problem from casual inspection.

## Desired Outcome

Either all renames succeed or none take effect (atomic batch). Simplest
approach: in `planResolve`, call `os.Link` for every destination before removing
any source — same two-phase pattern already used per-file, lifted to batch
level. If any Link fails, remove already-created links and return the error.

## Acceptance Criteria

- [ ] A mid-plan rename failure leaves no files renamed (all-or-nothing)
- [ ] Existing tests continue to pass
- [ ] New test covers the partial-failure scenario (simulate failure after first link)
