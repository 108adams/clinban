# Architecture Review
_Reviewer: Codex_
_Date: 2026-06-12_
_Artifact reviewed: pipeline/02_architecture.md_

## Challenges
- `pipeline/02_architecture.md:21-22` proposes `BatchRenameWithinDir(ops []RenameOp)` and then says `runResolve` can call `st.BatchRenameWithinDir(plan)`, but the current plan type is `cmd/clinban.resolveRename` with unexported fields in `cmd/clinban/resolve.go:39-42`. That call cannot compile unless the architecture explicitly moves the plan type to `internal/store`, changes `planResolve` to return store-owned ops, or adds a conversion layer.
> **Disposition:** ACCEPTED — verified resolve.go:39-42 fields are unexported. Architecture now adds exported `store.RenameOp{OldPath, NewBase string}` and changes `planResolve` to return `[]store.RenameOp` (Proposed Changes table).
- `pipeline/02_architecture.md:40-43` claims “Store unchanged on any error exit” and says rollback removes destination links, but that is not sufficient after Phase 2 has removed any source files. Once `os.Remove(oldPath)` succeeds for an earlier item, deleting its destination link during rollback deletes the ticket entirely. The design needs a reverse-link step for already-removed sources, or it cannot satisfy `tickets/0023-resolve-partial-rename-leaves-inconsistent.md:16-20` and `tickets/0023-resolve-partial-rename-leaves-inconsistent.md:29-31`.
> **Disposition:** ACCEPTED (critical) — confirmed against hard-link semantics in move.go. This is a real data-loss bug in the *ticket's own* execution model, not just the architecture. New "Execution Model & Rollback Algorithm" section specifies a data-loss-safe Phase-2 rollback (re-link removed sources, then remove dests). Flagged as a ticket correction needing a `/ba` pass (see "Requirement Corrections").
- `pipeline/02_architecture.md:43` assigns CLI-shaped errors (`resolve: link ...`) to the store-layer operation, while `pipeline/02_architecture.md:55-57` says rollback logic belongs in `internal/store`. Existing store errors are package-scoped (`internal/store/move.go:67-73` uses `store: rename: ...`). The artifact needs to define whether store returns typed/contextual errors for the CLI to format, or whether `internal/store` is allowed to know the `resolve` command name.
> **Disposition:** ACCEPTED — confirmed move.go:62-73 uses `store:` prefix. Architecture now defines typed `store.BatchError`; store stays package-scoped, CLI owns the `resolve:` formatting (Maintainability NFR row + ADR-1 lock).
- `pipeline/02_architecture.md:30` says `EXDEV` is impossible because of the same-directory guarantee, but `pipeline/02_architecture.md:86` leaves same-directory validation as an open question. If this is a correctness assumption, it cannot remain non-blocking without an explicit caller contract and tests.
> **Disposition:** ACCEPTED — same-directory is now an enforced contract (basename-only validation, mirroring move.go:61-63), not an assumption. Open question resolved; EXDEV-impossible is justified by construction.

## Missing aspects
- The artifact does not describe the rollback algorithm for Phase 2 in enough detail to preserve files whose source paths were already removed. It should track per-op state: linked destinations, sources removed, source restoration attempts, and destination cleanup attempts.
> **Disposition:** ACCEPTED — new "Execution Model & Rollback Algorithm" section tracks `linked` and `removed` per-op state and specifies the re-link-then-remove rollback.
- The artifact does not specify the API shape of `RenameOp`, including exported field names, whether paths are absolute, whether `NewBase` must be basename-only, and whether `BatchRenameWithinDir` returns the new paths for success output.
> **Disposition:** ACCEPTED — `store.RenameOp{OldPath, NewBase string}` defined; `NewBase` basename-only (validated); `BatchRenameWithinDir` returns `[]string` new paths (Proposed Changes + Integration Contracts).
- The artifact does not cover how `runResolve` prints `renamed: old -> new` after a successful batch. The current command prints each rename using the returned path from `RenameWithinDir` in `cmd/clinban/resolve.go:60-66`; a batch API either needs to return results or the CLI must compute display paths consistently.
> **Disposition:** ACCEPTED — confirmed resolve.go:65 prints from the returned path. `BatchRenameWithinDir` now returns new paths in plan order for the CLI to print.
- The artifact does not mention tests for rollback failure reporting, even though the ticket requires reporting original plus rollback errors at `tickets/0023-resolve-partial-rename-leaves-inconsistent.md:31-37`.
> **Disposition:** ACCEPTED — Testability note added (rollback-failure coverage + N>1 Remove-failure restore case). Detailed test enumeration belongs to the tasks stage.
- The requested relevant rule `.claude/rules/security.md` is absent in this checkout; the available security source is `docs/security.md`, which states malicious local files can matter because Clinban renames/removes files (`docs/security.md:17-24`). The architecture should account for path/basename validation in the new exported batch API.
> **Disposition:** ACCEPTED — confirmed `.claude/rules/security.md` absent, `docs/security.md` present (request-review passed the wrong path). Security NFR row + same-directory contract now cite `docs/security.md:17-24` and require basename-only validation.

