# Clinban — Functional Requirements

## 1. Scope

Clinban is a terminal CLI tool that manages kanban tickets as markdown files in a directory.
It serves two actor classes equally: human developers using interactive commands, and automata
(AI agents, CI/CD pipelines, scripts) that read and write ticket files directly via the schema.
The schema is the authoritative contract. The CLI enforces business rules for human flows;
lint is the integrity layer for machine-written files.

---

## 2. Actors

### Human Developer
Uses the CLI interactively. Subject to FSM transition enforcement via `clinban move`.
Editor-based creation and editing via `clinban new` and `clinban show`.

### Automaton
Any non-human actor: AI agent, CI/CD pipeline, script. Creates tickets via
`clinban new --no-interactive` or `clinban register`. Reads and writes ticket files directly.
Not constrained by FSM transition enforcement — `clinban lint` is their validation tool.

---

## 3. Data Entities

### 3.1 Ticket

A ticket is a markdown file with YAML frontmatter.

**Frontmatter fields:**

| Field | Owner | Required | Type | Constraints |
|---|---|---|---|---|
| `id` | Clinban | yes | string | 4-digit zero-padded (e.g. `"0042"`); unique across repo and archive |
| `status` | Clinban (initial); user via move | yes | string | One of: `backlog`, `in-progress`, `blocked`, `done` |
| `type` | User / automaton | yes | string | One of: `bug`, `task`, `feature`, `spike` |
| `title` | User / automaton | yes | string | Non-empty, not a placeholder value |
| `tags` | User / automaton | no | list of strings | Free-form; empty list permitted |
| `created` | Clinban | yes | timestamp (RFC3339) | Set on creation; never modified after |
| `updated` | Clinban | yes | timestamp (RFC3339) | Set on creation; updated on every Clinban write |

**Body:** Freeform markdown. No constraints.

**File naming:** `<id>-<slug>.md`
- Slug: first 5 words of title, lowercased, spaces replaced with hyphens, non-alphanumeric characters stripped
- Example: `0042-fix-login-timeout-on.md`

**Initial values on creation:**
- `status`: `backlog`
- `created`, `updated`: current timestamp at moment of creation

### 3.2 Directory Structure

```
<tickets_dir>/          # active tickets (configurable; default: current directory)
  0001-first-ticket.md
  0042-fix-login-timeout.md
  archive/              # closed tickets (configurable; default: <tickets_dir>/archive)
    0003-old-ticket.md
```

Active tickets are those not in `archive/`. All ticket files in both directories are considered
part of the repo for ID uniqueness checks.

### 3.3 Configuration

File: `.clinban` in the project root directory. Format: TOML.

```toml
tickets_dir = "tasks"
archive_dir = "tasks/archive"
```

Both fields are optional. Defaults: `tickets_dir` = current working directory,
`archive_dir` = `<tickets_dir>/archive`.

---

## 4. State Machine

Valid transitions:

| From | To | Allowed |
|---|---|---|
| `backlog` | `in-progress` | yes |
| `backlog` | `blocked` | yes |
| `in-progress` | `blocked` | yes |
| `in-progress` | `done` | yes |
| `blocked` | `in-progress` | yes |
| `blocked` | `done` | **no** |
| `done` | `backlog` | yes (reopen) |
| `done` | `in-progress` | **no** |

All other transitions not listed above are invalid.

**Enforcement scope:** FSM transition rules are enforced only by `clinban move`.
Direct file edits (via `clinban show` or external tools) are not transition-checked —
lint validates only that the status value is legal, not that the transition was valid.

---

## 5. Commands

### 5.1 `clinban new` — Interactive Ticket Creation (Human)

**Flow:**
1. Scan `<tickets_dir>` and `<archive_dir>` for highest existing ID; assign next integer (zero-padded to 4 digits). Start at `0001` if no tickets exist.
2. Create a template file in the system temp directory with pre-populated system fields (`id`, `status = backlog`, `created`, `updated`) and placeholder values for required user fields (`title`, `type`).
3. Open the file in `$EDITOR`. Fallback: `vi`.
4. Wait for the editor process to exit.
5. Compare required fields (`title`, `type`) to their placeholder values.
   - If unchanged: discard the temp file. Print "Ticket discarded." Exit.
   - If changed: proceed.
