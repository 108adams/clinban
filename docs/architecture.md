---
title: Architecture
kind: architecture
scope: architecture
summary: Describes Clinban's package map, dependency boundaries, and major design responsibilities.
updated: 2026-05-19
links:
  - cli
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
| `cmd/clinban` | Cobra command wiring, flag parsing, user-facing CLI behavior. |
| `internal/ticket` | Ticket struct, status/type constants, Markdown/YAML parse and marshal. |
| `internal/store` | Filesystem storage, ID scanning, reads, writes, active/archive moves. |
| `internal/lint` | Schema validation on parsed tickets. No filesystem dependency. |
| `internal/fsm` | Workflow transition table and validation. |
| `internal/config` | `.clinban` loading and path resolution. |
| `internal/editor` | `$EDITOR` process invocation. |
| `internal/slug` | Title-to-filename slug generation. |
| `internal/template` | Embedded interactive ticket template rendering. |

## Dependency Boundaries

`internal/ticket` is the schema boundary. It does not import store.

`internal/lint` validates parsed tickets and receives repository context from callers. It does not import store.

`internal/fsm` validates status transitions. It does not import store or command packages.

`internal/store` owns filesystem behavior but does not decide workflow validity.

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
