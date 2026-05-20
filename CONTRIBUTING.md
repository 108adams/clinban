# Contributing to Clinban

Clinban is a small project maintained by one person with LLM coding agents. Contributions are welcome. Keep changes small and focused.

## Setup

Requirements: Go 1.25 or later.

```bash
git clone https://gitlab.com/108adams/clinban.git
cd clinban
go build ./cmd/clinban
```

## Development Workflow

### Test

```bash
go test ./...
```

If the Go build cache is read-only (e.g. in a sandbox):

```bash
GOCACHE=/tmp/clinban-gocache go test ./...
```

### Vet and format

```bash
go vet ./...
gofmt -l .          # output must be empty
gofmt -w <file>     # fix a specific file
```

All three must pass before submitting.

### Module hygiene (CI guard)

After adding or removing imports, run:

```bash
go mod tidy && git diff --exit-code go.mod go.sum
```

The `git diff --exit-code` step fails if `go mod tidy` made any changes, which means `go.mod` or `go.sum` was out of sync. This command is run in CI to enforce that every commit keeps the module files tidy. Run it locally before pushing.

### TDD

Write a failing test before writing implementation. The cycle is:

1. Stub the function (compiles, no logic).
2. Write tests that call the stub — confirm they fail.
3. Implement until tests pass.

Do not write implementation before a failing test exists.

## Code Style

- Follow existing Go style and package boundaries (see `AGENTS.md`).
- Return `error` as the last return value; never panic in library code.
- Wrap errors with context: `fmt.Errorf("store: read %s: %w", id, err)`.
- Write no comments by default. Add one only when the why is non-obvious.
- No global mutable state outside of Cobra command wiring.

## Commit Hygiene

One commit per logical unit of work. A logical unit is one of:

- domain/model change
- store/filesystem change
- CLI command change
- tests
- documentation update

Always update the relevant `docs/` page and append to `docs/log.md` in the same commit as a behavior change.

Do not commit generated binaries (`clinban`, `clinban.test`).

## Documentation

The `docs/` directory is the maintained project wiki. The schema is in `docs/schema.md`.

When behavior changes:

1. Update code.
2. Update tests.
3. Update Go doc comments if exported behavior changed.
4. Update the relevant `docs/` page.
5. Append an entry to `docs/log.md`.

## Submitting Changes

Open a merge request on GitLab against `main`. Keep the MR scope to a single logical change. Describe what changed and why in the MR description, not in code comments.

## For LLM Agents

Read `AGENTS.md` before making any change. It defines package boundaries, validation commands, documentation rules, and commit strategy for automated contributors.
