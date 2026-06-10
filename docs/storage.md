---
title: Storage
kind: reference
scope: storage
summary: Describes Clinban filesystem layout, ticket discovery, ID scanning, writes, and archiving.
updated: 2026-06-10
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

## Writes

Ticket writes use a same-directory temporary file created with a random `.clinban-*.tmp` name, then rename into place.

This prevents readers from observing a partially written final file during normal operation and avoids predictable temp-name symlink attacks.

## Archive

Archiving moves done tickets from the active directory to the archive directory.

Archive and active moves refuse to overwrite an existing destination filename.

## Reopen

The valid reopen path is `done` to `backlog`. When reopening from archive, Clinban writes the updated active ticket first and removes the archived copy only after the write succeeds.
