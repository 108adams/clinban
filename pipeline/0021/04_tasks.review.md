# Tasks Review
_Reviewer: Codex_
_Date: 2026-06-14_
_Artifact reviewed: pipeline/0021/04_tasks.md_

## Challenges
- `pipeline/0021/04_tasks.md:173-182` defers all docs to T8 while `pipeline/0021/04_tasks.md:8` requires one commit per task. That conflicts with the project documentation rule that behavior changes update code, tests, Go docs, relevant `docs/` pages, `docs/index.md` when pages are added, and `docs/log.md` as part of the task. T4 adds a new user-facing command at `pipeline/0021/04_tasks.md:89-96`, so delaying docs until a later commit creates an intentionally stale wiki state.
  > **Disposition:** ACCEPTED — verified against AGENTS.md §"Documentation Rules" (74-83) and §"Definition of Done" (87-100): per-task docs are mandatory, not optional follow-up. Added a per-task docs rule to the preamble (now §9-13); moved docs into T4 (new `docs/clinban-board.md` + `docs/cli.md` + `docs/index.md` + `docs/log.md`) and into T5/T6/T7 (page + `log.md` per landed behavior). T8 retitled to "Verification + docs reconciliation".
- `pipeline/0021/04_tasks.md:89-106` omits the `r` reload and `?` help keys from the TUI scaffold even though the accepted design resolves the spike keymap to include `r` and `?` in `pipeline/0021/03_design.md:16`. T8 then asks for docs covering the keymap at `pipeline/0021/04_tasks.md:180`, but no task implements or tests the full documented keymap.
  > **Disposition:** ACCEPTED — confirmed design §16 keymap includes `r`/`?` and no task implemented them. T4 now owns the complete non-mutating keymap (`r` reload re-issues `loadTickets`; `?` toggles the `help.Model` already in the design Model struct), with two new tests (`r` issues reload, `?` toggles help). Summary table T4 row updated.
- `pipeline/0021/04_tasks.md:155` writes `lint.ValidateForCommit(raw, base[:4], base, st.AllIDs())`, but `st.AllIDs()` returns `([]string, error)`. The task text does not make the ID-scan error path explicit even though the design requires `st.AllIDs()` failures to leave the live file untouched and surface as parse/IO errors in `pipeline/0021/03_design.md:155-157`.
  > **Disposition:** ACCEPTED — verified `internal/store/scan.go:168` `func (s *Store) AllIDs() ([]string, error)`; the inline call would not compile and hid the scan-error path. T7 deliverable rewritten: `commitEdit` collects `allIDs, err := st.AllIDs()` first, surfaces the error as `parseOrIOErr` (live untouched, `ValidateForCommit` not reached), then calls `ValidateForCommit(raw, base[:4], base, allIDs)`.

## Missing aspects
- T8 docs scope is incomplete. `pipeline/0021/04_tasks.md:180-182` lists only a new board page and `docs/log.md`, but adding a page also requires `docs/index.md`, and adding `clinban board` should update the CLI docs. `cmd/clinban/schema.md` should also get an explicit relevance decision because AGENTS.md requires checking it when behavior affects generated `SCHEMA.md` guidance.
  > **Disposition:** ACCEPTED — confirmed `docs/index.md`, `docs/cli.md`, and `cmd/clinban/schema.md` all exist. T4 now updates `docs/cli.md` + `docs/index.md` and records an explicit schema relevance decision ("unchanged — board adds no frontmatter/`SCHEMA.md` guidance"). T8 reconciliation re-confirms all four.
- T4 does not specify command registration in the root help/CLI docs test surface beyond “self-registers” at `pipeline/0021/04_tasks.md:90`; a realistic command test should cover `clinban board --help` or root command discovery without launching a TTY.
  > **Disposition:** ACCEPTED — added a T4 test: `clinban board` registered on `rootCmd` and `clinban board --help` succeeds without launching the TUI (cobra short-circuits help; follows the existing `cmd/clinban` binary integration-test pattern noted in CLAUDE.md).
- T7 tests at `pipeline/0021/04_tasks.md:161-167` cover parse and lint failures, but not `st.AllIDs()` failure or scratch read failure, both of which the accepted design groups under `parseOrIOErr`.
  > **Disposition:** ACCEPTED — added two T7 tests: scratch read failure → `parseOrIOErr` (live byte-identical, no write) and `st.AllIDs()` scan failure (unreadable archive-dir fixture) → `parseOrIOErr`, `ValidateForCommit` not reached, no write.

## Alternative approaches
- Move documentation work into the implementation tasks that introduce behavior: T4 updates `docs/cli.md`, the new board page, `docs/index.md`, and `docs/log.md`; T5-T7 update the board page as preview/status/edit behavior lands. Trade-off: more doc churn per task, but each commit remains shippable and aligned with project rules.
  > **Disposition:** ACCEPTED — this is the implementation chosen for Challenge 1; docs distributed exactly as proposed (T4 the full page + cli.md + index.md + log.md; T5/T6/T7 incremental page + log.md). Accepted doc-churn trade-off in exchange for shippable, rule-compliant commits.
- Add a small T4/T5 task slice for non-mutating keymap completion: implement `r` reload and `?` help before edit/status mutation work. Trade-off: one more early UI test path, but the board’s documented controls match the accepted design before risky filesystem actions are added.
  > **Disposition:** ACCEPTED — folded into T4 rather than a separate slice (the scaffold already builds `help.Model` and `loadTickets`, so `r`/`?` cost nothing extra there and keep the documented keymap whole from the first TUI task).
- Make `commitEdit` collect IDs before calling `ValidateForCommit`, returning `editCommittedMsg{parseOrIOErr: err}` on failure. Trade-off: slightly more command plumbing, but it preserves the parse-vs-lint distinction and avoids hiding repository scan failures inside validation.
  > **Disposition:** ACCEPTED — adopted verbatim as the T7 commit sequence (collect `allIDs, err := st.AllIDs()` → `parseOrIOErr` on failure → `ValidateForCommit`). Matches design §"commitEdit failure modes".

## Risks
- Documentation drift is likely because the task plan makes several behavior commits before the docs catch up, especially for the new `clinban board` command and keymap.
  > **Disposition:** ACCEPTED — mitigated by distributing docs into T4–T7 (each behavior commit ships its own docs); T8 is now a reconciliation gate, not the catch-up point.
- If `r` and `?` are documented from the design but not implemented by the tasks, users and tests can disagree about the first TUI release’s controls.
  > **Disposition:** ACCEPTED — mitigated: `r`/`?` are now implemented and tested in T4, and T8 verifies the documented keymap matches landed behavior.
- If `AllIDs()` errors are not handled distinctly during edit commit, the TUI may either fail to compile, conflate repository scan errors with lint errors, or accidentally proceed with incomplete uniqueness context.
  > **Disposition:** ACCEPTED — mitigated: T7 now collects `st.AllIDs()` (compiles), routes its error to `parseOrIOErr` (distinct from lint), and never writes on scan failure; covered by a dedicated test.

## Summary verdict: REVISE
The task breakdown is mostly aligned with the accepted architecture and design, especially in T1-T7’s package boundaries and filesystem safety locks. It needs revision before implementation because “T8 — Verification + docs” conflicts with the project’s per-change documentation rules, “T4 — TUI scaffold” omits accepted `r`/`?` keymap behavior, and “T7 — Editor handoff” underspecifies the `st.AllIDs()` error path required by the design’s commit-gate contract.