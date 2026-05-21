---
title: Configuration
kind: reference
scope: configuration
summary: Describes .clinban configuration, project root discovery, and ticket directory defaults.
updated: 2026-05-21
links:
  - cli
  - storage
  - product
---

# Configuration

Clinban configuration lives in a `.clinban` TOML file at the project root. Use [`clinban init`](cli.md#clinban-init) to create the file and directories in one step.

## File Format

```toml
tickets_dir = "tickets"
archive_dir = "tickets/archive"
default_type = "task"
```

All fields are optional.

## Fields

| Field | Default | Description |
|---|---|---|
| `tickets_dir` | `tickets` | Directory for active tickets. |
| `archive_dir` | `<tickets_dir>/archive` | Directory for archived tickets. |
| `default_type` | _(none)_ | Pre-fills the `type` field when creating a ticket. Must be one of `bug`, `task`, `feature`, `spike`; ignored if invalid. |

## Defaults

If `.clinban` is absent:

- `tickets_dir` defaults to `tickets/` inside the project root.
- `archive_dir` defaults to `<tickets_dir>/archive`.
- `default_type` is unset; `--type` is required for `clinban new --no-interactive`.

If only `tickets_dir` is set, `archive_dir` defaults to `<tickets_dir>/archive`.

## Path Resolution

Relative paths are resolved against the project root. Absolute paths are used as provided.

## Project Root Discovery

The CLI walks upward from the current working directory looking for `.clinban`. If no config file is found, the current working directory is treated as the project root.

## Malformed Config

If `.clinban` exists but cannot be parsed as TOML, Clinban exits with an error.