6. Run lint on the file.
7. Move the file from temp to `<tickets_dir>` regardless of lint result.
8. If lint errors: print each error (one per line). Display interactive prompt: `Re-open in editor? [y/N]`.
   - If `y`: re-open the file (now in `<tickets_dir>`) in `$EDITOR`. Return to step 4.
   - If `n`: exit.
9. If no lint errors: print ticket ID and filename. Exit cleanly.

**Edge cases:**
- `$EDITOR` is unset: fall back to `vi`
- Editor exits without saving: file unchanged, discard
- Temp directory not writable: fatal error with message
- ID collision (race condition with concurrent write): out of scope for v1

---

### 5.2 `clinban new --no-interactive` — Non-Interactive Ticket Creation (Automata)

**Flags:**

| Flag | Required | Description |
|---|---|---|
| `--title` | yes | Ticket title |
| `--type` | yes | One of the valid type values |
| `--body` | no | Markdown body content |
| `--tags` | no | Comma-separated list of tags |

**Flow:**
1. Validate that `--title` and `--type` are provided and non-empty. If not: print error, exit 1.
2. Assign next ID (same scan logic as interactive flow).
3. Construct ticket file in memory with all fields populated and body from `--body` if provided.
4. Run lint. If lint errors: print errors (one per line), exit 1.
5. Write file to `<tickets_dir>`.
6. Print ticket ID and filename, exit 0.

---

### 5.3 `clinban register <path>` — File Adoption (Automata)

Adopts an externally created ticket file into the registry.

**Flow:**
1. Read the file at `<path>`. If not found: print "File not found", exit 1.
2. Overwrite `id` with next assigned ID (same scan logic).
3. Overwrite `created` and `updated` with current timestamp.
4. Run lint on the resulting file.
5. If lint errors: print errors (one per line), exit 1. Do not move the file.
6. If no errors: move file to `<tickets_dir>`. Print ticket ID and filename. Exit 0.

**Edge cases:**
- File has pre-existing `id`, `created`, `updated`: overwritten without warning
- File is not valid YAML frontmatter at all: parse fails before lint runs; print parse error, exit 1
- `<path>` is already inside `<tickets_dir>`: out of scope for v1

---

### 5.4 `clinban list` — List Active Tickets

Lists all tickets in `<tickets_dir>` (not `archive/`).

**Default output:** All active tickets, sorted by status (`in-progress` → `blocked` → `backlog` → `done`), then by ID ascending within each group. One line per ticket, truncated to terminal width.

**Output columns (in order):** ID · status · type · title

**Filters (combinable):**

| Flag | Description |
|---|---|
| `--status <value>` | Filter by exact status value |
| `--type <value>` | Filter by exact type value |
| `--tag <value>` | Filter tickets containing the given tag |

**Edge cases:**
- No active tickets: print "No active tickets"
- No tickets match filters: print "No active tickets"
- Title longer than terminal width: truncate with ellipsis

---

### 5.5 `clinban show <id>` — View a Ticket

Prints the ticket to stdout in human-readable format. Does not open an editor and does not modify any files.

**Output format:** One labeled line per field, then a blank line followed by the body (if non-empty):

- `ID:      <id>`
- `Status:  <status>`
- `Type:    <type>`
- `Title:   <title>`
- `Tags:    <tag1>, <tag2>` (line omitted when Tags is empty)
- `Created: <RFC3339>`
- `Updated: <RFC3339>`
- `[archived]` (only if ticket is in `archive/`)

**Edge cases:**
- ID not found: print "Ticket not found", exit 1
- Ticket is in `archive/`: print with `[archived]` label; no files are modified

---

### 5.6 `clinban edit <id>` — Edit a Ticket

Opens the ticket file in `$EDITOR` (fallback: `vi`). Writes back only when parse and lint both pass.

