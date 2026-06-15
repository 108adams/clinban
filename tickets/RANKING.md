# Ticket Implementation Ranking

Prepared: 2026-06-11
Scope: 0014, 0020, 0021, 0024, 0025

Ranking criteria (top → bottom):
1. Correctness bugs that corrupt or leave the ticket store in an unrecoverable state
2. Blockers on a planned phase (spike/ADR tickets that gate all downstream work)
3. Small, zero-dependency CLI polish features
4. Store-layer refactors that precondition safe phase-two work
5. Internal cleanup with no user-visible impact
6. Design stubs needing BA pass before they can be planned

---

## Tier 2 — Next batch

| # | Ticket | Why |
|---|--------|-----|
| 1 | **0021** Choose terminal UI foundation | ADR written, decision made (Charm stack); open questions must close before any TUI implementation ticket can be planned — blocks all phase-two work |
| ~~2~~ | ~~**0020** 'version' command~~ | ~~Zero-dependency CLI polish; can ride alongside 0021~~ |

## Tier 3 — Then

| # | Ticket | Why |
|---|--------|-----|
| ~~1~~ | ~~**0025** refactor: extract shared directory-scan helper in store/scan.go~~ | ~~Three near-identical loops in scan.go; consolidate before TUI work adds a fourth store consumer~~ |
| ~~2~~ | ~~**0024** refactor: planResolve — use store.NextID, pre-sort IDs, single-pass group build~~ | ~~No user-visible impact; do after 0025 since both touch store internals~~ |

## Tier 4 — Later

| # | Ticket | Why |
|---|--------|-----|
| 1 | **0014** MCP design | Stub with no acceptance criteria; needs /ba pass before it can enter the pipeline |

## Dependencies

- 0021 blocks all TUI implementation tickets (not yet created)
- 0025 is soft precondition for 0024 (deeper store refactor first)
- 0014 needs a /ba pass before architecture can plan it
