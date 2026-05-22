# CLAUDE.md

Project guide is in **AGENTS.md** — MUST read that first. This file adds Claude Code-specific workflow on top of it.

## Skills

Use skills before acting:

- `/dev` — orient on a task and start implementing (loads task context, follows TDD)
- `/techlead` — design sessions and task decomposition
- `/architect` — architectural decisions
- `/simplify` — post-implementation review pass
- `/librarian` — docs wiki work (create, ingest, update, lint `docs/` pages)

The `superpowers:brainstorming` skill must run before any feature work. `/dev` invokes it automatically.

## Dev Workflow

1. Read the relevant `docs/` pages before touching code (see AGENTS.md §Start Here).
2. Use `/dev` to pick up a task from `pipeline/04_tasks.md` if it exists.
3. Follow AGENTS.md §Validation Commands — always set `GOCACHE=/tmp/clinban-gocache`.
4. Commit after each completed ticket/task.

## Go Commands

```bash
export GOCACHE=/tmp/clinban-gocache
go test ./...
go vet ./...
gofmt -w <changed-files>
```

## Codebase Patterns

- `//go:embed` paths must be within or below the package directory. Assets for `cmd/clinban` go in `cmd/clinban/` or a subdir — cannot embed from `.claude/` or repo root.
- `cmd/clinban/init_test.go` uses binary integration tests (`buildBinary(t)` + `exec.Command`), not unit tests. New init tests must follow this pattern.
- Use `os.MkdirAll` when creating nested dirs (e.g. `.claude/skills/tickets/`); `os.Mkdir` for single-level only.

## Commit Hygiene

- One commit per logical unit of work (domain change → store → CLI → tests → docs).
- Do not commit generated binaries (`clinban`, `clinban.test`).
- Always update relevant `docs/` pages and append to `docs/log.md` in the same commit as the behavior change. Use `/librarian` for any non-trivial docs update.