**Flow:**
1. Open file in `$EDITOR`. Wait for editor to exit.
2. Re-read file from disk. Parse frontmatter.
   - If parse error: print error, display `Re-open in editor? [y/N]` prompt. If `n`: exit 1.
3. Run lint.
   - If lint errors: print errors, display `Re-open in editor? [y/N]` prompt. If `n`: exit 1.
4. Set `updated` to current timestamp. Write file atomically. Exit 0.

**Edge cases:**
- ID not found: print "Ticket not found", exit 1
- Ticket is in `archive/`: open from archive; write back to archive path after a valid edit

---

### 5.7 `clinban move <id> <status>` — Transition Ticket Status

Moves a ticket to a new status, enforcing FSM rules.

**Flow:**
1. Resolve ticket by ID. If not found: print "Ticket not found", exit 1.
2. Read current status.
3. If target status equals current status: no-op, exit 0 silently.
4. If transition is not in the valid transitions table: print suggested valid next statuses. Example: `"blocked" cannot transition to "done". Valid transitions: in-progress`. Exit 1.
5. Update status field and `updated` timestamp. Write file.
6. Print confirmation: `0042 moved to done`.

**Edge cases:**
- Ticket in `archive/`: locate and operate on it there; `done → backlog` moves file back to `<tickets_dir>`
- Invalid status value as argument: print valid status list, exit 1

---

### 5.8 `clinban archive` — Move Done Tickets to Archive

**`clinban archive`** (bulk):
1. Scan `<tickets_dir>` for tickets with `status = done`.
2. If none found: print "No done tickets to archive". Exit 0.
3. List matching tickets. Prompt: `Archive N ticket(s)? [y/N]`.
4. If `y`: move all to `<archive_dir>`. Print count. Exit 0.
5. If `n`: exit 0.

**`clinban archive <id>`** (single):
1. Resolve ticket by ID. If not found: print "Ticket not found", exit 1.
2. If status is not `done`: print "Ticket must be in `done` status to archive", exit 1.
3. Move file to `<archive_dir>`. Print confirmation. Exit 0.

**Edge cases:**
- `<archive_dir>` does not exist: create it automatically
- File with same name already exists in `<archive_dir>`: out of scope for v1

---

### 5.9 `clinban lint` — Schema Validation

Validates ticket files against the schema.

**`clinban lint`** — validates all tickets in `<tickets_dir>` and `<archive_dir>`.

**`clinban lint <id>`** — validates a single ticket.
- If ID not found: print "Ticket not found", exit 1.

**Checks performed (in order):**

| # | Check |
|---|---|
| 1 | Required fields present: `id`, `status`, `title`, `type`, `created`, `updated` |
| 2 | `status` is a valid value |
| 3 | `type` is a valid value |
| 4 | `id` matches the numeric portion of the filename |
| 5 | `created` and `updated` are valid RFC3339 timestamps |
| 6 | `tags`, if present, is a list of strings |
| 7 | `id` is unique across all tickets in `<tickets_dir>` and `<archive_dir>` |

**Output format:** One line per error:
```
0042-fix-login-timeout.md: missing required field 'type'
0042-fix-login-timeout.md: id does not match filename
```

**Exit codes:** 0 if no errors, 1 if any errors found.

---

## 6. Business Rules

1. Clinban exclusively owns `id`, `created`, and `updated`. These fields are always overwritten by Clinban on creation or registration — external values are ignored.
2. FSM transition enforcement applies only to `clinban move`. No other command checks transitions.
3. A ticket in any status may be edited freely via `clinban edit` or direct file write; lint validates field values, not transitions.
4. Only tickets with `status = done` may be archived.
5. `done → backlog` is the only valid reopen path. A done ticket cannot go directly to `in-progress`.
6. `blocked` tickets must transition through `in-progress` before reaching `done`.
7. ID uniqueness is enforced across both active and archived tickets.
8. Ticket files are identified by schema, not filename pattern; the filename is derived from the ticket content at creation time and is not renamed on updates.
9. The `.clinban` config file, if absent, causes all paths to use defaults silently — no error.
10. Lint errors block writes in all non-interactive flows (`clinban new --no-interactive`, `clinban register`, `clinban edit`). In interactive creation (`clinban new`), the new file is written regardless of lint result and the user is prompted to repair.

