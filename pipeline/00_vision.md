# Clinban — Product Vision

## Vision Statement

For individual developers and small teams building software in code repositories, who lose time
and context managing work across scattered markdown files with no state tracking or shared schema —
Clinban is a terminal-native kanban system where every ticket is a structured markdown file living
alongside the code it describes. Unlike Trello or GitHub Issues, Clinban requires no external
service, no browser, and no proprietary format. Tickets are files: readable, writable, and
versionable by humans, AI agents, and CI/CD pipelines equally. Clinban is the work-tracking layer
of a fully markdown-native product management stack — one that spans from architecture decisions
and knowledge base through to the code itself.

---

## Users

### Humans
Single developers or very small teams building software products. Work is stored in a git
repository; tickets live in the same repository as the code they describe.

### Automata (first-class)
CI/CD pipelines, AI coding agents, LLMs, and testing infrastructure. These actors read and write
tickets directly via the file schema — they do not use the CLI and cannot be constrained by the
state machine. The schema is their contract; lint is their safety net.

---

## Capabilities

### Must-have

| Capability | Description |
|---|---|
| Ticket schema | YAML frontmatter defining status, type, tags, ID, timestamps. The schema is the authoritative contract for all users — human and machine. |
| CRUD CLI | `clinban new`, `clinban show`, `clinban list`, `clinban move` — create, view, query, and transition tickets from the terminal. |
| State machine (CLI) | Defined ticket statuses with valid transitions enforced by the CLI. External writers are not constrained — lint compensates. |
| Directory convention | Active tickets in the repo root directory (visible via `ls` or GitHub dir view). Closed tickets moved to `archive/` automatically. Tickets are identified by schema, not filename pattern. |
| Lint | `clinban lint [<id>]` — validates a single ticket or the entire repository against the schema. The primary integrity check for machine-written tickets. |

### Should-have

| Capability | Description |
|---|---|
| Search and filter | `clinban list` with `--status`, `--tag`, `--type` flags. |
| Inter-ticket linking | Wiki-style `[[ticket-id]]` references between tickets. |
| Cross-system linking | Linking convention for ADR entries and wiki pages in the same or related repositories. |
| Board view | Terminal kanban with columns mapped to statuses. Deferred until CRUD CLI is stable — revisit as next iteration. |

---

## Non-Goals (v1 and beyond unless explicitly revisited)

- No external services or network calls
- No authentication or multi-user access control
- No git integration (versioning handled independently by the user)
- No time tracking, estimation, burndown, or velocity metrics
- No notifications or reminders
- No web UI
- No multiple board configurations — one board per repository

---

## Acceptance Criteria

v1 is complete when:

1. `clinban new` creates a valid ticket file with correct YAML frontmatter and a unique ID
2. `clinban list` shows active tickets; `--status`, `--tag`, `--type` flags filter results
3. `clinban show <id>` renders a ticket readably in the terminal
4. `clinban move <id> <status>` transitions a ticket and enforces valid state transitions; invalid moves are rejected with a clear error message
5. Closing a ticket moves its file to `archive/` automatically
6. A machine (script or agent) can create, read, and update a valid ticket by writing YAML frontmatter directly — no CLI required; the schema is the contract
7. `clinban lint` detects and reports schema violations in a single ticket or across the whole repository
8. The ticket schema is documented well enough that an AI agent can consume it without human explanation

---

## Technology

- Language: Go (latest stable)
- Storage: filesystem only — no database
- Target platforms: Linux, macOS
- Repository assumption: the ticket directory is a git repository (managed independently)

---

## Ecosystem Context

Clinban is one layer of a markdown-native product management stack:

```
Knowledge Base (wiki, markdown)
Architecture Decisions (ADR registry, markdown)
Work Registry (Clinban tickets, markdown)
Code (git repository)
```

All layers share the same repository or a set of related repositories. Cross-linking between layers
is a first-class design concern.
