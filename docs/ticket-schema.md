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
| `title` | yes | string | Author | Non-empty. |
| `status` | yes | string | Clinban/user via `move` | One of `backlog`, `in-progress`, `blocked`, `done`. |
| `type` | yes | string | Author | One of `bug`, `task`, `feature`, `spike`. |
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

## System-Derived Fields

### id

The ticket ID is derived exclusively from the filename's four-digit prefix and is **not stored in frontmatter**. For example, the file `0042-fix-login-timeout.md` has ID `0042`. The store layer injects this value at read time; it is never written to the YAML frontmatter block. Do not add an `id:` line to any ticket file.

## System-Owned Fields

Clinban owns `created` and `updated`. There are two paths for providing these fields:

- **Via `clinban register <path>`:** you may omit `created` and `updated`; the tool sets them for you, overwriting any values already present.
- **Writing directly into the managed directory:** you must include valid RFC 3339 timestamps for both `created` and `updated`. `clinban lint` checks for zero values and rejects any file that is missing these fields or has them set to the zero timestamp.

`status` is initialized by Clinban and normally changed through `clinban move`. Direct file writers can still change it, but `clinban lint` only validates that the value is legal; it does not reconstruct transition history.

## Body

The body is freeform Markdown. Clinban does not interpret body structure.
