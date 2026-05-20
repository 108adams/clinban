---
name: librarian
description: Maintain the Clinban project wiki under docs/. Use when creating, ingesting, updating, linting, or reorganizing documentation — including after behavior changes, schema updates, or architecture decisions. Invoke /librarian before any non-trivial docs work.
---

# Librarian

<!-- Adapted from /home/adam/.codex/skills/librarian/SKILL.md for Claude Code -->

## Purpose

Maintain the living project wiki under `docs/`. The wiki distills current project knowledge into stable, linked Markdown pages — it is not a document dump or a planning artifact store.

## Three Layers

- **Sources** — implementation diffs, planning notes, conversations, specs. Read them; do not treat them as final wiki content.
- **Wiki** — maintained pages under `docs/`. This is the source of truth.
- **Schema** — `docs/schema.md` defines page conventions. Read it before any non-trivial update.

## First Files

Every wiki must have:

- `docs/schema.md` — page frontmatter conventions, link rules, index/log format.
- `docs/index.md` — content-oriented map of all pages.
- `docs/log.md` — chronological maintenance log.

Read `docs/schema.md` first. If it does not exist, create it before ingesting anything.

## Workflow

### 1. Read the request and sources

- Identify the operation: scaffold, ingest, update, lint, or reorganize.
- Use `Read` to read relevant source files. Prefer current implementation over older planning notes when they conflict.

### 2. Plan the update

- Decide which existing pages to update.
- Create a new page only when the concept is stable, reusable, and too large for an existing page.
- Keep planning artifacts out of the wiki unless distilled into current truth.

### 3. Write semantically

- Start each page with minimal YAML frontmatter (see below).
- Use clear headings and concise prose.
- Prefer current behavior and durable contracts over task history.
- Add cross-links using relative Markdown links.

### 4. Update navigation

- Update `docs/index.md` whenever a page is added, removed, renamed, or materially repurposed.
- Append one entry to `docs/log.md` for each coherent update pass.

### 5. Retire sources when requested

- Delete source files only after reading and accounting for their content.
- Do not delete sources you have not read.

### 6. Lint the wiki

- Check for missing frontmatter, broken links, duplicate page purpose, stale statements, orphan pages, and contradictions.
- Keep fixes small and direct.

## Frontmatter

```yaml
---
title: Ticket Schema
kind: reference
scope: tickets
summary: Defines the Markdown/YAML ticket format used by Clinban.
updated: 2026-05-19
links:
  - cli
  - validation
---
```

Required fields:

- `title` — human-readable page title.
- `kind` — one of `overview`, `reference`, `architecture`, `decision`, `workflow`, `log`, `schema`.
- `scope` — short lowercase area: `project`, `cli`, `tickets`, `storage`, `validation`, `docs`.
- `summary` — one sentence describing why the page exists.
- `updated` — date of last meaningful content update, `YYYY-MM-DD`.
- `links` — related doc slugs, or `[]`.

## Ingestion Rules

- Distill, do not copy.
- One stable page per durable concept.
- Convert "planned/proposed" wording to current behavior only when implementation confirms it.
- Preserve unresolved questions explicitly only if they still matter.
- Move ADR-like decisions into `docs/adr/` when that directory exists.
- Delete migrated source files only when the user requested deletion and migration is complete.

## Log Entry Format

```markdown
## [YYYY-MM-DD] ingest | Short title

- Source: `path/or/conversation`
- Updated: `docs/page.md`, `docs/other.md`
- Notes: One concise sentence.
```

Use `update`, `lint`, `decision`, or `scaffold` instead of `ingest` when more accurate.

## Tools

Prefer:

- `Read` — read wiki and source files.
- `Edit` — update existing pages in place.
- `Write` — create new pages.
- `Glob` / `Bash` with `find` — locate pages and orphans.
- `Grep` / `Bash` with `rg` — search for cross-link targets, stale references, or missing frontmatter.

Avoid bulk-loading large file trees. Read a few strong sources and verify against current code.
