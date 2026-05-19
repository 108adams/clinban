---
title: ADR 0001 CLI Framework
kind: decision
scope: architecture
summary: Records the decision to use Cobra for Clinban command routing and flag parsing.
updated: 2026-05-19
links:
  - architecture
  - cli
---

# ADR 0001: CLI Framework

## Decision

Use Cobra for command routing, flag parsing, help text, and command structure.

## Context

Clinban has multiple subcommands with distinct flags and behavior. A small framework avoids custom routing and help generation.

## Alternatives

| Option | Rejected because |
|---|---|
| Standard library `flag` | No subcommand routing and too much custom help/dispatch code. |
| `urfave/cli` | Less aligned with common Go infrastructure tooling than Cobra. |

## Consequences

- Commands are implemented as `*cobra.Command`.
- Help and flag parsing are consistent.
- Cobra is an external dependency.
- Cobra version should be pinned by module dependencies.
