---
name: kb-lint
description: "Health-check the knowledge base: detect orphan pages, frontmatter violations, tag errors, broken cross-references, contradictions, and coverage gaps. Run periodically or before major ingests. Invoke /kb-lint to audit KB health."
---

# KB Lint

**Role:** KB health auditor.

**Mission:** Systematically review `kb/` for quality issues and produce an actionable report.
Fix Critical issues in-place if the user confirms. Leave Warnings and Suggestions open unless
directed otherwise.

## Activation

Read these first:

1. `KB_RULES.md`
2. `kb/index.md`
3. `kb/tag_dictionary.yaml`

## Lint Checks

Run all checks, then compile a single report.

### 1. Coverage

- `find kb/ -name "*.md" | sort` — list all pages
- Cross-reference against `kb/index.md` — identify pages not listed in the index
- List directories that contain no `.md` files (empty sections)

### 2. Orphans

For each page (excluding `index.md` and `log.md`):
- Check whether any other page links to it (search for its filename in all other `.md` files)
- Pages with zero inbound links from other pages are orphans

### 3. Freshness

- Pages with `status: draft` — flag if `last_updated` is more than 14 days ago
- Pages missing a `last_updated` field entirely

### 4. Frontmatter Compliance

Check every page except `log.md` against required fields from `KB_RULES.md`:

- `title`
- `type`
- `status`
- `summary`
- `last_updated`
- `sources`

Flag any page missing one or more required fields.

### 5. Tag Compliance

- Extract all tags used across pages
- Cross-reference against `kb/tag_dictionary.yaml`
- Flag any tag not in the controlled vocabulary

### 6. Cross-Reference Integrity

- Find `related:` frontmatter entries pointing to files that don't exist
- Find internal links (`[text](path)`) pointing to files that don't exist

### 7. Concept Gaps

Read each page and look for:
- Domain concepts, proper nouns, or system components mentioned repeatedly that lack
  their own KB page
- Concepts that appear in `related:` entries on multiple pages but have no page of their own

### 8. Contradictions

Read pairs of pages with overlapping topics and flag:
- Conflicting claims about the same fact
- Claims on one page that are superseded by newer evidence on another
- `status: stable` pages that conflict with `Confirmed:` claims elsewhere

## Report Format

```markdown
## Critical
- [page] missing required frontmatter fields: [list]
- [page] related: references non-existent file: kb/...
- [page] internal link broken: [target]

## Warnings
- [page] orphan — no inbound links from other pages
- [page] tag violation: "[tag]" not in tag_dictionary.yaml
- [page] stale draft: last_updated [date], status: draft

## Suggestions
- concept "[X]" mentioned in N pages but has no dedicated page
- [page A] and [page B] may contradict on [claim]
- empty directory: kb/[dir]/ — consider whether this section is needed yet

## Summary
Critical: N | Warnings: N | Suggestions: N
```

## After the Report

- Offer to fix **Critical** issues in-place
- For **Warnings** and **Suggestions**: present them for user decision; do not auto-fix

## Log Entry

Always append to `kb/log.md`:

```
## [YYYY-MM-DD] lint | N critical, N warnings, N suggestions
```

## Typical Commands

```bash
find kb/ -name "*.md" | sort
grep -r "related:" kb/ --include="*.md" -n
grep -r "\[.*\](kb/" kb/ --include="*.md" -n
rg "tags:" kb/ --include="*.md" -A5
```
