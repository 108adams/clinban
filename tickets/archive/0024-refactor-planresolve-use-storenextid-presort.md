---
title: 'refactor: planResolve — use store.NextID, pre-sort IDs, single-pass group build'
status: done
type: task
tags: [cli, resolve, refactor]
created: 2026-06-11T10:50:23.71484912+02:00
updated: 2026-06-14T18:06:20.070502031+02:00
---

## Background

Cross-model review (Claude reviewing Codex's implementation of 0022) surfaced
three cleanup issues in `planResolve` (`cmd/clinban/resolve.go`):

1. **maxID duplicates `store.NextID`** — `planResolve` manually iterates all
   managed files and calls `strconv.Atoi` to find the maximum ID, then starts
   allocating at `maxID+1`. `store.NextID` already does the same scan. If
   `NextID` semantics ever change (reserved ranges, gap handling), `planResolve`
   silently diverges and can assign IDs that `clinban new` will later reuse.

2. **sort.Slice re-parses IDs on every comparison** — the sort over duplicate
   group IDs calls `strconv.Atoi(ids[i])` and `strconv.Atoi(ids[j])` on each
   comparison, repeating O(k log k) parses of strings already parsed in the
   groups-building loop.

3. **Two-pass loop** — `planResolve` first builds `groups` (one pass over all
   files), then ranges over `groups` to collect duplicate IDs into a slice
   (second pass). A single pass with a size-counter map eliminates the second
   iteration.

## Changes

- Replace the manual `maxID` loop with a call to `st.NextID()` and use its
  return value as `nextID` directly. Remove the `strconv.Atoi` error path that
  currently aborts if any ID is non-numeric (already guaranteed by `idPattern`).
- Pre-convert the duplicate `ids` slice to `[]int` before sorting; replace the
  `sort.Slice` with a straightforward integer sort.
- Fold the duplicate-detection pass into the first `groups`-building loop using
  a count map.

## Acceptance Criteria

- [ ] `planResolve` calls `st.NextID()` instead of reimplementing maxID scan
- [ ] No `strconv.Atoi` calls inside `sort.Slice` comparisons
- [ ] Single loop builds and filters duplicate groups
- [ ] All existing resolve tests pass unchanged
