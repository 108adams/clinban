---
title: Validation
kind: reference
scope: validation
summary: Explains parse errors, lint rules, and workflow transition enforcement.
updated: 2026-05-19
links:
  - ticket-schema
  - cli
  - architecture
---

# Validation

Clinban validation has two phases: parsing and linting.

## Parse Errors

Parsing checks whether a file can be read as a Markdown ticket with YAML frontmatter. Parse failures include missing frontmatter fences and malformed YAML.

Lint cannot run on a ticket that failed to parse.

## Lint Rules

`clinban lint` validates parsed tickets against the schema.

Rules run in order:

1. Required fields are present: `id`, `status`, `title`, `type`, `created`, `updated`.
2. `status` is valid.
3. `type` is valid.
4. `id` matches the numeric filename prefix.
5. `created` and `updated` are valid timestamps.
6. `tags`, if present, is a list of non-empty strings.
7. `id` is unique across active and archived tickets.

Lint errors are printed one per line:

```text
0042-fix-login-timeout.md: field 'type': required field missing
```

## Transition Enforcement

Workflow transitions are enforced only by `clinban move`.

Valid transitions:

| From | To |
|---|---|
| `backlog` | `in-progress` |
| `backlog` | `blocked` |
| `in-progress` | `blocked` |
| `in-progress` | `done` |
| `blocked` | `in-progress` |
| `done` | `backlog` |

All other transitions are rejected by the CLI. Direct file edits are not transition-checked.

## Automata Contract

Automata can write ticket files directly. Their safety net is `clinban lint`, not the CLI state machine.
