# Implementation Design
_Produced by: techlead-agent_
_Date: 2026-05-22_
_Status: draft_
_Input: design session (ticket 0019 — clinban init SKILL.md artifact)_

## Module Structure

### cmd/clinban — init command

**Files:**
- `cmd/clinban/init.go` — extends `runInit` to create a 5th artifact: `.claude/skills/tickets/SKILL.md`
- `cmd/clinban/skills/tickets/SKILL.md` — embedded asset containing the tickets skill content (copy of `.claude/skills/tickets/SKILL.md`)
- `cmd/clinban/init_test.go` — integration tests against the compiled binary; extended with 4 new test cases

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|----------------|
| `runInit` | `(flags initFlags) error` | Creates all 5 init artifacts with pre-flight, force, and fully-initialized logic |
| `skillMD` | `var skillMD string` (embed) | Holds the raw content of `skills/tickets/SKILL.md` at compile time |

**Interface contract:**
- Accepts: `initFlags` struct (existing fields unchanged; no new flags required)
- Returns: `error` — wrapped with context prefix `"init: ..."` on failure
- Errors:
  - `"init: create skill dir: %w"` — returned when `os.MkdirAll` fails for `.claude/skills/tickets/`
  - `"init: write skill file: %w"` — returned when `os.WriteFile` fails for `SKILL.md`

**Pre-flight check (5th artifact):**
```go
absSkillFile := filepath.Join(cwd, ".claude", "skills", "tickets", "SKILL.md")
_, errSkill := os.Stat(absSkillFile)
skillFileExists := errSkill == nil
```

**Creation logic (5th artifact):**
```go
if !skillFileExists {
    if err := os.MkdirAll(filepath.Dir(absSkillFile), 0o755); err != nil {
        return fmt.Errorf("init: create skill dir: %w", err)
    }
    if err := os.WriteFile(absSkillFile, []byte(skillMD), 0o644); err != nil {
        return fmt.Errorf("init: write skill file: %w", err)
    }
    fmt.Println("created: .claude/skills/tickets/SKILL.md")
}
```

**Reporting strings:**
- `"already exists: .claude/skills/tickets/SKILL.md"`
- `"missing: .claude/skills/tickets/SKILL.md"`

**Fully-initialized guard:**
The boolean expression covering all 5 artifacts (no-force path and force path) must include `skillFileExists`.

## Inter-Component Communication

| From | To | Method | Data |
|------|----|--------|------|
| `runInit` | filesystem | `os.MkdirAll` + `os.WriteFile` | embedded `skillMD` string written to `.claude/skills/tickets/SKILL.md` |
| `init.go` | `skills/tickets/SKILL.md` | `//go:embed skills/tickets/SKILL.md` | raw Markdown bytes bound to package-level `skillMD` var |

## Test Strategy

All tests are integration tests that invoke the compiled binary, matching the existing pattern in `init_test.go`.

**Critical paths (must be tested before first ship):**
1. Fresh init creates all 5 artifacts; `.claude/skills/tickets/SKILL.md` exists and has non-empty content
2. Without `--force`, an existing `SKILL.md` causes exit code 1 with `"already exists: .claude/skills/tickets/SKILL.md"` on stderr
3. With `--force`, an existing `SKILL.md` is skipped; artifacts that are missing are still created
4. All 5 artifacts already exist plus `--force` → binary reports `"already fully initialized"` and exits 0

**Unit tests (per module):**
- `cmd/clinban`: no isolated unit tests required beyond the 4 integration scenarios above; logic is exercised end-to-end through the binary

**Integration tests:**
- Run `go test ./...` from repo root; all existing init tests must remain green after changes

## Resolved Architecture Questions

| Question (from 02_architecture.md) | Decision | Rationale |
|------------------------------------|----------|-----------|
| Where to store the embeddable asset? | `cmd/clinban/skills/tickets/SKILL.md` | Go `//go:embed` requires the path to be within or below the package directory; `.claude/` is outside `cmd/clinban/` |
| Should `--force` overwrite an existing SKILL.md? | No — skip (same as other artifacts) | Consistent with existing force behaviour: force creates missing artifacts, does not overwrite present ones |
