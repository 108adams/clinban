---
title: Development
kind: workflow
scope: development
summary: Records build, test, documentation, and maintenance workflows for Clinban.
updated: 2026-05-20
links:
  - architecture
  - documentation
  - schema
---

# Development

## Build

```bash
go build ./...
```

## Test

```bash
go test ./...
```

When the default Go cache is unavailable or read-only, use a writable cache:

```bash
GOCACHE=/tmp/clinban-gocache go test ./...
```

## Vet

```bash
go vet ./...
```

## Documentation

Go package documentation is written as doc comments and package-level `doc.go` files.

```bash
go doc ./internal/ticket
go doc ./internal/store
```

Markdown project documentation lives under `docs/` and follows [Documentation Schema](schema.md).

## CI

GitHub Actions runs `go vet ./...` and `go test ./...` on every push and pull request to `main`. The workflow is defined in `.github/workflows/ci.yml`.

## Test Strategy

Critical test areas:

- Ticket parse/marshal semantics.
- Lint rule coverage.
- FSM transition matrix.
- Store filesystem behavior with temporary directories.
- CLI command smoke tests using scripted editors where needed.

`internal/editor` is verified through CLI integration tests rather than unit tests because it spawns an OS process.
