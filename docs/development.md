---
title: Development
kind: workflow
scope: development
summary: Records build, test, documentation, and maintenance workflows for Clinban.
updated: 2026-05-21
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

## End-to-End Testing

Clinban is a local CLI, so end-to-end tests should exercise the compiled
`clinban` binary as a black box. Prefer Go's standard `testing` package with
`os/exec`, temporary work directories, and real file assertions.

E2E tests should emulate a user by:

1. Creating a temporary project directory.
2. Running `clinban` commands as subprocesses.
3. Capturing stdout, stderr, and exit code.
4. Inspecting the resulting `.clinban`, `SCHEMA.md`, and ticket files.
5. Cleaning up through `t.TempDir`.

Use scripted editors for commands that invoke `$EDITOR`. Set `EDITOR` to a
small shell script that edits the temporary ticket file, then exits. Do not open
a real editor in automated tests.

Recommended first-class scenarios:

- `init -> new -> list -> show -> lint`
- `new -> push -> push -> archive`
- interactive `new` with a fake editor
- `edit` parse/lint failure leaves the original ticket unchanged
- `register` adopts an external Markdown ticket
- invalid tickets produce expected lint output
- `archive`, then `move <id> backlog` reopens from archive
- custom `tickets_dir`, `archive_dir`, and `default_type`

Use pseudo-terminal tooling only when behavior depends on terminal semantics.
Most Clinban flows should use ordinary stdin/stdout/stderr pipes. If a future
feature needs terminal control, use a small PTY-specific test layer instead of
making all E2E tests PTY-based.

If command coverage matters for subprocess tests, build a coverage-instrumented
binary with `go build -cover` and run it with `GOCOVERDIR` so integration
coverage is collected from the executed binary.

If the number of command scenarios grows large, consider adding
`github.com/rogpeppe/go-internal/testscript` for readable file-backed scenario
scripts. Keep the default harness simple until plain Go tests become hard to
scan.
