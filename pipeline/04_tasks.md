# Developer Tasks
_Produced by: techlead-agent_
_Date: 2026-05-20_
_Status: draft_
_Input: pipeline/03_design.md_

## Task List

### TASK-001: Write cmd/clinban/schema.md
- **Description:** Create the static Markdown reference document at `cmd/clinban/schema.md`. This file is the content that `clinban init` will embed and write to the project root as `SCHEMA.md`. It must be complete and accurate because it is the primary interface between the tool and LLM agents.
- **Module(s):** `cmd/clinban/schema.md` (new file)
- **Done criteria:**
  - [ ] File exists at `cmd/clinban/schema.md`
  - [ ] Section 1 — Intro: explains what Clinban is and instructs the reader to check `.clinban` for the values of `tickets_dir` and `archive_dir`
  - [ ] Section 2 — Ticket format: includes a complete YAML frontmatter example with all seven fields
  - [ ] Section 3 — Fields table: covers `id`, `status`, `type`, `title`, `tags`, `created`, `updated`; each row states required/optional, owner (tool vs human vs agent), and constraints (e.g. RFC3339 for timestamps, zero-padded 4-digit for id)
  - [ ] Section 4 — File naming: documents the `<id>-<slug>.md` convention
  - [ ] Section 5 — Status transitions: lists all five valid transitions (`backlog→in-progress`, `in-progress→blocked`, `in-progress→done`, `blocked→in-progress`, `done→backlog`)
  - [ ] Section 6 — Agent operations: step-by-step instructions for create a ticket, update a ticket, move status, archive
  - [ ] Valid Markdown; no Go changes in this task
- **Depends on:** none
- **Notes:** Write for an LLM reader that has no prior context. Keep instructions imperative and unambiguous. The file is embedded verbatim — no templating.

---

### TASK-002: Update runInit to emit SCHEMA.md
- **Description:** Wire `cmd/clinban/schema.md` into `init.go` using `//go:embed`. Extend `runInit` to stat, guard, and create `SCHEMA.md` as a fourth artifact alongside the existing three.
- **Module(s):** `cmd/clinban/init.go`
- **Done criteria:**
  - [ ] `//go:embed schema.md` directive present; `var schemaMD string` declared at package level
  - [ ] Step 4: `absSchema` and `schemaExists` computed from `filepath.Join(cwd, "SCHEMA.md")`
  - [ ] Step 5 (no-force check): `schemaExists` included; prints `"already exists: SCHEMA.md"` to stderr when true
  - [ ] Step 6 (force full-init guard): condition requires all four — `ticketsExists && archiveExists && configExists && schemaExists`
  - [ ] Step 7: `if !schemaExists` branch calls `os.WriteFile(absSchema, []byte(schemaMD), 0o644)` and prints `"created: SCHEMA.md"` to stdout; error wrapped as `"init: write schema: %w"`
  - [ ] `go build ./cmd/clinban/...` succeeds
  - [ ] `go vet ./cmd/clinban/...` clean
- **Depends on:** TASK-001
- **Notes:** The `//go:embed` directive requires `schema.md` to be in the same directory as the `.go` file. The `import _ "embed"` blank import is required if no other embed is already present in the package.

---

### TASK-003: Update init tests
- **Description:** Update the four existing `TestInit*` functions in `cmd/clinban/init_test.go` to account for the fourth artifact, and add the new `TestInitPartial_SchemaOnly_Force` test.
- **Module(s):** `cmd/clinban/init_test.go`
- **Done criteria:**
  - [ ] `TestInitFreshDirectory`: stdout contains `"created: SCHEMA.md"`; `os.Stat(filepath.Join(dir, "SCHEMA.md"))` succeeds; file size is greater than zero
  - [ ] `TestInitAlreadyExists_WithForce`: pre-creates `SCHEMA.md` alongside the other three artifacts before running `--force`; test still exits 1 with "already fully initialized"
  - [ ] `TestInitPartial_DirsExist_NoConfig_Force`: asserts stdout contains both `"created: .clinban"` and `"created: SCHEMA.md"`
  - [ ] `TestInitPartial_ConfigExists_NoDirs_Force`: asserts stdout contains `"created: SCHEMA.md"`
  - [ ] `TestInitPartial_SchemaOnly_Force` (new): pre-creates `tickets/`, `archive/`, `.clinban`; does NOT pre-create `SCHEMA.md`; runs `clinban init --force`; asserts exit 0; asserts stdout contains `"created: SCHEMA.md"`; asserts stdout does NOT contain `"created: tickets"`, `"created: archive"`, or `"created: .clinban"`
  - [ ] `go test ./cmd/clinban/...` passes
- **Depends on:** TASK-002
- **Notes:** Use the existing subprocess harness pattern for `TestInitPartial_SchemaOnly_Force`. The assertion that no other "created:" lines appear is important — it confirms the partial-init logic correctly skips already-present artifacts.

---

### TASK-004: Update docs
- **Description:** Update `docs/cli.md` to list `SCHEMA.md` as a fourth artifact created by `clinban init`, and append a log entry to `docs/log.md`.
- **Module(s):** `docs/cli.md`, `docs/log.md`
- **Done criteria:**
  - [ ] `docs/cli.md` init section lists `SCHEMA.md` as a fourth artifact with a one-line description of its purpose (human/LLM-readable schema reference)
  - [ ] `docs/log.md` has a new entry for this feature following the existing log entry format
- **Depends on:** TASK-003
- **Notes:** Keep the `docs/cli.md` addition concise — one bullet or table row is sufficient. The log entry should reference the ticket ID (0001) and briefly state what changed.

---

## Dependency Order

```
TASK-001 (cmd/clinban/schema.md content)
    └── TASK-002 (runInit wiring in init.go)
            └── TASK-003 (init_test.go updates + new test)
                    └── TASK-004 (docs/cli.md + docs/log.md)
```

Linear sequence. No task depends on more than one predecessor. Each task is under one hour of work.
