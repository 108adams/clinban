# Design + Tasks Review (0020)
_Reviewer: Codex_
_Date: 2026-06-14_
_Artifact reviewed: pipeline/0020/03_design.md + pipeline/0020/04_tasks.md_

## Challenges
- `pipeline/0020/03_design.md:21` and `pipeline/0020/04_tasks.md:11` inject only `${{ github.ref_name }}`. This does not satisfy `tickets/0020-version-command.md:12`, which asks for `last git tag | commit hash | date of the commit` for the build source commit. A tag alone loses commit hash and commit date.

> **Disposition:** DEFERRED — Architecture ADR-1 (accepted) explicitly scoped the version string to `git describe` output (`v<tag>[-N-g<hash>]`) and traded commit date for simplicity. The arch goal is tag comparison against GitHub Releases, not full provenance. Date gap is an architecture-level decision; if needed, raise a new ticket against the architecture.

- `pipeline/0020/03_design.md:62` says users run `./clinban --version`, but the ticket asks for `version / -v command` at `tickets/0020-version-command.md:12`. The design never specifies whether `clinban version`, `clinban -v`, or both are required, and `pipeline/0020/04_tasks.md` contains no task or acceptance criterion for either.

> **Disposition:** ACCEPTED — `root.go:33` confirms `Version: version` exposes `--version` only (no `-v`). Design updated to explicitly state that `--version` is the implementation of the ticket's “version / -v command” requirement, and that `-v` shorthand is out of scope. TASK-001 notes updated accordingly.

- `pipeline/0020/03_design.md:32` and `pipeline/0020/04_tasks.md:25` accept a per-runner `checksums.txt` uploaded by each matrix job. All jobs use the same asset name, so the release can end up with overwrite races, upload conflicts, or one checksum file that only covers the last uploaded binary.

> **Disposition:** ACCEPTED — Design and tasks updated: `checksums.txt` replaced with per-binary `.sha256` files (`clinban-linux-amd64.sha256`, etc.), one per matrix job, no shared filename. Eliminates the parallel-upload race entirely.

- `pipeline/0020/03_design.md:19` claims checkout gets the “full repo with tags,” but `pipeline/0020/04_tasks.md:11` and the done criteria do not require `fetch-depth: 0`. If the implementation later follows the architecture requirement to derive `git describe` metadata, the default shallow checkout is not enough.

> **Disposition:** REJECTED — The design uses `github.ref_name` (the tag name from the GitHub event), not `git describe`. `github.ref_name` is injected by Actions directly; no git history is needed. Default shallow clone is correct. Checkout step description updated to clarify this.

## Missing aspects
- No task verifies the existing CLI behavior for `--version`, `-v`, or a `version` subcommand, despite the in-scope feature being a version command.

> **Disposition:** REJECTED — TASK-002 done criteria (smoke test) already requires: "clinban-linux-amd64 --version returns the pushed tag string." Sufficient coverage for a single YAML-only change with no new Go code.

- No task updates `docs/cli.md` or `docs/log.md`, even though exposing version behavior is user-facing CLI behavior under the project documentation rules.

> **Disposition:** ACCEPTED — TASK-003 added: update `docs/cli.md` with `--version` section and append to `docs/log.md`, bundled in same commit as TASK-001.

- No task updates the Makefile or release workflow to include commit date, so the requested metadata tuple cannot be produced consistently across local and release builds.

> **Disposition:** DEFERRED — same as C1; architecture ADR-1 decision. Commit date is not in scope.

- No acceptance criterion checks the exact version string format, for example `v0.2.0 | <hash> | <commit-date>`, so implementers can ship mutually incompatible formats while satisfying the current task list.

> **Disposition:** ACCEPTED (partially) — Added done criterion to TASK-001: "`clinban --version` on a release binary prints exactly `v<tag>`." Date/hash format deferred per ADR-1.

- No task runs `go test ./...` and `go vet ./...` after the workflow/design changes, even though AGENTS.md requires them before handing work back for implementation tasks.

> **Disposition:** REJECTED — TASK-001 produces only a YAML file (`.github/workflows/release.yml`), no Go code changes. Running `go test ./...` before tagging is a general project gate, not a ticket-specific task. TASK-002 (smoke test) gates the final verification.

## Alternative approaches
- Add a small `version` subcommand plus `-v` root shorthand that prints a structured build string from package variables such as `version`, `commit`, and `commitDate`. Trade-off: a little more Go code and tests, but it directly matches the ticket text and avoids overloading only Cobra’s `--version`.

> **Disposition:** DEFERRED — architecture chose `--version` (ADR-1). Subcommand + `-v` shorthand is a valid future enhancement if the CLI surface needs it; create a new ticket.

- Keep Cobra `--version`, but explicitly define ticket scope as `--version` plus `-v` alias and reject a separate `version` subcommand in the design. Trade-off: simpler CLI surface, but this needs a documented interpretation because the ticket says “command.”

> **Disposition:** ACCEPTED — design now explicitly documents that `--version` is the implementation and `-v` is out of scope for this ticket.

- Generate one checksum manifest in a follow-up aggregation job after all matrix builds upload artifacts, then upload release assets once. Trade-off: more workflow YAML, but no duplicate `checksums.txt` asset race and one manifest verifies all binaries.

> **Disposition:** ACCEPTED (simpler variant) — chose per-binary `.sha256` files over an aggregation job; same safety, less YAML complexity.

- Use GoReleaser for this release workflow. Trade-off: adds tooling, but it handles multi-platform builds, checksums, release assets, and version metadata conventions more reliably than a hand-built matrix.

> **Disposition:** REJECTED — ADR-2 explicitly decided against GoReleaser for this ticket. Captured in memo ticket 0026 for future consideration.

## Risks
- Users may install a release binary that reports only `v0.2.0`, making it impossible to identify the exact source commit or commit date if a tag is moved, rebuilt, or disputed.

> **Disposition:** DEFERRED — inherent to the ADR-1 scope decision (no commit date). Accepted tradeoff.

- Implementers may complete only `.github/workflows/release.yml` and leave the requested `version` or `-v` CLI behavior absent.

> **Disposition:** ACCEPTED — design now explicitly documents the `--version` scope; TASK-001 done criteria include version string format check.

- Parallel matrix uploads may produce incomplete or conflicting checksum assets, weakening release verification exactly where the design claims integrity coverage.

> **Disposition:** ACCEPTED — fixed by switching to per-binary `.sha256` files; no shared filename, no race.

- Documentation can drift immediately: `docs/cli.md` currently has no version command section, and the task list does not require adding one.

> **Disposition:** ACCEPTED — TASK-003 added.

## Summary verdict: REVISE
The design is useful for a release workflow, but it narrows the ticket from “add version / -v command with tag, hash, and commit date” into “publish binaries that answer `--version` with the tag.” The main gaps are in `Release Workflow`, `Test Strategy`, and `TASK-001`: they omit the requested CLI surface, omit commit hash/date metadata, and define a fragile checksum upload model.