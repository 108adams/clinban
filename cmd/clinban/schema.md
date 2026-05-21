# Clinban Schema Reference

This document is the authoritative reference for Clinban ticket files. It is
intended for LLM agents and automated tools that create, read, update, or move
tickets. Read every section before performing any operation.

## 1. What is Clinban

Clinban is a local, file-based Kanban tool. Tickets are plain Markdown files
with YAML frontmatter stored on disk. There is no database or network service.

**How to find the ticket directories:**

Read the `.clinban` file in the repository root. It is a plain-text key=value
file with the following entries:

```
tickets_dir = "tickets"
archive_dir = "tickets/archive"
```

- `tickets_dir` — path to active tickets (relative to the repository root, or
  absolute). All operations on open tickets read from and write to this
  directory.
- `archive_dir` — path to archived (done) tickets. Archived tickets have been
  moved here by `clinban archive`.

Always resolve these paths before reading or writing any ticket file.

---

## 2. Ticket Format

A ticket is a Markdown file whose content begins with a YAML frontmatter block
delimited by `---`. The body (below the closing `---`) is freeform Markdown and
is not interpreted by Clinban.

Complete example:

```markdown
---
title: "Fix login timeout on staging"
id: "0042"
status: "in-progress"
type: "bug"
tags: ["auth", "backend"]
created: "2026-05-18T14:30:00Z"
updated: "2026-05-18T15:00:00Z"
---

## Description

Describe the problem here. Markdown is supported.

## Acceptance criteria

- [ ] Reproduction steps documented
- [ ] Root cause identified
- [ ] Fix deployed to staging
```

All seven frontmatter fields must be present. Do not add extra frontmatter
fields; Clinban ignores them but they clutter the file.

---

## 3. Fields

| Field | Required | Owner | Constraints |
|-------|----------|-------|-------------|
| `id` | required | Clinban (tool) | Four zero-padded decimal digits: `0001`–`9999`. Unique across active and archived tickets. Never set or change this field manually. |
| `status` | required | Clinban (tool) via `clinban move` | One of: `backlog`, `in-progress`, `blocked`, `done`. Change only through `clinban move`; see Section 5 for valid transitions. The new-ticket template includes a `# states: backlog, in-progress, blocked, done` hint comment below this field. |
| `type` | required | Author | One of: `bug`, `task`, `feature`, `spike`. |
| `title` | required | Author | Non-empty string. |
| `tags` | optional | Author | YAML sequence of strings. Use an empty sequence `[]` when there are no tags. |
| `created` | required | Clinban (tool) | RFC 3339 timestamp (e.g. `2026-05-18T14:30:00Z`). Set by Clinban on creation or registration. Do not change this field. |
| `updated` | required | Clinban (tool) | RFC 3339 timestamp. Refreshed by Clinban on every write. Do not change this field. |

**Field ownership rules:**

- Fields owned by Clinban (`id`, `created`, `updated`) are set and maintained
  by the tool. If you write a ticket file directly, omit those fields and let
  Clinban fill them in — or use `clinban register` which will overwrite them.
- `status` is initialized by Clinban and must only be changed through
  `clinban move`. Direct edits to `status` bypass transition validation.
- `type`, `title`, and `tags` are owned by the author and may be edited freely.

---

## 4. File Naming

Active ticket filenames follow this pattern:

```
<id>-<slug>.md
```

Where:

- `<id>` is the four-digit zero-padded ticket ID (e.g. `0042`).
- `<slug>` is derived from the first five meaningful words of the title. Words
  are lowercased, stripped to ASCII letters and digits only, and joined with
  hyphens.

Examples:

```
0001-first-ticket.md
0042-fix-login-timeout-on-staging.md
0100-add-oauth2-support.md
```

Files in `tickets_dir` whose name does not match the pattern `[0-9]{4}-*.md`
are ignored by all Clinban commands. Do not rename ticket files manually;
rename through `clinban move` or by creating a new ticket with `clinban new`.

Archived tickets use the same filename pattern and are stored in `archive_dir`.

---

## 5. Status Transitions

`status` must progress along this directed graph. Only the six edges listed
below are valid; any other transition is rejected by `clinban move`.

```
backlog ──► in-progress ──► done ──► backlog
  │              │
  │              ▼
  └──────────► blocked ──► in-progress
```

Valid transitions:

| From | To | Meaning |
|------|----|---------|
| `backlog` | `in-progress` | Work has started |
| `backlog` | `blocked` | Ticket is blocked before work begins |
| `in-progress` | `blocked` | Work is stalled on an external dependency |
| `in-progress` | `done` | Work is complete |
| `blocked` | `in-progress` | Blocker resolved; work resuming |
| `done` | `backlog` | Ticket reopened for additional work |

No other transitions are permitted. For example, you cannot move directly from
`backlog` to `done`, or from `blocked` to `done`.

---

## 6. Agent Operations

Follow these steps exactly. Do not skip or reorder steps.

### 6.1 Create a ticket

1. Read `.clinban` to find `tickets_dir`.
2. Run `clinban new --no-interactive --title "<title>" --type <type> [--tags tag1,tag2] [--body "<body>"]`.
   - `--no-interactive` is required for non-interactive/agent use; without it the
     command opens `$EDITOR` and blocks.
   - `<type>` must be one of: `bug`, `task`, `feature`, `spike`. If `default_type`
     is set to a valid type in `.clinban`, `--type` may be omitted.
   - `--tags` is optional and comma-separated.
   - `--body` is optional Markdown body text.
3. The command prints the new ticket's filename to stdout (e.g.
   `created: 0043-my-new-ticket.md`). Record the ID from that filename.

### 6.2 Update a ticket

1. Run `clinban edit <id>`.
2. Clinban opens the ticket in `$EDITOR` on a temporary copy.
3. Edit the desired frontmatter fields (`type`, `title`, `tags`) or the body.
4. Save and close the editor. Clinban validates the result and writes the ticket
   only if parse and lint pass; on failure it prompts to reopen.
5. Do not modify `id`, `created`, or `updated` — Clinban owns these fields and
   refreshes `updated` automatically on every write.

### 6.3 Move status

1. Verify the transition is valid using the table in Section 5.
2. Run `clinban move <id> <new-status>`.
   - `<id>` may be the full four-digit form (`0042`) or the short numeric form
     (`42`); Clinban normalizes it.
3. If the transition is invalid, `clinban move` exits non-zero with a
   descriptive error. Do not attempt to edit the `status` field in the file
   directly.

### 6.4 Archive a ticket

Archiving moves a `done` ticket from `tickets_dir` to `archive_dir`.

1. Confirm the ticket's `status` is `done`. If it is not, move it to `done`
   first (see Section 6.3).
2. Run `clinban archive <id>`.
3. Clinban moves the file to `<archive_dir>/<filename>`. The file is removed
   from `tickets_dir`.
4. Archived tickets are read-only from the perspective of normal operations.
   To reopen an archived ticket, run `clinban move <id> backlog`, which moves
   it back to `tickets_dir` with `status: backlog`.

### 6.5 Remove a ticket

Removing permanently deletes the ticket file from disk (active or archive
directory). Use this only when a ticket should be fully discarded.

1. Run `clinban remove <id>`.
2. Clinban deletes the file. The ID is freed for reuse (though `clinban new`
   will not reuse it — it always takes `max+1`).
3. If no file matches the ID, Clinban exits 1 with `ticket not found`.
4. If multiple files share the ID (a collision), Clinban exits 1, lists all
   colliding filenames, and suggests running `clinban lint` to diagnose.
