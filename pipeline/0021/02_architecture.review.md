# Architecture Review
_Reviewer: Codex_
_Date: 2026-06-14_
_Artifact reviewed: pipeline/0021/02_architecture.md_

## Challenges
- `pipeline/0021/02_architecture.md:41-42` says the TUI introduces no new filesystem mutation, but `pipeline/0021/02_architecture.md:107-121` accepts editing the live ticket file directly. That is filesystem mutation outside `internal/store`, bypasses the same-directory temp-file plus rename pattern, and weakens the existing CLI edit safety model in `cmd/clinban/edit.go:55-119`.
> **Disposition:** ACCEPTED — Verified against edit.go:55-119 (scratch copy + parse+lint gate, commit only on success). ADR-3 rewritten to edit a same-directory scratch copy and commit exclusively via `Store.WriteTicket` after parse+lint pass; boundary statement reworded to "no new ticket-mutation primitive — all managed-ticket writes go through `Store.WriteTicket`."
- `pipeline/0021/02_architecture.md:119-120` treats persisted invalid live edits as acceptable for the spike, but the ticket requires opening `$EDITOR` using existing editor behavior and refreshing afterward (`tickets/0021-choose-terminal-ui-foundation.md:164`). Existing user-facing edit behavior gates writes through parse+lint and preserves the original on invalid edits (`cmd/clinban/edit.go:23-26`, `cmd/clinban/edit.go:35-40`).
> **Disposition:** ACCEPTED — Folded into the ADR-3 rewrite. The "invalid save persists" consequence is removed; on parse/lint failure the original is left untouched and errors are surfaced in-UI (replacing the stdin `[y/N]` reopen loop, which cannot run under the alt-screen).
- `pipeline/0021/02_architecture.md:48` assigns panic recovery/terminal restoration to Charm, but the architecture does not require verification that Bubble Tea recovers from application panics in `Update`/`View` or describe a command-level recovery fallback. Terminal corruption is explicitly named as a failure mode, so this should not rest on an unchecked library assumption.
> **Disposition:** ACCEPTED — Spike Acceptance now requires a manual normal-terminal check that a forced panic in `Update` (plus resize and editor handoff) leaves the terminal restored (cooked mode, main screen) on exit — "verified, not assumed."
- `pipeline/0021/02_architecture.md:155` leaves the home for shared board ordering undecided even though `pipeline/0021/02_architecture.md:38` makes sharing the order a proposed change and `pipeline/0021/02_architecture.md:102` calls it required. This is small, but it is still an unresolved package-boundary decision in an architecture artifact.
> **Disposition:** ACCEPTED — Inconsistency reconciled. The board-order *contract* (in-progress → blocked → backlog → done) is locked as must-not-diverge; the *extraction home* is a Tech Lead call with a recommendation (`internal/board`), and the spike may re-derive the order inline if extraction is deferred. ADR-2 consequence and the open-questions row reworded accordingly.

## Missing aspects
- The spike acceptance omits normal-terminal verification from the ticket requirement. The ticket requires verification both in a normal terminal and CI-testable model/unit tests (`tickets/0021-choose-terminal-ui-foundation.md:168`), while the artifact only locks model/unit coverage at `pipeline/0021/02_architecture.md:148-149`.
> **Disposition:** ACCEPTED — Spike Acceptance now has a Verification block listing both model/unit tests *and* a normal-terminal manual check, citing tickets/0021-…:168.
- The architecture does not specify how status changes write safely. `pipeline/0021/02_architecture.md:147-148` says to advance through `fsm.NextStatus` plus existing store write, but does not spell out timestamp refresh, selected-record path handling, lint/reload behavior after write, or how errors from `WriteTicket` surface without corrupting in-memory state.
> **Disposition:** ACCEPTED — New "TUI Write & Edit Contracts" section added: status advance does a fresh `FindByID`+`ReadTicket`, `fsm.NextStatus`, refresh `Updated`, `Store.WriteTicket` (mirroring `runMove`, move.go:60-102); `WriteTicket` errors surface in-UI and the snapshot is not mutated until the write succeeds; reload via `ListActive` on success.
- The preview contract says “raw Markdown source verbatim” (`pipeline/0021/02_architecture.md:126`) but the artifact does not specify whether the source is read from `store.Record.Ticket` re-marshaled content or the original file bytes. Verbatim preview requires original bytes; parsed ticket state can reorder or normalize frontmatter.
> **Disposition:** ACCEPTED — Verified `ticket.Marshal` reorders frontmatter (ticket.go:128). ADR-4 Decision and Locks now require the preview to read original file bytes (`os.ReadFile` on `Record.Path`), never a re-marshaled `Ticket`.
- The dependency decision still defers canonical v2 import-path confirmation (`pipeline/0021/02_architecture.md:39`, `pipeline/0021/02_architecture.md:84`). That is acceptable for implementation, but the architecture should at least make the dependency verification an explicit spike acceptance item.
> **Disposition:** ACCEPTED — Spike Acceptance Verification block now lists "confirm the canonical Bubble Tea / Bubbles / Lip Gloss v2 import paths before writing code" as an explicit item.

