---
title: Security Model
kind: reference
scope: security
summary: Captures Clinban's local trust model and filesystem safety assumptions.
updated: 2026-05-19
links:
  - storage
  - cli
  - configuration
---

# Security Model

Clinban is a local CLI for trusted developer workspaces. It does not provide authentication, authorization, or remote service isolation.

## Trust Boundary

The repository and configured ticket directories are assumed to be under the user's control. Malicious local files can still matter because Clinban reads, writes, renames, and removes files.

## Filesystem Safety

Clinban uses same-directory temporary files for writes and renames them into place. Temporary names are randomized.

Archive moves check for destination collisions and refuse to overwrite existing files.

Generated ticket filenames are derived from numeric IDs and sanitized slugs.

## Editor Execution

Interactive commands execute `$EDITOR`, or `vi` if `$EDITOR` is unset. The editor inherits standard input, output, and error.

The editor command is user-controlled environment state. Clinban does not sandbox the editor.

## No Network Surface

Clinban makes no network calls as part of normal operation.

## Out of Scope

- Authentication.
- Multi-user access control.
- Remote execution safety.
- Concurrent writer locking.
- Protection against all malicious repository layouts.
