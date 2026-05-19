---
title: Documentation Schema
kind: schema
scope: docs
summary: Defines the lightweight wiki conventions used to maintain Clinban documentation for humans and LLMs.
updated: 2026-05-19
links:
  - index
  - log
---

# Documentation Schema

Clinban documentation is a lightweight project wiki under `docs/`. It serves one human maintainer and one LLM collaborator, so the process should stay small, explicit, and easy to update.

## Layers

- **Sources**: planning files, reviews, implementation diffs, conversations, and specs. Sources are read and distilled.
- **Wiki**: maintained Markdown pages under `docs/`. These pages describe current project truth.
- **Schema**: this page plus the `librarian` skill. They define how the wiki is maintained.

## Frontmatter

Each wiki page starts with minimal YAML frontmatter:

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

Fields:

- `title`: human-readable page title.
- `kind`: one of `overview`, `reference`, `architecture`, `decision`, `workflow`, `log`, `schema`.
- `scope`: short lowercase area such as `project`, `cli`, `tickets`, `storage`, `validation`, or `docs`.
- `summary`: one sentence explaining why the page exists.
- `updated`: date of last meaningful content update in `YYYY-MM-DD`.
- `links`: related document slugs without `.md`; use `[]` when none apply.

Avoid process-heavy metadata such as author, reviewer, approval state, or ownership.

## Page Rules

- Write for current truth, not historical planning.
- Distill source material instead of copying it.
- Prefer one stable page per durable concept.
- Use relative Markdown links in body text.
- Keep pages concise and strongly headed.
- Record durable decisions in `docs/adr/`.

## Navigation

`index.md` is the content map. Update it when pages are added, removed, renamed, or materially repurposed.

`log.md` is the chronological maintenance record. Append to it for each coherent documentation update.

## Source Retirement

When source files are imported and the user wants them retired, remove only files whose knowledge has been migrated or intentionally discarded.
