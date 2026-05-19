---
title: ADR 0002 Package Decomposition
kind: decision
scope: architecture
summary: Records the separation of ticket model, store, lint, and FSM packages.
updated: 2026-05-19
links:
  - architecture
  - validation
  - ticket-schema
---

# ADR 0002: Package Decomposition

## Decision

Separate the ticket model, filesystem store, lint engine, and state machine into distinct internal packages.

## Context

The ticket schema is the external contract for both humans and automata. Lint is the safety net for machine-written files and must be testable without a real filesystem.

## Consequences

- `internal/ticket` owns parsing and marshaling.
- `internal/store` owns filesystem behavior.
- `internal/lint` owns schema validation and receives repository context from callers.
- `internal/fsm` owns transition validation.
- `lint`, `fsm`, and `ticket` must not import `store`.

This keeps core rules testable and prevents filesystem details from becoming implicit schema behavior.
