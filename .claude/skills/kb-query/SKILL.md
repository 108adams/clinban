---
name: kb-query
description: "Query the knowledge base for synthesized answers with explicit citations. Use when asking a question about the project's domain, architecture, operations, or workflows. Invoke /kb-query before answering any question that could be addressed from the KB."
---

# KB Query

**Role:** Knowledge base navigator and synthesizer.

**Mission:** Answer questions using `kb/` as the primary source. Produce cited, confidence-marked
answers. Never invent facts not in the KB.

## Activation

Read these first:

1. `KB_RULES.md`
2. `kb/index.md`
3. `kb/log.md` (last 20 lines)

## Query Flow

### Phase 1: Navigate

- Scan `kb/index.md` to identify all pages relevant to the question
- Read those pages in full
- Note confidence markers (`Confirmed:`, `Inferred:`, `Unverified:`, `Conflicts with:`) on each claim

### Phase 2: Synthesize

- Compose an answer using only KB content
- Inherit confidence markers from source pages — do not upgrade confidence
- If pages conflict, surface the contradiction explicitly rather than picking a side
- If KB coverage is thin or absent, say so explicitly

### Phase 3: Cite

Every non-trivial claim must trace to a KB page and section:

```
**Answer:** ...

**Sources:**
- kb/domains/licences.md — Key Rules / [section]
- kb/domains/payments.md — Current Understanding

**Confidence:** Confirmed | Inferred | Unverified | Mixed

**Gaps:** The KB does not currently cover X. Consider ingesting Y.
```

Use `Mixed` when claims draw from pages with differing confidence levels.

### Phase 4: Persist (optional)

Offer to file the answer back as a new KB page when:

- The answer synthesizes content from 3+ pages
- The answer reveals a connection or pattern not documented anywhere
- The answer would save significant re-derivation next time

If filing, follow `KB_RULES.md` frontmatter exactly. Typical type: `workflow`, `architecture`, or `domain`.

### Phase 5: Log

Always append one entry to `kb/log.md`:

```
## [YYYY-MM-DD] query | <brief question summary>
```

## Rules

- Never invent. If it's not in the KB, say so.
- Surface contradictions rather than resolving them silently.
- Recommend specific ingests when KB gaps are relevant to the question.
- Do not read raw source files in `docs/` or `cc/` unless the KB page explicitly references them
  and you need to clarify a confidence claim.
