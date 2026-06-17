# Codex ↔ Claude Cross-Check: Why the Reviewer Finds Valid Flaws

_Created: 2026-06-14_
_Status: analysis for later review (continue in a fresh chat)_
_Context: written after the ticket 0021 (Terminal UI foundation) pipeline runs — architecture and design stages each went Claude-authored → Codex-reviewed → Claude-reconciled. Codex returned REVISE on both; most findings were valid. This doc explains why, and proposes a proportionate fix._

---

## 0. The question being answered

When Claude authors a pipeline artifact (architecture, design) and Codex cross-reviews it, Codex
reliably finds a number of **valid** flaws. Why? Is Claude's design weaker than Codex's review?

Observed counter-evidence: when **Codex authors** and **Claude reviews**, Claude finds just as many
valid flaws. So the effect is **symmetric** — it is about *complementary positions*, not one model
being better than the other. This doc analyses the structural cause and asks whether the authoring
skills can be tuned to ship fewer mechanical flaws *without* undermining the value of the second pass.

---

## 1. The core asymmetry: generator seat vs. critic seat

The author and the reviewer are doing cognitively different jobs with different economics.

**Author (generator seat).** Holds the entire artifact in working memory and optimises for a
*coherent, forward-moving whole*: structure, happy path, plausibility, narrative completeness. Biased
toward finishing the artifact. Maintaining N sections in mutual consistency is roughly O(N²) pairwise
checks — done approximately, under load, *while simultaneously inventing the content*.

**Reviewer (critic seat).** Treats the finished text as a *closed system*. Does not need to maintain
global coherence — needs only to find **one** place where two statements disagree or a path is
unhandled. Finding a single inconsistency in a finished artifact is a targeted, high-yield search.

> Breaking local coherence is cheaper than maintaining global coherence.

This is why the effect is **symmetric**: swap seats and the advantage swaps. Whoever holds the review
seat inherits the structurally easier job. **The review value is positional, not an
intelligence differential.**

### 1a. The author's second handicap: too much context

The author *knows what they meant*, so they unconsciously fill their own gaps and do not re-read their
own invariants as a stranger would. The reviewer has **zero intent** and reads every sentence
literally — which is exactly the right amount of context to catch "you stated X globally, then did
not-X locally." The author has too much context; the reviewer has none, and none is better for this
particular task.

---

## 2. What Codex actually caught (the recurring pattern)

Grouping the valid findings from the 0021 architecture + design reviews, the root causes are narrow
and repeatable:

| Class | Examples (0021) | Root cause |
|-------|-----------------|------------|
| **Invariant violated by a local decision** | scratch I/O placed in `Update` vs. the stated "no blocking I/O in `Update`"; "no new filesystem mutation" vs. accepting live-file edit (arch round) | invariant written in one pass, the contradicting decision in another, never cross-checked |
| **Failure paths under-specified** | `editor.Command` error path ignored; `AllIDs()` failure mode absent from the command table; parse-error vs. lint-error not separated | happy path designed richly; failure branches added as afterthoughts, unevenly |
| **Resource lifecycle incomplete** | scratch temp file: no cleanup-on-every-exit, no crash case | resource created without tracing its full lifetime |
| **Upstream conformance gap** | design did not "cash" the architecture's fresh-read/concurrency contract | each parent contract not walked down into a concrete mechanism |
| **Requirement dropped** | normal-terminal verification and v2 import-path confirmation missing (arch round) | coverage check verified "did I produce sections," not "did I hit every spec line" |
| **Layering / boundary taste** | `internal/board` depending on `internal/store` | genuine judgement, least mechanical |

**Key split:**
- The first four classes are **mechanical** — a checklist catches them.
- The last is **judgement** — where an independent second mind genuinely adds value.

**Second key nuance:** a sizeable share of Codex's "valid" findings were **imprecision, not error** —
a careful developer might have implemented the artifact correctly, but the *artifact did not say it*.
So part of the review's job is **forcing precision**, which even a correct design benefits from.

---

## 3. Why this is complementary, not better/worse

