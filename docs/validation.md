---
title: Validation
kind: reference
scope: validation
summary: Explains parse errors, lint rules, and workflow transition enforcement.
updated: 2026-05-21
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

1. **Required fields are present:** `id`, `status`, `title`, `type`, `created`, `updated` are all non-zero.
   - `id` is not a frontmatter field. It is derived from the ticket file's four-digit filename prefix (e.g. `0042-fix-login.md` → `"0042"`) and injected by `store.ReadTicket` before lint runs. `ruleRequiredFields` checks that this injection happened — if `t.ID` is empty, lint reports a missing `id`.
   - `created` and `updated` are checked for zero values here. A zero timestamp means the field was absent or could not be parsed as RFC 3339.
2. **`status` is valid:** must be one of `backlog`, `in-progress`, `blocked`, `done`. Skipped if `status` is empty (rule 1 already flags that).
3. **`type` is valid:** must be one of `bug`, `task`, `feature`, `spike`. Skipped if `type` is empty (rule 1 already flags that).
4. **`tags` contains only non-empty strings:** each element in the `tags` list must be a non-empty, non-blank string.
5. **`id` is unique:** `t.ID` must appear exactly once across all active and archived tickets.

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