## Alternative approaches
- Preserve CLI edit safety in the TUI: copy the selected ticket to a same-directory scratch file, run `tea.ExecProcess` on the scratch path, parse+lint after exit, then commit with `Store.WriteTicket` only on success. Trade-off: more TUI state and no stdin reopen prompt, but it preserves data integrity and existing edit semantics.
> **Disposition:** ADOPTED — this is now the ADR-3 decision verbatim.
- Add an `internal/editor.Command` export, but keep live-file decisions outside the editor package. Trade-off: clean single-source command construction without forcing the TUI to bypass store write guarantees.
> **Disposition:** ADOPTED — ADR-3 Locks: "live-file decisions stay out of the `editor` package."
- Move ordering/filter helpers into a small domain-adjacent package such as `internal/board` or `internal/query`. Trade-off: one more package, but avoids importing CLI code and prevents duplicate sort/filter behavior.
> **Disposition:** ADOPTED (as recommendation) — `internal/board` is now the recommended extraction home; final call left to Tech Lead.
- If direct live edit remains intentionally accepted, make it a separate explicit ADR with a rollback plan and a first-follow ticket to restore scratch-copy lint gating. Trade-off: faster spike, but the data-integrity exception is visible and bounded.
> **Disposition:** REJECTED (moot) — live edit is dropped entirely in favor of the scratch-copy gate, so there is no data-integrity exception to bound and no separate ADR is needed.

## Risks
- Invalid or partially saved ticket files can persist after a TUI edit, causing `Store.ListActive` to fail the entire board load on the next refresh (`internal/store/scan.go:221-249`).
> **Disposition:** ACCEPTED — eliminated by the ADR-3 scratch-copy + parse+lint gate; invalid content never reaches a managed ticket file.
- A status advance after a stale snapshot can overwrite changes made by an external editor or another Clinban command because the artifact does not define a reload-before-write or conflict check.
> **Disposition:** ACCEPTED — TUI Write & Edit Contracts now mandate a fresh `FindByID`+`ReadTicket` immediately before any status write (mirroring `runMove`), so the TUI never writes from a stale snapshot.
- If raw preview is produced from parsed/marshaled tickets instead of file bytes, users may see normalized content rather than the literal file, violating the product-owner intent captured in ADR-4.
> **Disposition:** ACCEPTED — ADR-4 now locks preview to original file bytes from `Record.Path`, never re-marshaled `Ticket`.
- If terminal panic restoration is not tested in a real terminal, failures in resize/render/editor handoff can leave raw mode or alt-screen state behind.
> **Disposition:** ACCEPTED — Spike Acceptance adds a forced-panic normal-terminal check confirming the terminal is restored on exit.

## Summary verdict: REVISE
The Charm-stack decision and `internal/tui` boundary are sound, especially in “ADR-1: Charm stack as the TUI foundation” and “ADR-2: TUI is a pure consumer in `internal/tui`; `clinban board` is a thin entry.” The artifact needs revision before approval because “ADR-3: Editor handoff via extracted `editor.Command` + `tea.ExecProcess`” contradicts the stated no-new-filesystem-mutation contract and weakens the existing parse/lint-gated edit behavior, while “Spike Acceptance” misses normal-terminal verification and leaves several write/preview contracts underspecified.