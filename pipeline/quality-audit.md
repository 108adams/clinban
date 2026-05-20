---
title: Clinban Codebase Quality Audit
author: Tech Lead review
date: 2026-05-20
scope: Full codebase (49 Go files, ~7.9k LOC) — practices, security, test quality
---

# Clinban Quality Audit

## Executive Summary

Clinban is a well-engineered small Go CLI. Package boundaries are clean and match
the documented architecture, error wrapping is consistent, Go docs are thorough,
and the test suite is broad (≈199 test functions) and exercises error and edge
paths — not just the golden path. `go vet ./...` is clean and `gofmt` reports no
diffs.

The findings below are mostly **medium and low severity**. There are **no
exploitable security vulnerabilities** under the project's stated single-user,
local-filesystem trust model. The most legitimate gaps are around **crash
durability of writes**, an **unprotected overwrite race in the move/archive
path** that contradicts a documented safety invariant, and **test/tooling
hygiene** (untested `editor`/`template` packages, a misleading coverage signal,
and an un-`tidy`'d `go.mod`).

**Overall grade: B+ / strong.** Address the P0/P1 items below to reach A.

| Area | Assessment |
|------|------------|
| Architecture & boundaries | Excellent — boundaries documented and respected |
| Security (local-CLI model) | Good — no real holes; minor hardening notes |
| Data integrity | Good, with two real gaps (fsync, overwrite race) |
| Test quality | Good breadth & error-path coverage; 3 concrete gaps |
| Tooling/hygiene | Needs attention (go.mod, no linter, os.Exit pattern) |

---

## Methodology

- Read every non-test source file across `cmd/clinban` and the eight `internal/`
  packages.
- Read representative test files; counted and classified test functions.
- Ran `go vet ./...`, `gofmt -l .`, `go test -cover ./...`, and a `go mod tidy`
  dry-run.
- Evaluated against AGENTS.md review priorities: ticket-file data integrity, safe
  filesystem behavior, clear CLI output/exit codes, schema compatibility,
  realistic-fixture tests, and doc alignment.

---

## Strengths (keep doing)

- **Clean dependency direction.** `internal/ticket`, `internal/lint`,
  `internal/fsm` own pure domain logic and do not import `internal/store`; the CLI
  layer coordinates. Matches AGENTS.md exactly.
- **Consistent error wrapping** with `%w` and package-prefixed messages
  (`store: ...`, `ticket: parse: ...`), and sentinel errors (`ErrNotFound`,
  `ErrMissingFrontmatter`, `ErrMalformedConfig`) used with `errors.Is`.
- **Parse-vs-lint separation** is preserved: `ReadTicket` parses; lint runs
  separately; rule 2/3 deliberately defer to rule 1 on empty values.
- **Atomic write pattern** (temp-file-in-target-dir + rename + `0o600` chmod) is
  applied consistently in `store.WriteTicket` and the interactive `new` flow.
- **Tests use realistic filesystem fixtures** and a black-box subprocess harness
  that verifies real exit codes and stdout/stderr routing. Error paths (not found,
  invalid type/status, parse failure, lint failure) and edge cases (template
  discard, lint-error reopen loop, sequential ID assignment, non-matching files
  ignored) are all covered.

---

## Findings

Severity key: **P0** fix before further feature work · **P1** fix soon · **P2**
opportunistic / nice-to-have.

### Security (threat model: single-user, local files, user-controlled `$EDITOR`)

**S-1 (P2 / informational) — `$EDITOR` and `register <path>` are trusted by
design.** `editor.Open` execs `$EDITOR` and `register` reads then deletes an
arbitrary source path. Both are intentional per AGENTS.md ("treat `$EDITOR` as
user-controlled"; do not sandbox). No action required, but worth an explicit line
in `docs/security.md` confirming these are accepted, in-scope behaviors so future
reviewers don't "fix" them.

**S-2 (P2) — Path-containment check in `register` is sound but fragile.**
`register.go:93` guards with `strings.HasPrefix(rel, "..")`. Traversal is already
impossible because `slug.Slugify` strips everything outside `[a-z0-9-]`, so the
filename can never contain `/` or `..`. The check is therefore defense-in-depth.
Recommend `rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator))`
to avoid a false match on a (currently impossible) sibling like `..foo`. Low
priority given the slug guarantee.

**S-3 (P2) — YAML/TOML parsing of untrusted-ish files.** Ticket frontmatter is
parsed with `gopkg.in/yaml.v3` and `.clinban` with BurntSushi/toml. Files are
trusted per the model, but if Clinban ever ingests files from CI/agents, note that
yaml.v3 mitigates alias bombs but not arbitrarily large documents. No change now;
flag if the trust model widens.

> **Net: no exploitable vulnerability found.** State this explicitly in the audit
> trail so the "MUST audit security" requirement is satisfied with a documented
> negative result, not silence.

### Data Integrity & Correctness

**D-1 (P1) — Atomic write is not crash-durable (no fsync).**
`store.WriteTicket` (`internal/store/write.go`) writes the temp file, closes it,
then `os.Rename`s. There is no `tmp.Sync()` before close and no `fsync` of the
parent directory after rename. `rename(2)` guarantees name atomicity, but on power
loss/crash the renamed file's *contents* and the directory entry may not be
durable, risking a zero-length or truncated ticket after recovery. For a tool
whose #1 stated priority is ticket-file data integrity, this is the most
legitimate gap. Fix: `tmp.Sync()` before `Close()`, and after `os.Rename` open the
parent dir and `Sync()` it. ~6 lines; add a test that asserts content is fully
flushed.

**D-2 (P1) — `MoveToArchive`/`MoveToActive` can overwrite the destination
(TOCTOU), violating a documented invariant.** AGENTS.md says "Do not overwrite
archive/active destination files silently," and the code attempts to honor it via
`os.Stat(dest)` then `os.Rename` (`internal/store/move.go:19` and `:35`). But
`os.Rename` on POSIX silently overwrites an existing file, and there is a
check-then-act race between `Stat` and `Rename`. A concurrent or interleaved
invocation can clobber a ticket. Single-user likelihood is low, but the invariant
is safety-critical and the protection is non-atomic. Fix: use `os.Link(src, dest)`
+ `os.Remove(src)` (Link fails if dest exists), or `unix.Renameat2` with
`RENAME_NOREPLACE` where available. **Also see T-3 — this branch is untested.**

**D-3 (P2) — Duplicate lint errors for zero timestamps.** `ruleTimestampsNonZero`
(`internal/lint/rules.go:99`) comments that it "only checks fields that were not
already flagged by rule 1," but the code does not implement that guard. A ticket
with a missing `created`/`updated` therefore emits *two* messages for the same
field ("required field missing" from rule 1 and "zero timestamp…" from rule 5).
Confusing CLI output. Fix: either skip rule 5 when the field is the zero value AND
rule 1 already fired, or merge the two messages.

**D-4 (P2) — Empty slug from a non-empty title yields `0001-.md`.** A title made
entirely of non-ASCII or punctuation tokens (e.g. all-CJK) slugs to `""`, so the
filename becomes `<id>-.md`. It still matches the ID regex and works, but the
filename is degenerate and `ruleIDMatchesFilename` still passes. Consider a
fallback slug (e.g. `ticket`) when `Slugify` returns empty, and add a slug test
case for this input.

### Test Quality

**T-1 (P1) — `internal/editor` and `internal/template` have zero tests (0%
coverage).** `template.New` has untested parse/execute error branches, and
`editor.Open` has an untested `$EDITOR`-empty→`vi` fallback and error-propagation
path. `editor` is testable by pointing `$EDITOR` at a stub script (the cmd tests
already do this) or `/bin/true`/`/bin/false`; `template` is trivially testable.
Add focused unit tests.

**T-2 (P1) — Coverage signal for `cmd/clinban` is misleading (reports 4.5%).**
The command tests are black-box subprocess tests (`exec.Command(bin, …)`), so the
standard in-process coverage instrumentation never sees the command code. Behavior
coverage is actually high, but the *metric* is near-useless and won't catch an
untested new branch. Fix: build the binary with `-cover` and aggregate via
`GOCOVERDIR` (Go 1.20+ binary coverage), so subprocess execution contributes to a
real coverage number. This is a process gap, not a behavior gap.

**T-3 (P1) — The "refuse to overwrite destination" safety branch is untested at
the store level.** `store_test.go` covers `MoveToArchiveCreatesDir`,
`PreservesFilename`, and `MoveToActive`, but no test exercises the
"destination already exists" rejection in either move function. This is exactly
the invariant flagged in D-2. Add `TestMoveToArchiveRefusesExistingDestination`
and the active-direction equivalent; they will also pin the D-2 fix.

**T-4 (P2) — `WriteTicket` error paths untested.** Marshal failure and
temp-file-create failure (e.g. unwritable dir) are not exercised. Low priority but
cheap to add via a read-only directory fixture.

### Code Quality & Hygiene

**Q-1 (P1) — `go.mod` is not `tidy`.** All requires are marked `// indirect`, but
`cobra`, `BurntSushi/toml`, `yaml.v3`, and `x/term` are direct imports. A
`go mod tidy` dry-run reorganizes the file into proper direct/indirect blocks.
Run `go mod tidy` and commit. Add a CI check (`go mod tidy && git diff --exit-code
go.mod go.sum`) to keep it honest.

**Q-2 (P1) — Pervasive `os.Exit` inside command `RunE` functions (~22 sites).**
Commands mix returning `error` with calling `os.Exit(1)` directly (e.g.
`new.go` validation, `lint.go:106`, `move.go`, `register.go`). Consequences:
- It is the root cause of the subprocess-only test style (T-2) — these functions
  cannot be unit-tested in-process because they kill the test runner.
- It bypasses deferred cleanup (e.g. `edit.go`'s `defer os.Remove(tmpPath)` is
  fine because exit happens before the defer is registered, but the pattern is
  fragile).
- It produces dead code such as `reportLintErrors`'s `return nil // unreachable`.
Recommend funnelling failures through returned errors and centralizing exit-code
mapping in `main`/`Execute` (e.g. a typed `exitError`). This unlocks in-process
table tests for commands. Medium-sized refactor; do it incrementally per command.

**Q-3 (P2) — No static-analysis tooling configured.** Only `go vet` is used (and
passes). Adding `staticcheck` or `golangci-lint` to the dev workflow / CI would
catch the `// indirect` drift, the unreachable return, and future issues
automatically.

**Q-4 (P2) — `formatRecord` mixes byte length and rune count.** `list.go:165`
computes `prefixLen := len(prefix)` (bytes) and subtracts from `width` to size a
rune-based title budget. Safe today only because `id`/`status`/`type` are ASCII.
If the prefix format ever includes non-ASCII, truncation math breaks. Use
`utf8.RuneCountInString(prefix)` for clarity and future-proofing.

**Q-5 (P2) — Bulk archive has no partial-failure reporting.** `runArchiveBulk`
(`archive.go:114`) moves tickets in a loop and returns on the first error after
having already moved some; the user isn't told which succeeded. Consider
collecting per-ticket results and printing a summary. Low priority for a
single-user tool.

---

## Prioritized Action List

### P0 — none
No defect rises to "block all work." (The closest, D-1/D-2, are P1.)

### P1 — address before/with next feature work
1. **D-1** Add `fsync` of temp file + parent dir to `store.WriteTicket`.
2. **D-2 + T-3** Make destination-overwrite refusal atomic (`os.Link` /
   `RENAME_NOREPLACE`) and add the missing store-level collision tests.
3. **T-1** Add unit tests for `internal/editor` and `internal/template`.
4. **T-2** Wire `-cover` + `GOCOVERDIR` so `cmd/clinban` coverage is real.
5. **Q-1** `go mod tidy`; add a CI guard.
6. **Q-2** Begin replacing in-command `os.Exit` with returned errors + a single
   exit-code mapper (incremental, per command).

### P2 — opportunistic
- D-3 (dedupe zero-timestamp lint output), D-4 (empty-slug fallback),
  S-2/S-3 (hardening notes), T-4 (`WriteTicket` error paths),
  Q-3 (add linter), Q-4 (`formatRecord` rune math), Q-5 (bulk-archive summary).

---

## Notes for the Maintainer

- AGENTS.md asks contributors not to resurrect `pipeline/`. This report was placed
  here per the explicit audit request; consider relocating durable conclusions
  into `docs/` (e.g. a `docs/security.md` line for S-1 and a `docs/log.md` entry)
  and treating this file as transient.
- Every finding is paired with a concrete fix and, where relevant, a test to add,
  so each can be turned directly into a ticket.
