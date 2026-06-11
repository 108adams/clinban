---
title: Add conflict resolution command
status: done
type: feature
tags: [cli, storage, conflicts]
created: 2026-06-10T18:56:21+02:00
updated: 2026-06-10T19:22:10.820122875+02:00
---

# Add `clinban resolve`

## Problem

Multiple users can create tickets independently in separate clones of the same repository. After syncing through git, two or more different ticket files can have the same four-digit filename prefix.

Example:

```text
tickets/0023-add-tui.md
tickets/0023-fix-parser.md
```

Clinban treats the numeric prefix as the ticket ID, so this creates an ID collision. Existing lint detects the problem, but the user still has to resolve it manually.

## Desired Outcome

`clinban resolve` rewrites filenames so active and archived ticket IDs are unique again.

After the command finishes successfully:

- no active or archived managed ticket files share the same ID;
- older conflicting tickets keep their existing ID;
- younger conflicting tickets are renamed to the next available IDs;
- ticket frontmatter remains otherwise unchanged except `updated`, if the implementation decides the rename should count as a Clinban write;
- no destination file is overwritten.

## Proposed Behavior

Add:

```text
clinban resolve
```

The command scans both configured ticket locations:

- `tickets_dir`
- `archive_dir`

It should consider only managed ticket filenames matching the same pattern used by existing store/list/lint behavior.

For each duplicated ID group:

1. Parse every file in the group.
2. Sort the group by `created` ascending.
3. Keep the oldest ticket at the original ID.
4. Rename every later ticket to the next available four-digit ID.
5. Continue until all duplicate ID groups are resolved.

The algorithm should handle groups larger than two directly. Do not implement this as pair-only logic. A group like `0023-a.md`, `0023-b.md`, `0023-c.md` should keep the oldest as `0023` and assign new IDs to the other two in creation order.

Recommended implementation model:

1. Build a full inventory of active and archived managed tickets.
2. Build the set of currently occupied IDs.
3. Find duplicate groups from the inventory.
4. For each duplicate group, ordered by numeric ID:
   - parse and lint enough to trust `created`;
   - sort by `created`, then by path as a deterministic tie-breaker;
   - keep the first file unchanged;
   - for each remaining file, allocate `max(existing IDs)+1`, zero-pad to four digits, and reserve it immediately.
5. Execute planned renames only after all rename destinations have been checked for collisions.

This avoids repeatedly refreshing the file list after each pair. The command can compute one deterministic rename plan from a single inventory, then apply it.

## Edge Cases

- If more than two files share one ID, resolve all of them in one pass.
- If `created` timestamps tie, use stable path ordering as the tie-breaker and report that tie in verbose output if such output exists.
- If a conflicting ticket cannot be parsed, fail without renaming anything.
- If a planned destination already exists, fail without renaming anything.
- If active and archived tickets share an ID, include both in the same collision group.
- If an archived ticket is renumbered, it stays archived.
- If an active ticket is renumbered, it stays active.
- If there are no conflicts, exit `0` and print a short "no conflicts found" message.

## Output

For changed tickets, print one line per rename:

```text
renamed: tickets/0023-fix-parser.md -> tickets/0024-fix-parser.md
```

Exit codes:

- `0` when no conflicts exist or all conflicts are resolved;
- `1` when parsing, validation, destination collision, or filesystem rename fails.

## Acceptance Criteria

- `clinban resolve` detects duplicate IDs across active and archived tickets.
- The oldest ticket in each duplicate group keeps the original ID.
- Every younger ticket in the group receives the next available ID.
- Groups of three or more conflicting tickets are resolved correctly.
- The command does not overwrite existing files.
- The command leaves active tickets active and archived tickets archived.
- The command fails before making changes if any conflicting ticket cannot be parsed.
- Tests cover active-only conflicts, archive-only conflicts, active/archive conflicts, and a group with at least three files sharing one ID.
- Documentation is updated:
  - `docs/cli.md`
  - `docs/storage.md` or `docs/validation.md` if resolution behavior affects those pages
  - `cmd/clinban/schema.md` if generated agent guidance should mention conflict resolution
  - `docs/log.md`