## Alternative approaches
- Keep `BatchRenameWithinDir` in `internal/store`, but make it stateful and reversible: Phase 1 links all destinations; Phase 2 removes sources one at a time; on Phase 2 failure, recreate any removed source path from its destination link before removing that destination. Trade-off: more code and more failure modes, but it is the only hard-link approach that can meet zero net change after partial Phase 2 success.
> **Disposition:** ACCEPTED — adopted verbatim as the Execution Model. This is the core fix.
- Expose a store-owned `RenameOp` and have `planResolve` return `[]store.RenameOp`. Trade-off: the CLI planning function becomes coupled to a store DTO, but it avoids an otherwise pointless conversion and keeps fields exported and testable.
> **Disposition:** ACCEPTED — adopted; coupling to `store.RenameOp` accepted over a conversion layer (ADR-1 consequences).
- Return typed batch errors from `internal/store`, such as operation kind, basename, original error, rollback errors, and inconsistent-state flag. Trade-off: more API surface, but it keeps `resolve:` formatting in `cmd/clinban` while preserving exact ticket-required messages.
> **Disposition:** ACCEPTED — adopted as `store.BatchError`.
- Use a temporary staging name strategy with `os.Rename` inside the same directory instead of hard links. Trade-off: it may align better with normal rename semantics, but it needs careful collision handling and recovery for multi-step cycles; it also diverges from the existing `RenameWithinDir` hard-link pattern.
> **Disposition:** REJECTED — `os.Rename` silently overwrites an existing destination, discarding the ErrExist collision-safety that move.go:65-67 and the whole store family rely on. ADR-1 commits to consistency with the established `os.Link + os.Remove` idiom (MoveToArchive/MoveToActive/RenameWithinDir). Staging-name recovery is strictly more complex with no offsetting benefit here.

## Risks
- Data loss risk: the proposed rollback cleanup can delete tickets after a Phase 2 remove failure because some destination links may be the only remaining names for their inodes.
> **Disposition:** ACCEPTED — eliminated by the re-link-then-remove rollback (Execution Model). Correctness NFR row updated to call out the partial-Phase-2 case.
- Compile-time integration risk: the described `st.BatchRenameWithinDir(plan)` call does not match current package types and may lead implementers to force store concerns into `cmd` or duplicate structs poorly.
> **Disposition:** ACCEPTED — removed by exporting `store.RenameOp` and having `planResolve` return it directly.
- Error contract drift risk: if the store returns `store:` errors and the CLI wraps them naively, acceptance tests for exact `resolve: link/remove/rollback` messages may fail.
> **Disposition:** ACCEPTED — typed `store.BatchError` + CLI-owned `resolve:` formatting prevents drift; the CLI builds the exact strings from structured fields.
- Security/path safety risk: an exported batch API without basename and same-directory validation could let future callers create links outside the intended directory, contrary to the local filesystem safety assumptions in `docs/security.md:17-24`.
> **Disposition:** ACCEPTED — basename-only validation in pre-flight (same-directory contract) closes this.
- Test reliability risk: `chmod 0555` remove-failure tests from `tickets/0023-resolve-partial-rename-leaves-inconsistent.md:58-59` may behave differently when run as privileged users or on non-POSIX filesystems; the architecture does not offer an injectable filesystem or failure hook.
> **Disposition:** ACCEPTED — Testability note added: `t.Skip` under `os.Geteuid()==0` or use an injectable failure hook. Hook-vs-skip choice left to the tasks stage.

## Summary verdict: REVISE
The direction in “ADR-1: Batch rename and rollback responsibility — store layer” is aligned with the package boundary, but the rollback design under “NFRs” and “Integration Contracts” is not yet correct enough to implement safely. The artifact must specify a Phase 2 rollback that restores already-removed source paths, resolve the `RenameOp` API/type mismatch, and define store-vs-CLI error ownership before it can satisfy ticket 0023’s all-or-nothing requirement.