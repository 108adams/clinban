# Clinban

Clinban is a terminal kanban board stored as plain Markdown files.

Each ticket lives beside your code, with YAML frontmatter for the fields tools
need and Markdown for the human notes. There is no server, database, web UI, or
account to manage.

It is built for small software projects where humans, scripts, CI, and LLM
agents need to read the same work queue.

## Why Use It

- Tickets are normal files you can review, diff, edit, and commit.
- The schema is stable enough for automation.
- The CLI handles IDs, timestamps, status moves, archiving, and linting.
- Your project keeps its own work history without depending on an issue tracker.

## Install

You need Go 1.25 or later.

```bash
go install github.com/108adams/clinban/cmd/clinban@latest
```

Make sure your Go binary directory is on `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Check the install:

```bash
clinban --help
```

## Start A Board

Run this in the repository you want to track:

```bash
clinban init
```

This creates:

- `tickets/`
- `tickets/archive/`
- `.clinban`
- `SCHEMA.md`

## Create A Ticket

Interactive:

```bash
clinban new "Investigate the memory leak in the worker pool"
```

Non-interactive:

```bash
clinban new --no-interactive \
  --title "Fix login timeout" \
  --type bug \
  --tags auth,backend \
  --body "Users are signed out too early."
```

Useful next commands:

```bash
clinban list
clinban show 1
clinban push 1
clinban lint
```

## Ticket Files

A ticket is just Markdown:

```yaml
---
id: "0042"
status: "in-progress"
type: "bug"
title: "Fix login timeout"
tags: ["auth", "backend"]
created: "2026-05-18T14:30:00Z"
updated: "2026-05-18T15:00:00Z"
---

Notes go here.
```

## Documentation

- [CLI Reference](docs/cli.md)
- [Ticket Schema](docs/ticket-schema.md)
- [Configuration](docs/configuration.md)
- [Architecture](docs/architecture.md)
- [Documentation Index](docs/index.md)

## Development

```bash
GOCACHE=/tmp/clinban-gocache go test ./...
GOCACHE=/tmp/clinban-gocache go vet ./...
```

Clinban was built by Adam Kucharczyk with LLM coding agents. The project keeps
that collaboration model in mind: clear files, clear schema, clear docs.
