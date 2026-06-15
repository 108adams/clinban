---
title: Storage
kind: reference
scope: storage
summary: Describes Clinban filesystem layout, ticket discovery, ID scanning, writes, and archiving.
updated: 2026-06-15
links:
  - ticket-schema
  - configuration
  - security
  - validation
---

# Storage

Clinban stores tickets as Markdown files on disk. There is no database.

## Layout

```text
<tickets_dir>/
  0001-first-ticket.md
  0042-fix-login-timeout.md
  archive/
    0003-old-ticket.md
```

Active tickets live in `tickets_dir`. Archived tickets live in `archive_dir`.

## Managed Ticket Discovery

Managed ticket files match:

```text
[0-9]{4}-*.md
```

Non-matching Markdown files are ignored by list/archive scans. This allows README-style files to exist near tickets without being parsed as tickets.

## ID Assignment

Clinban scans active and archived filenames, finds the highest four-digit prefix, and assigns the next integer.

ID uniqueness is enforced across active and archived tickets.

## ID Conflict Resolution

`clinban resolve` repairs duplicate filename IDs across active and archived tickets. It builds a full managed-file inventory, groups files by ID, and only parses files in duplicate groups.

Within each duplicate group, the oldest ticket by `created` timestamp keeps the original ID. Younger tickets are renamed to IDs above the current repository maximum. Active tickets remain in the active directory, archived tickets remain in the archive directory, and ticket contents are not rewritten.

Resolution refuses planned destination collisions before applying renames. Each rename is performed within the same directory and refuses to overwrite an existing file.

## Batch rename (`BatchRenameWithinDir`)

All renames planned by `resolve` are applied through a single store call,
`BatchRenameWithinDir`, which guarantees all-or-nothing semantics over the whole
batch. Each rename stays within its source file's directory.

The batch runs in two phases over hard links (`link` + `remove`, the same
TOCTOU-safe idiom as single moves):

1. **Phase 1 — Link.** Create every destination hard link. If any link fails,
   the already-created destinations are removed and the batch aborts. No source
   was touched, so the filesystem is unchanged.
2. **Phase 2 — Remove.** Delete every source name. On success each ticket now
   exists only under its new name.

If a Phase-2 removal fails partway, rollback runs **restore-then-cleanup**:

- **Restore first.** Re-link every already-removed source from its destination
  hard link. After `remove(old)` succeeds, the destination is the inode's *only*
  remaining name, so it must be re-linked back to the old path **before** the
  destination is touched — otherwise removing the destination would permanently
  delete the ticket.
- **Cleanup second.** Remove all Phase-1 destination links.

Both rollback steps are best-effort: every item is attempted even if an earlier
one fails. Failures are collected on the returned `BatchError`. When rollback
completes cleanly the batch leaves zero net change; when it does not,
`BatchError.Inconsistent()` reports `true`, signalling the filesystem may need
manual inspection. The CLI surfaces these as `resolve:` and `resolve: rollback:`
error lines — see the [resolve command](cli.md#clinban-resolve).

## Writes

Ticket writes use a same-directory temporary file created with a random `.clinban-*.tmp` name, then rename into place.

This prevents readers from observing a partially written final file during normal operation and avoids predictable temp-name symlink attacks.

## Archive

Archiving moves done tickets from the active directory to the archive directory.

Archive and active moves refuse to overwrite an existing destination filename.

## Reopen

The valid reopen path is `done` to `backlog`. When reopening from archive, Clinban writes the updated active ticket first and removes the archived copy only after the write succeeds.
