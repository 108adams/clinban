# Design Review
_Reviewer: Codex_
_Date: 2026-06-14_
_Artifact reviewed: pipeline/0021/03_design.md_

## Challenges
- `pipeline/0021/03_design.md:130-141` does not satisfy the accepted architecture's concurrency contract in `pipeline/0021/02_architecture.md:61-63`: edit commit validates the scratch file and writes it to `livePath`, but it does not fresh-read or otherwise verify the live ticket immediately before writing, so an external edit made after scratch creation can be overwritten.
> **Disposition:** ACCEPTED (scope clarified) — Verified `edit.go:113-117`: the CLI `edit` also writes the scratch over the live file without a pre-write re-read. The architecture's "never writes from the snapshot" lock still holds (edit writes the user's scratch content, not the board snapshot). Added an "Edit concurrency (scope note)" subsection: the fresh-read-before-write lock applies to **status advance**; edit mirrors the CLI and the narrow external-edit window is an accepted single-user limitation. The proposed conflict-check is DEFERRED (would make the TUI stricter than the CLI — see Alternative #2). `beginEdit` now fresh-resolves livePath via `FindByID` (added to Locks).
- `pipeline/0021/03_design.md:132-135` puts scratch-copy creation directly in `Update`, while `pipeline/0021/03_design.md:97-100` says `Update` performs no blocking I/O. The design needs to move scratch creation into a `tea.Cmd` or explicitly narrow that contract.
> **Disposition:** ACCEPTED — Real contradiction. Edit handoff rewritten around a `beginEdit(st, id)` `tea.Cmd` that performs all filesystem setup off the Update path; `Update` on `e` just returns `beginEdit`. New `editReadyMsg`/`editBeginFailedMsg`. The "no blocking I/O in `Update`" contract is now honoured.
- `pipeline/0021/03_design.md:132-135` calls `editor.Command(scratch)` inline but does not specify the error path from `editor.Command`, even though the proposed signature returns `(*exec.Cmd, error)` at `pipeline/0021/03_design.md:46`. This leaves scratch cleanup and user-facing error behavior undefined before `tea.ExecProcess` starts.
> **Disposition:** ACCEPTED — Folded into `beginEdit`: an `editor.Command` error yields `editBeginFailedMsg{err}` with the partial scratch removed before `tea.ExecProcess` is ever reached. Covered by the new "Scratch cleanup policy" subsection and a `beginEdit` failure test.
- `pipeline/0021/03_design.md:130` says `commitEdit` calls `st.AllIDs()` but the command table does not include the resulting failure mode. If ID scanning fails, the live file should remain untouched, the scratch should be cleaned up, and the model should surface an error distinct from lint errors.
> **Disposition:** ACCEPTED — `editCommittedMsg` now carries `{lintErrs, parseOrIOErr}`. New "commitEdit failure modes" subsection: scratch-read, `ticket.Parse`, and `st.AllIDs()` errors all leave the live file untouched, remove the scratch, and report via `parseOrIOErr` distinct from `lintErrs`. Three-outcome test added.

## Missing aspects
- No explicit live-file change detection for the edit workflow, despite the architecture claiming stale snapshots cannot clobber concurrent external edits (`pipeline/0021/02_architecture.md:61-63`).
> **Disposition:** ACCEPTED (scope clarified) — Same root as Challenge 1. The "stale snapshot cannot clobber" lock is precise for status advance; edit commits user-edited scratch content, mirroring the CLI. Documented in the "Edit concurrency" note; change detection deferred (Alternative #2).
- No specified message for scratch-copy completion/failure. The design has messages for editor completion and commit completion, but no `scratchCreatedMsg` or equivalent to keep filesystem setup out of `Update`.
> **Disposition:** ACCEPTED — Added `editReadyMsg{scratch, livePath, cmd}` (success) and `editBeginFailedMsg{err}` (failure), both produced by `beginEdit`. This is the message that keeps filesystem setup out of `Update`.
- No cleanup policy for abandoned scratch files if scratch creation succeeds but editor command construction fails, the program exits unexpectedly after setting `scratch`, or `commitEdit` cannot read/remove the scratch.
> **Disposition:** ACCEPTED — New "Scratch cleanup policy" subsection: scratch removed in every terminal handler (`editBeginFailed`, `editFinished` error, `editCommitted`). A crash-leftover scratch is harmless — its `.`-prefixed name never matches the managed `^[0-9]{4}-…\.md` convention (scan.go:13), so `ListActive` ignores it — matching the CLI `edit` best-effort cleanup (edit.go:66).
- No design detail for mapping list selection to a stable ticket identity across reloads. After `loadTickets`, selection may jump if sorting changes or the selected ticket moves status; preserving by ID would make status advance/edit reloads less surprising.
> **Disposition:** ACCEPTED (good catch) — New "Reload & selection stability" subsection: reload re-selects by ticket ID (clamp if gone), so a status advance that re-sorts the ticket keeps the cursor on it. Added to T6 deliverable + a unit test.
- No package-boundary note for `internal/board` depending on `internal/store` (`pipeline/0021/03_design.md:36`). That may be acceptable, but because this package becomes shared ordering infrastructure, the design should justify why a display-order package owns a `store.Record` sorter instead of exposing a comparator/key over ticket fields.
> **Disposition:** ACCEPTED — `board` now depends on `internal/ticket` only and exposes `Less(a, b *ticket.Ticket) bool` (+ `StatusOrder`). Callers (`list.go`, `tui`) sort their own `[]store.Record` via `sort.SliceStable` + `board.Less`. T2 updated.

## Alternative approaches
- Use a `beginEdit(st, selectedID)` command that fresh-resolves the selected ticket, reads bytes, creates the same-directory scratch copy, builds `editor.Command`, and returns either `editReadyMsg{scratch, livePath, cmd}` or `editBeginFailedMsg{err}`. Trade-off: one more message, but `Update` remains non-blocking and error cleanup is centralized.
> **Disposition:** ADOPTED — taken verbatim; this is the new edit handoff sequence.
- Track the original live file fingerprint when creating the scratch copy, then compare it before commit. Trade-off: adds a conflict path, but it is the clearest way to satisfy the accepted “stale snapshot cannot clobber” contract.
> **Disposition:** DEFERRED — Would make the TUI stricter than the CLI `edit` (which has no such check, edit.go:113-117) and is beyond spike scope for a single-user tool. Recorded as a future enhancement in the "Edit concurrency" note.
- Extract a shared edit transaction helper outside `cmd/clinban/edit.go`, for example in an internal package that owns scratch lifecycle and parse/lint commit validation. Trade-off: more abstraction now, but it prevents the CLI edit loop and TUI edit gate from drifting.
> **Disposition:** PARTIAL/DEFERRED — The parse+lint **validation** is already shared (`lint.ValidateForCommit`, T3). The **scratch lifecycle** differs structurally (CLI = synchronous `defer`; TUI = async message handlers), so a shared lifecycle helper would be an awkward fit; deferred rather than forced now.
- Make `internal/board` sort `[]*ticket.Ticket` or expose `Less(a, b *ticket.Ticket) bool`, leaving `store.Record` sorting adapters in callers. Trade-off: slightly more caller code, but the board package avoids depending on filesystem record types.
> **Disposition:** ADOPTED — `board.Less(a, b *ticket.Ticket) bool`; callers keep the `[]store.Record` adapter.

## Risks
- Concurrent external edits can be silently lost during TUI edit commit unless the design adds a live-file precondition check.
> **Disposition:** ACCEPTED as a known, bounded limitation — identical to the CLI `edit` (edit.go:113-117); documented in "Edit concurrency". Conflict detection deferred (Alternative #2).
- Filesystem work inside `Update` can freeze the UI on slow filesystems or complicate test expectations around Bubble Tea’s message loop.
> **Disposition:** ACCEPTED — `beginEdit` moves all edit-path filesystem work into a `tea.Cmd`; `Update` does no I/O. (Other store access was already in `tea.Cmd`s.)
- Unhandled `editor.Command` failures can leave scratch files behind and produce inconsistent model state before the editor ever opens.
> **Disposition:** ACCEPTED — `beginEdit` removes the partial scratch and returns `editBeginFailedMsg` on an `editor.Command` error; covered by the cleanup policy + a test.
- If reloads do not preserve selection by ticket ID, status changes can make the operator act on a different ticket than intended after a refresh.
> **Disposition:** ACCEPTED — Reload re-selects by ticket ID ("Reload & selection stability"); tested in T6.
- If parse errors and lint errors are not represented separately in edit commit messages, the UI may show misleading validation output and tests may miss the parse-vs-lint distinction required by project rules.
> **Disposition:** ACCEPTED — `editCommittedMsg{lintErrs, parseOrIOErr}` separates the two; `lint.ValidateForCommit` already returns parse error distinct from lint errors. Distinct parse-error and lint-violation tests added to T7.

## Summary verdict: REVISE
The design is directionally aligned with the accepted Charm/Bubble Tea architecture and preserves the main store/fsm/editor boundaries, especially in “Module structure,” “Interface contracts,” and “Locks honoured.” It still needs revision before implementation because the edit workflow in “Edit handoff sequence” does not meet the architecture’s fresh-read/concurrency claim, performs scratch filesystem setup in `Update` despite the model contract, and omits concrete error/cleanup paths for editor command construction and ID scanning.