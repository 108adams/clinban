# Developer Tasks
_Produced by: techlead-agent_
_Date: 2026-05-22_
_Status: draft_
_Input: pipeline/03_design.md_

## Task List

### TASK-001: Add SKILL.md artifact to `clinban init`
- **Description:** Add `.claude/skills/tickets/SKILL.md` as the 5th artifact created by `clinban init`. Copy the skill content from `.claude/skills/tickets/SKILL.md` into the new embedded asset at `cmd/clinban/skills/tickets/SKILL.md`. Declare a `//go:embed` var in `init.go`. Extend `runInit` with the same pre-flight existence check, `os.MkdirAll` + `os.WriteFile` creation block, force-skip logic, fully-initialized guard, and reporting strings used by the existing 4 artifacts. Add 4 integration tests to `init_test.go`. Update `docs/cli.md` and append to `docs/log.md`.
- **Module(s):**
  - `cmd/clinban/init.go` (modify)
  - `cmd/clinban/skills/tickets/SKILL.md` (create — copy from `.claude/skills/tickets/SKILL.md`)
  - `cmd/clinban/init_test.go` (modify)
  - `docs/cli.md` (modify)
  - `docs/log.md` (append)
- **Done criteria:**
  - [ ] `cmd/clinban/skills/tickets/SKILL.md` exists and contains the tickets skill content (copied verbatim from `.claude/skills/tickets/SKILL.md`)
  - [ ] `init.go` declares `//go:embed skills/tickets/SKILL.md` with a package-level `skillMD` var
  - [ ] `runInit` computes `absSkillFile`, checks `skillFileExists` via `os.Stat`, and creates the file with `os.MkdirAll` + `os.WriteFile` when absent
  - [ ] Pre-flight reporting includes `"already exists: .claude/skills/tickets/SKILL.md"` and `"missing: .claude/skills/tickets/SKILL.md"`
  - [ ] Fully-initialized guard (both no-force and force paths) includes `skillFileExists`
  - [ ] 4 new integration tests pass: fresh-init creates 5th artifact, no-force exits 1 on existing SKILL.md, force skips existing SKILL.md, all-five-plus-force reports fully initialized
  - [ ] `go test ./...` green
  - [ ] `gofmt -l .` produces no output
  - [ ] `docs/cli.md` artifact list for `clinban init` includes `.claude/skills/tickets/SKILL.md`
  - [ ] `docs/log.md` has a new entry for this change
- **Depends on:** none
- **Notes:** The embedded asset path must be relative to `cmd/clinban/`; the source file `.claude/skills/tickets/SKILL.md` lives at repo root and cannot be referenced directly by `//go:embed`. Create the copy at `cmd/clinban/skills/tickets/SKILL.md` before adding the embed directive, or the build will fail.

## Dependency Order

```
TASK-001  (no prerequisites — self-contained)
```
