# Implementation Plan: `clinban resolve`

## Summary

Implement `clinban resolve` to repair duplicate ticket IDs caused by parallel
ticket creation across git clones. The command scans active and archived managed
ticket files, keeps the oldest ticket in each duplicate-ID group, and renames
younger tickets to new IDs above the current repository maximum.

## Key Changes

- Add a new Cobra command `resolve` under `cmd/clinban`.
- Add store-level primitives for managed-file inventory and conflict-safe
  filename renumbering.
- Preserve ticket contents exactly during renumbering. Do not update `updated`,
  because the ticket body/frontmatter is not rewritten and ID lives only in the
  filename.
- Build one deterministic rename plan before applying any filesystem changes:
  collect managed files, group by ID, sort duplicate groups by numeric ID, sort
  each group by `created` then path, keep the first file unchanged, allocate
  new IDs as `max(existing IDs)+1`, and reserve each allocated ID immediately.
- Rename by replacing only the leading four-digit prefix, preserving slug and
  directory.

## Behavior

- Command shape: `clinban resolve`, no arguments, no flags.
- If no duplicate IDs exist, print `no conflicts found` to stdout and exit `0`.
- For each successful rename, print `renamed: <old-path> -> <new-path>` to
  stdout, using paths relative to the project root when possible.
- If a duplicate-group ticket cannot be parsed, exit `1` and rename nothing.
- If any planned destination exists, exit `1` and rename nothing.
- If a filesystem rename fails during application, exit `1` with context.
- Unrelated non-duplicate malformed tickets do not block `resolve`.
- The command fixes filename ID collisions only; it does not promise full
  repository lint cleanliness.

## Implementation Notes

- Keep `internal/store` responsible for filesystem inventory and safe renames.
- Keep sorting by `created`, planning, and command output in CLI code unless a
  cleaner internal helper emerges naturally.
- Add a low-level same-directory rename helper using `os.Link` + `os.Remove`,
  matching existing move behavior, so destination existence is refused
  atomically.
- Suggested helper concepts:
  - managed file record: path, basename, id, inArchive;
  - rename plan item: old path, new path, old id, new id;
  - plan builder: pure logic over inventory and parsed duplicate tickets.

## Tests

- Store/unit tests:
  - inventory includes active and archived managed files;
  - inventory ignores non-managed files;
  - same-directory renumber refuses existing destination;
  - renumber preserves file content byte-for-byte.
- CLI integration tests:
  - no conflicts prints `no conflicts found`, exits `0`;
  - active-only two-file conflict keeps oldest and renumbers younger;
  - archive-only conflict keeps archived tickets archived;
  - active/archive conflict is resolved as one duplicate group;
  - three-file conflict assigns two new sequential IDs;
  - `created` tie uses path ordering deterministically;
  - parse failure in a duplicate group exits `1` and leaves all filenames unchanged;
  - planned destination collision exits `1` and leaves all filenames unchanged;
  - stdout contains one `renamed:` line per changed file.

## Documentation

- Update `docs/cli.md` with `clinban resolve`, output, and exit-code behavior.
- Update `docs/storage.md` to describe duplicate-ID repair and safe renumbering.
- Update `docs/validation.md` to clarify that lint detects collisions while
  resolve repairs them.
- Update `cmd/clinban/schema.md` so generated `SCHEMA.md` tells humans/agents
  to use `clinban resolve` for duplicate filename IDs.
- Append a short entry to `docs/log.md`.

## Assumptions

- `clinban resolve` is intentionally non-interactive in v1.
- Renumbering is a filename-only operation; `updated` is not changed.
- The command repairs duplicate IDs across both active and archived tickets.
- Existing pipeline files were `pipeline/03_design.md` and
  `pipeline/04_tasks.md`; both were removed before this plan was added.