- The advantage is **positional** and **swaps with the seat**. Confirmed empirically: Claude reviewing
  Codex finds a comparable number of valid flaws.
- The two models also have **diverse blind spots**. Two different passes catch a union of defects
  larger than either pass alone, even at equal skill.
- Therefore the pipeline's "author → independent review → reconcile" loop is not a workaround for a
  weak author; it is exploiting a real, structural property of generate-vs-critique.

---

## 4. Can the author ship fewer mechanical flaws? Yes — with a ceiling, on purpose

The author *can* import some of the reviewer's discipline via a **pre-handoff self-audit** — put on the
critic hat once before writing the artifact — targeting exactly the four mechanical classes:

1. **Invariant ledger** — list every global "never / always / MUST" claim, then scan every decision and
   confirm none violates it. (catches: invariant violations)
2. **Failure-path sweep** — for every signature returning `error` and every fallible external call,
   state the failure branch *and* its state/cleanup consequence. (catches: under-specified failures)
3. **Upstream conformance** — list every lock/contract from the parent artifact
   (ticket → architecture; architecture → design) and point at the mechanism that satisfies each.
   (catches: conformance gaps + dropped requirements)
4. **Resource lifecycle** — for every created resource (temp file, goroutine, handle), trace
   creation → every exit path → cleanup → crash. (catches: lifecycle gaps)

### 4a. Why the target is NOT zero flaws

Two hard limits make full self-review a mistake:

- **Author blind spot is irreducible.** The author cannot fully de-context themselves; some gaps are
  only visible to a stranger.
- **Independence is the asset.** Two diverse passes beat one deep pass. Exhaustive self-audit costs
  significantly more time/tokens *and* erodes the second reviewer's marginal value. The pipeline is
  built around two seats; collapsing them into one is a regression disguised as efficiency.

**Right target:** catch the cheap **mechanical** classes in self-audit; leave **conformance nuance**
and **taste** to the cross-review, where an independent mind genuinely pays off. This shrinks the
human-visible flaw count without pretending the review is redundant.

---

## 5. Proposed change (for discussion)

Add the 4-item self-audit as a **lean final gate** before the handoff step in the authoring skills.
Keep it to ~4 bullets, run once — explicitly *not* a heavyweight process (respect the no-gold-plating
preference).

**Open decisions for the later session:**
- Which skills get it: `architect` + `techlead` for sure; also a trimmed variant in `dev` (same defect
  classes appear there as code-level issues)?
- Exact placement: a "Pre-handoff self-audit" block immediately before each skill's Handoff section.
- Whether to also record the generator/critic-asymmetry insight as a **feedback memory** so it persists
  regardless of the skill edits.
- Metric to watch: count of *valid* (accepted, non-deferred) Codex findings per stage before vs. after
  the change. Success = the mechanical classes (1–4) trend toward zero while taste/conformance findings
  remain (and remain welcome).

### Draft block to drop into the skills

```markdown
## Pre-Handoff Self-Audit (run once before offering to write the artifact)

Put on the reviewer's hat. Do not skip — these are the mechanical defect classes an
independent reviewer reliably finds:

1. Invariant ledger — list every global "never/always/MUST" claim in the artifact;
   scan every decision and confirm none violates it.
2. Failure-path sweep — for every error-returning signature and every fallible call,
   state the failure branch and its state/cleanup consequence.
3. Upstream conformance — list every lock/contract from the parent artifact; point at
   the exact mechanism that satisfies each.
4. Resource lifecycle — for every created resource, trace creation → every exit →
   cleanup → crash.

Fix what this surfaces. Leave conformance nuance and design taste for the cross-review —
do not try to fully internalise the second reviewer; independence is the point.
```

---

## 6. One-line summary

Codex finds valid flaws because **reviewing a finished artifact is a structurally easier, higher-yield
position than authoring one** — and the effect is symmetric, so it is complementary by design. A short
self-audit can move the *mechanical* defects upstream; the *judgement* defects should stay with the
independent reviewer, because that diversity is exactly what the cross-review buys.
