# Implementation Design
_Produced by: techlead-agent_
_Date: 2026-05-20_
_Status: draft_
_Input: design summary for ticket 0001 (SCHEMA.md for LLM agents)_

## Module Structure

### cmd/clinban/schema.md (new embedded file)

**Files:**
- `cmd/clinban/schema.md` — static Markdown reference document embedded at compile time; written verbatim to `SCHEMA.md` in the project root by `clinban init`

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|---------------|
| `schemaMD` | `string` (package-level var) | Holds the embedded content of `schema.md` via `//go:embed`; zero runtime cost |

**Interface contract:**
- Accepts: N/A — purely static content read at compile time
- Returns: N/A — consumed as a `string` by `runInit`
- Errors: none; any embed failure is a compile error, not a runtime error

**Required sections in schema.md:**
1. Intro — what Clinban is; how to locate `tickets_dir` and `archive_dir` from `.clinban`
2. Ticket format — complete YAML frontmatter example
3. Fields table — `id`, `status`, `type`, `title`, `tags`, `created`, `updated`; for each: required/optional, owner, constraints
4. File naming convention — `<id>-<slug>.md` pattern
5. Status transitions — `backlog→in-progress`, `in-progress→blocked`, `in-progress→done`, `blocked→in-progress`, `done→backlog`
6. Agent operations — create a ticket, update a ticket, move status, archive

---

### cmd/clinban/init.go (modified)

**Files:**
- `cmd/clinban/init.go` — manages creation of four project artifacts; extended to emit `SCHEMA.md`

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|---------------|
| `runInit` | `(flags initFlags) → error` | Orchestrate creation of tickets dir, archive dir, `.clinban` config, and `SCHEMA.md`; signature unchanged |
| `schemaMD` | `var schemaMD string` | Package-level embedded string declared in this file |

**Interface contract:**
- Accepts: `initFlags` struct (unchanged)
- Returns: `nil` on success; wrapped error on any I/O failure
- Errors:
  - `"init: write schema: %w"` — `os.WriteFile` failure when writing `SCHEMA.md`
  - Existing error prefixes for tickets dir, archive dir, and config are unchanged

**Step-by-step changes to `runInit`:**

Step 4 (pre-flight stat):
- Add `absSchema := filepath.Join(cwd, "SCHEMA.md")`
- Add `_, errSchema := os.Stat(absSchema); schemaExists := errSchema == nil`

Step 5 (no-force early-exit check):
- Include `schemaExists` in the already-initialized guard
- Print `"already exists: SCHEMA.md"` to stderr when true

Step 6 (force full-init guard):
- Require all four artifacts to exist before treating the project as "already fully initialized":
  `ticketsExists && archiveExists && configExists && schemaExists`

Step 7 (creation):
- Add branch: `if !schemaExists { os.WriteFile(absSchema, []byte(schemaMD), 0o644); print "created: SCHEMA.md" }`
- Error wrap: `fmt.Errorf("init: write schema: %w", err)`
- File permissions: `0o644` (world-readable documentation)

**Embed directive (top of file, alongside any existing embed directives):**
```go
//go:embed schema.md
var schemaMD string
```

---

### cmd/clinban/init_test.go (modified)

**Files:**
- `cmd/clinban/init_test.go` — existing subprocess-based test file; updated and extended

**Key types / functions:**

| Name | Signature | Responsibility |
|------|-----------|---------------|
| `TestInitFreshDirectory` | `(t *testing.T)` | Fresh dir: assert all four artifacts created, SCHEMA.md non-empty |
| `TestInitAlreadyExists_WithForce` | `(t *testing.T)` | All four present: `--force` exits 1 with "already fully initialized" |
| `TestInitPartial_DirsExist_NoConfig_Force` | `(t *testing.T)` | Dirs exist, no config, no schema: `--force` creates both `.clinban` and `SCHEMA.md` |
| `TestInitPartial_ConfigExists_NoDirs_Force` | `(t *testing.T)` | Config exists, no dirs, no schema: `--force` creates dirs and `SCHEMA.md` |
| `TestInitPartial_SchemaOnly_Force` | `(t *testing.T)` | NEW — three artifacts present, no schema: `--force` creates only `SCHEMA.md` |

**Interface contract (test perspective):**
- All tests run `clinban init` as a subprocess via the existing test harness
- `TestInitPartial_SchemaOnly_Force` pre-creates `tickets/`, `archive/`, and `.clinban`; does NOT pre-create `SCHEMA.md`; asserts exit 0 and stdout contains exactly `"created: SCHEMA.md"` with no dir/config creation messages

---

### docs/cli.md + docs/log.md (updated)

**Files:**
- `docs/cli.md` — CLI reference; `init` section updated to list `SCHEMA.md` as fourth artifact
- `docs/log.md` — append one entry for this feature

**Interface contract:**
- `docs/cli.md` init section must mention `SCHEMA.md` with a one-line description of its purpose
- `docs/log.md` entry format follows the existing log convention

---

## Inter-Component Communication

| From | To | Method | Data |
|------|----|--------|------|
| `cmd/clinban/init.go runInit` | `cmd/clinban/schema.md` (embedded) | `//go:embed` + `os.WriteFile` | `schemaMD string` → `[]byte` written to `SCHEMA.md` |
| LLM agent / human | `SCHEMA.md` (project root) | file read at runtime | Markdown text; agent locates dirs via `.clinban` |

---

## Test Strategy

**Unit tests (per module):**
- `cmd/clinban`: all init tests via subprocess harness in `init_test.go`

**Critical paths (must be tested before first ship):**
1. `TestInitFreshDirectory` — `clinban init` on a clean directory creates `SCHEMA.md`, the file exists on disk, and its size is greater than zero
2. `TestInitAlreadyExists_WithForce` — when all four artifacts are present, `--force` exits 1 with "already fully initialized" (no regression from adding the fourth guard)
3. `TestInitPartial_SchemaOnly_Force` — when only `SCHEMA.md` is missing, `--force` creates it and prints no other "created:" lines

**Integration tests:**
- No new integration tests required; the existing subprocess harness in `cmd/clinban` covers end-to-end behavior for all three critical paths above

---

## Resolved Architecture Questions

| Question | Decision | Rationale |
|----------|----------|-----------|
| New package vs embed in existing `cmd/clinban`? | No new package; `//go:embed schema.md` in `init.go` | The schema is a single static file with one consumer; a new package would add indirection with no benefit |
| Where does the LLM read dir paths from? | LLM reads `.clinban` at runtime | `SCHEMA.md` is static; dynamic paths (tickets dir, archive dir) are in `.clinban` which the agent reads separately |
| File permissions for `SCHEMA.md`? | `0o644` | Documentation file; world-readable is appropriate and consistent with other static project files |
| Scope: is `schema.md` content defined in this sprint? | Yes — content is part of TASK-001 and must include all six required sections | The document has no value if it is empty or incomplete at ship time |
