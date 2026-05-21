---
title: Ticket Schema
kind: reference
scope: tickets
summary: Defines Clinban ticket files, YAML frontmatter fields, filename convention, and ownership rules.
updated: 2026-05-21
links:
  - cli
  - validation
  - storage
  - product
---

# Ticket Schema

A Clinban ticket is a Markdown file with YAML frontmatter. The frontmatter is the contract shared by humans, CLI commands, scripts, CI jobs, and LLM agents. The Markdown body is freeform.

## Frontmatter

```yaml
---
title: "Fix login timeout"
id: "0042"
status: "in-progress"
type: "bug"
tags: ["auth", "backend"]
created: "2026-05-18T14:30:00Z"
updated: "2026-05-18T15:00:00Z"
---

Markdown body goes here.
```

| Field | Required | Type | Owner | Constraint |
|---|---:|---|---|---|
| `id` | yes | string | Clinban | Four zero-padded digits, unique across active and archived tickets. |
| `status` | yes | string | Clinban/user via `move` | One of `backlog`, `in-progress`, `blocked`, `done`. |
| `type` | yes | string | Author | One of `bug`, `task`, `feature`, `spike`. |
| `title` | yes | string | Author | Non-empty. |
| `tags` | no | list of strings | Author | Free-form labels; empty list is valid. |
| `created` | yes | RFC3339 timestamp | Clinban | Set on creation or registration. |
| `updated` | yes | RFC3339 timestamp | Clinban | Refreshed on Clinban writes. |

## File Naming

Managed ticket filenames use:

```text
<id>-<slug>.md
```

Example:

```text
0042-fix-login-timeout.md
```

The slug is derived from the first five meaningful words of the title. Tokens are lowercased, stripped to ASCII letters and digits, and joined with hyphens.

## System-Owned Fields

Clinban owns `id`, `created`, and `updated`. On creation or registration, external values for those fields are ignored or overwritten.

`status` is initialized by Clinban and normally changed through `clinban move`. Direct file writers can still change it, but `clinban lint` only validates that the value is legal; it does not reconstruct transition history.

## Body

The body is freeform Markdown. Clinban does not interpret body structure.
