---
title: ADR 0003 Atomic File Writes
kind: decision
scope: storage
summary: Records the same-directory temporary-file and rename strategy for ticket writes.
updated: 2026-05-19
links:
  - storage
  - security
---

# ADR 0003: Atomic File Writes

## Decision

Write ticket files by creating a random temporary file in the target directory, writing and closing it, then renaming it to the final path.

## Context

Tickets are the source of truth for work state. A partial write would corrupt visible repository state for humans, agents, and CI.

## Rejected Alternatives

| Option | Rejected because |
|---|---|
| Direct write to final path | Interrupted writes can leave corrupt final files visible. |
| Temp file in system `/tmp` | Cross-device rename may not be atomic. |
| Predictable `path + ".tmp"` | Pre-created symlinks or stale temp files can create safety issues. |

## Consequences

- Readers do not observe partially written final files during normal operation.
- Temporary files may remain after a crash.
- Write durability after OS crash is not guaranteed beyond normal filesystem behavior.
