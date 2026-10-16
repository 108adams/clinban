---
name: tickets
description: "Clinban ticket lifecycle interface for LLM agents. Use when creating a new ticket, finding the ticket for the current task, advancing status, resolving blockers, closing after a commit, or archiving done work. Invoke /tickets for any ticket operation."
---

# Tickets Skill

**Role:** Ticket lifecycle operator for Clinban.

**Mission:** Manage tickets correctly — discover, create, advance, block, close, and archive — without
ever corrupting frontmatter or bypassing FSM transitions.

---

## Activation

Read first:

1. `.clinban` (in the project root) — find `tickets_dir` and `archive_dir` before touching any file
2. `SCHEMA.md` **in the project root** (created by `clinban init`, *not* a sibling of this skill) —
   the authoritative field rules, FSM transitions, and agent operation steps. If it is present, read
   it; it is your source of truth and this skill adds LLM reasoning guidance on top of it.

If `SCHEMA.md` is absent (e.g. this skill is installed globally, outside an initialized project),
do not block: this skill plus the `clinban` binary are sufficient. The binary is the authoritative
validator — run `clinban <command> --help` and `clinban lint` when in doubt rather than guessing.

---

## When to Create a Ticket

Create a ticket when:

- Starting a discrete, self-contained piece of work with identifiable done criteria
- Capturing a bug that has been reproduced and can be described precisely
- Documenting a spike (research task) before beginning it

Do NOT create a ticket for:

- Sub-steps within an existing ticket's body — use a checklist there instead
- Work already described by an open ticket — find and use it
- Vague intentions without a clear title and type

**Before creating:** run `clinban list` to scan for an existing match. Duplicates waste triage effort.

---

## Operations

### Discover

```bash
clinban list                          # all open tickets
clinban list --status backlog         # not yet started
clinban list --status in-progress     # actively worked
clinban list --tag <tag>              # filtered by tag
clinban show <id>                     # full detail including body
```

### Create

```bash
clinban new --no-interactive --title "<title>" --type <type> [--tags tag1,tag2] [--body "<text>"]
```

Valid types: `bug`, `task`, `feature`, `spike`. After creation, Clinban prints the filename —
record the ID. To add acceptance criteria or context, edit the body below the closing `---`;
never touch frontmatter fields.

### Advance Status

Check the FSM before every move:

```
backlog → in-progress → done → backlog
              ↓             ↑
           blocked ─────────┘
              ↓
           in-progress
```

```bash
clinban move <id> <new-status>   # explicit transition
clinban push <id>                # auto-advance one step
```

If `clinban move` exits non-zero, the transition is invalid. Read the error; do not edit `status`
directly in the file.

### Block and Unblock

When work stalls on an external dependency:

1. `clinban move <id> blocked`
2. Edit the ticket body — append a **Blocker** section describing what is needed and from whom
3. When resolved: `clinban move <id> in-progress` and remove or strike through the blocker note

### Close After a Commit

The canonical sequence after completing work:

1. Confirm the commit succeeded (`git log -1`)
2. `clinban move <id> done`
3. `clinban archive <id>`

Do not archive until the commit is on the branch. Do not skip `move` — archiving a non-done ticket
is not permitted.

### Validate

```bash
clinban lint          # all tickets
clinban lint <id>     # single ticket
```

Run lint after any direct file edit to confirm frontmatter is still valid.

If lint reports duplicate filename IDs after a git sync, run:

```bash
clinban resolve
```

It keeps the oldest conflicting ticket at the original ID and renames younger
conflicting tickets to the next available IDs.

---

## Rules

- `created` and `updated` are owned by Clinban. Never edit them directly.
- The ticket ID is derived from the filename's four-digit prefix. Never rename a
  ticket file or add an `id:` line to frontmatter.
- `status` must only change via `clinban move` or `clinban push`.
- `title`, `type`, `tags` are author-owned and may be edited freely.
- Body below `---` is freeform — edit freely, but do not bleed content into frontmatter.
- One ticket per unit of work. If a ticket covers two separable outcomes, split it.
- Adopt externally written ticket files with `clinban register <path>`, not direct writes.
- Use `clinban resolve` for duplicate filename IDs; do not manually rename
  conflicting ticket files.

---

## Relationship to `/dev`

`/dev` drives pipeline tasks (`pipeline/04_tasks.md`). `/tickets` drives the board (`tickets/`).

- Use `/tickets` to track *what* is being built, at product level
- Use `/dev` to track *how* it is built, at implementation level
- After `/dev` reports completion and the commit is confirmed, invoke `/tickets` to close and archive

---

## Handoff

After any lifecycle operation, state:

> "Ticket `<id>` is now `<status>`. [Archived / Still open.] Run `clinban list` to confirm board state."

If you discover lint errors, stale blocked tickets, or tickets with no body, flag them and offer
to triage.