---

## 7. Acceptance Criteria

**`clinban new` (interactive)**
- Given a valid filled template, when the editor closes, then a ticket file appears in `<tickets_dir>` with correct frontmatter and a unique 4-digit ID.
- Given an unchanged template, when the editor closes, then no file is written and "Ticket discarded." is printed.
- Given a template with lint errors, when the editor closes, then the file is written, errors are listed, and the user is prompted to re-open.

**`clinban new --no-interactive`**
- Given valid `--title` and `--type` flags, when invoked, then a ticket is written to `<tickets_dir>` and the ID and filename are printed to stdout with exit 0.
- Given missing `--title`, when invoked, then an error is printed and exit code is 1.

**`clinban register <path>`**
- Given a valid file at `<path>`, when invoked, then `id`, `created`, `updated` are overwritten by Clinban, the file passes lint, is moved to `<tickets_dir>`, and exit code is 0.
- Given a file that fails lint after system fields are filled, when invoked, then errors are printed, the file is not moved, and exit code is 1.

**`clinban list`**
- Given active tickets in multiple statuses, when invoked with no flags, then output is sorted `in-progress` → `blocked` → `backlog` → `done`, one line per ticket, showing ID · status · type · title.
- Given `--status in-progress`, when invoked, then only `in-progress` tickets are shown.
- Given no active tickets, when invoked, then "No active tickets" is printed.

**`clinban show <id>`**
- Given a valid ID, when invoked, then ticket fields and body are printed to stdout and exit code is 0.
- Given an archived ticket ID, when invoked, then the ticket is printed from archive with `[archived]` label.
- Given an unknown ID, when invoked, then "Ticket not found" is printed and exit code is 1.

**`clinban edit <id>`**
- Given a valid ID, when invoked, then the ticket opens in `$EDITOR`; on close, if parse and lint succeed, `updated` is refreshed and the file is written atomically.
- Given a valid ID, when the user saves with lint errors, then errors are printed and the user is prompted to re-open.
- Given an unknown ID, when invoked, then "Ticket not found" is printed and exit code is 1.

**`clinban move <id> <status>`**
- Given ticket `0042` in `in-progress`, when `clinban move 0042 done` is run, then status is updated and confirmation is printed.
- Given ticket `0042` in `blocked`, when `clinban move 0042 done` is run, then an error and valid transition list are printed, exit code is 1.
- Given ticket `0042` already in `in-progress`, when `clinban move 0042 in-progress` is run, then the command exits silently with code 0.

**`clinban archive`**
- Given two tickets with `status = done`, when `clinban archive` is run and confirmed, then both files are moved to `<archive_dir>`.
- Given ticket `0042` with `status = in-progress`, when `clinban archive 0042` is run, then "Ticket must be in `done` status to archive" is printed and exit code is 1.
- Given no done tickets, when `clinban archive` is run, then "No done tickets to archive" is printed and exit code is 0.

**`clinban lint`**
- Given a ticket missing the `type` field, when `clinban lint` is run, then a line identifying the file and the missing field is printed and exit code is 1.
- Given all tickets valid, when `clinban lint` is run, then nothing is printed and exit code is 0.
- Given two tickets with the same `id` value, when `clinban lint` is run, then both files are flagged for ID collision.

---

## 8. Out of Scope (v1)

- Git integration of any kind
- Web UI or browser-based access
- Authentication or multi-user access control
- Time tracking, estimation, burndown, or velocity metrics
- Notifications or reminders
- Multiple board configurations
- Concurrent write safety (two processes writing simultaneously)
- Renaming ticket files when the title is updated
- Handling filename collisions in `archive/`
- Constraining FSM transitions for automata
- Extended lint error behaviours beyond list + exit (noted for future iteration)
- `clinban register` where `<path>` is already inside `<tickets_dir>`
