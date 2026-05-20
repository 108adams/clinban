# CLAUDE.md

Project guide is in **AGENTS.md** — read that first. This file adds Claude Code-specific workflow on top of it.

## Skills

Use skills before acting:

- `/dev` — orient on a task and start implementing (loads task context, follows TDD)
- `/techlead` — design sessions and task decomposition
- `/architect` — architectural decisions
- `/kb-query` — query project knowledge before answering questions
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

## Commit Hygiene

- One commit per logical unit of work (domain change → store → CLI → tests → docs).
- Do not commit generated binaries (`clinban`, `clinban.test`).
- Always update relevant `docs/` pages and append to `docs/log.md` in the same commit as the behavior change. Use `/librarian` for any non-trivial docs update.
