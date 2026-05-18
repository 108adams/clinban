---
name: kb-import
description: "Knowledge-base ingestion workflow for this repo. Use when importing a new topic into top-level kb/, migrating enduring knowledge out of docs or cc, or refining an existing kb page with code-vs-doc reconciliation. Invoke /kb-import before future wiki imports."
---

# KB Import

**Role:** Repo-specific knowledge-base maintainer.

**Mission:** Import a topic into `kb/` using the established workflow:
gather sources, synthesize a canonical page, add source notes, refine against
live code, then update KB navigation and history.

## Activation

Read these first:

1. `KB_RULES.md`
2. `kb/index.md`
3. `kb/tag_dictionary.yaml`
4. `kb/log.md`

Then identify the topic and the likely target page:

- domain overview: `kb/domains/<topic>.md`
- architecture topic: `kb/architecture/<topic>.md`
- operations topic: `kb/operations/<topic>.md`
- workflow topic: `kb/workflows/<topic>.md`
- glossary term: `kb/glossary/<term>.md`

## Source Hierarchy

Use sources in this order of authority:

1. Current code
2. Stable formal docs in `docs/`
3. Focused tests when they clarify current behavior
4. `cc/` memory, analyses, and historical artifacts

Do not flatten disagreements. Record them.

## Standard Import Flow

### Phase 1: Gather

Collect:

- formal docs for the topic if they exist
- relevant `cc/` memory or historical analysis
- code files that currently enforce the behavior
- tests if they reveal current rules or stale assumptions

Prefer broad entry points first, then narrow to implementation anchors.

### Phase 2: Synthesize

Create:

1. one canonical KB page for the topic
2. one source note per meaningful source cluster

The canonical page should favor:

- `Confirmed from code`
- `Confirmed from code and docs`
- `Inferred`
- `Conflicts with`
- `Needs attention`

Do not mirror source documents one-to-one. Synthesize.

### Phase 3: Refine

After the first draft, perform a refinement pass:

- read deeper implementation files or dedicated docs pages
- tighten over-broad claims
- surface code-vs-doc drift
- add missing source anchors
- lower confidence or mark conflicts where needed

This refinement step is required. The first draft is not the final ingest.

### Phase 4: Maintain the KB

Every import must also:

- update `kb/tag_dictionary.yaml` if new controlled tags are genuinely needed
- update `kb/index.md`
- append an entry to `kb/log.md`

## Frontmatter Rules

Follow `KB_RULES.md` exactly.

Required fields:

- `title`
- `type`
- `status`
- `summary`
- `last_updated`
- `sources`

Optional fields to use when valuable:

- `source_of_truth`
- `tags`
- `related`
- `confidence`

Tags must come from `kb/tag_dictionary.yaml`.

## Page Quality Rules

- Prefer current behavior over historical intent.
- Keep canonical pages scoped to enduring knowledge.
- Move test progress, phase tracking, and execution history into source notes or
  leave them in `cc/`.
- If a source is stale but historically useful, preserve it as a source note and
  mark the drift explicitly.
- When permissions vary by view or path, describe the nuance rather than forcing
  a fake single rule.

## Memory Phase-Out

When a KB page clearly supersedes a `cc/MEMORY` file, do not delete the memory
file immediately.

Use this sequence:

1. demote the memory file
2. add a pointer to the canonical `kb/` page
3. keep only narrow residual testing or historical context
4. update `cc/MEMORY/MEMORY.md`

## Output Checklist

Before finishing, verify:

- canonical page exists
- source notes exist
- frontmatter is complete
- tags are valid
- `kb/index.md` updated
- `kb/log.md` updated
- major conflicts called out explicitly
- any superseded `cc/MEMORY` page has been demoted, if requested

## Typical Commands

Use fast repo inspection:

- `rg --files docs cc <app_or_dir>`
- `rg -n "<pattern>" <paths> -g '*.py' -g '*.rst' -g '*.md'`
- `sed -n '1,260p' <file>`

Prefer reading a few strong source files over bulk-loading large trees.
