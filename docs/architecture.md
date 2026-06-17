---
title: Architecture
kind: architecture
scope: architecture
summary: Describes Clinban's package map, dependency boundaries, and major design responsibilities.
updated: 2026-06-17
links:
  - cli
  - clinban-board
  - ticket-schema
  - storage
  - validation
  - security
---

# Architecture

Clinban is a Go CLI with small internal packages. The package boundaries keep schema parsing, storage, validation, and workflow transitions independently testable.

## Package Map

| Package | Responsibility |
|---|---|
| `cmd/clinban` | Cobra command wiring, flag parsing, user-facing CLI behavior (including the `board` TUI entry point). |
| `internal/ticket` | Ticket struct, status/type constants, Markdown/YAML parse and marshal. |
| `internal/store` | Filesystem storage, ID scanning, reads, writes, active/archive moves. |
| `internal/board` | Canonical board display ordering (status rank, then numeric ID), shared by `clinban list` and the TUI. |
| `internal/lint` | Schema validation on parsed tickets. No filesystem dependency. |
| `internal/fsm` | Workflow transition table and validation. |
| `internal/config` | `.clinban` loading and path resolution. |
| `internal/editor` | `$EDITOR` invocation: `Command` builds the `*exec.Cmd` (stdio unset); `Open` runs it blocking. |
| `internal/tui` | Bubble Tea model for the `clinban board` terminal UI. A pure consumer of store/fsm/lint/editor — no new ticket-truth logic. |
| `internal/slug` | Title-to-filename slug generation. |
| `internal/template` | Embedded interactive ticket template rendering. |

## Dependency Boundaries

`internal/ticket` is the schema boundary. It does not import store.

`internal/lint` validates parsed tickets and receives repository context from callers. It does not import store.

`internal/fsm` validates status transitions. It does not import store or command packages.

`internal/store` owns filesystem behavior but does not decide workflow validity.

`internal/board` depends only on `internal/ticket`. It holds the display order and is imported by both `cmd/clinban` (list) and `internal/tui` (board) so the ordering never diverges; it does not import store.

`internal/tui` is a pure consumer at the UI boundary: it reads through `internal/store` and mutates only through `store.WriteTicket` + `internal/fsm`, never touching the filesystem directly except for an edit scratch copy. It introduces no ticket-truth logic, mirroring the CLI handlers.

## Validation Flow

```text
file bytes -> ticket.Parse -> lint.Lint -> command/store action
```

Parse errors and lint errors are separate categories.

## CLI Flow

Command handlers coordinate internal packages:

- Load config and store at command startup.
- Resolve ticket files through store.
- Parse tickets through ticket.
- Validate schema through lint.
- Validate transitions through fsm.
- Write or move files through store.

## Decisions

Architecture decisions are recorded under [ADRs](adr/0001-cli-framework.md).
