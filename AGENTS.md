# AGENTS.md

This file is the working guide for LLM agents contributing to Clinban.

Clinban is a Go CLI that manages kanban tickets as Markdown files with YAML frontmatter. The code and documentation are both part of the product: future changes should keep implementation, tests, Go docs, and the Markdown wiki aligned.

## Start Here

Read these first, in order:

1. `docs/index.md` - map of maintained project knowledge.
2. `docs/product.md` - product purpose and boundaries.
3. `docs/architecture.md` - package responsibilities and dependency boundaries.
4. `docs/cli.md` - command behavior.
5. `docs/ticket-schema.md` - ticket file contract.
6. `docs/validation.md` - parse, lint, and FSM rules.
7. `docs/storage.md` and `docs/security.md` - filesystem behavior and trust model.

The `docs/` wiki is the maintained source of truth. Do not resurrect `pipeline/` planning files; that directory was intentionally emptied after migration.

## Project Shape

Important packages:

- `cmd/clinban`: Cobra CLI commands and user-facing behavior.
- `internal/ticket`: ticket schema, status/type constants, parse/marshal.
- `internal/store`: filesystem storage, ID scanning, atomic writes, active/archive moves.
- `internal/lint`: schema validation for parsed tickets.
- `internal/fsm`: status transition rules.
- `internal/config`: `.clinban` loading and path resolution.
- `internal/editor`: `$EDITOR` invocation.
- `internal/slug`: title-to-filename slug generation.
- `internal/template`: embedded interactive ticket template.

Respect the package boundaries:

- `internal/ticket` must not import `internal/store`.
- `internal/lint` must not import `internal/store`.
- `internal/fsm` must not import `internal/store`.
- CLI code coordinates packages; internal packages own domain behavior.

## Development Rules

- Follow existing Go style and package boundaries.
- Keep behavior filesystem-native; do not introduce databases, services, network calls, auth, or web UI unless explicitly requested.
- Keep ticket files compatible with humans, scripts, CI, and LLM agents.
- Use structured parsing/marshaling for ticket frontmatter; avoid ad hoc string edits for schema data.
- Preserve the parse-vs-lint distinction: parse errors happen before lint can run.
- For file writes, preserve the same-directory temp-file plus rename pattern.
- Do not overwrite archive/active destination files silently.
- Treat `$EDITOR` as user-controlled environment state; do not try to sandbox it inside Clinban.

## Documentation Rules

There are two documentation layers:

- Go package docs: package-level `doc.go` files and exported symbol comments.
- Markdown wiki: `docs/` pages with lightweight YAML frontmatter.

When behavior changes:

1. Update code.
2. Update tests.
3. Update Go doc comments if exported behavior changed.
4. Update the relevant `docs/` page.
5. Update `docs/index.md` if pages were added, removed, renamed, or repurposed.
6. Append a short entry to `docs/log.md`.

Use the `librarian` Codex skill for documentation wiki work when available. The wiki schema is in `docs/schema.md`.

## Validation Commands

In this workspace, use a writable Go build cache by default. The normal cache
may be read-only in sandboxed sessions.

```bash
export GOCACHE=/tmp/go-trello-gocache
```

Run before handing work back:

```bash
gofmt -w <changed-go-files>
GOCACHE=/tmp/go-trello-gocache go test ./...
GOCACHE=/tmp/go-trello-gocache go vet ./...
```

Use the same cache setting for other Go commands:

```bash
GOCACHE=/tmp/go-trello-gocache go run ./cmd/clinban --help
GOCACHE=/tmp/go-trello-gocache go doc ./internal/ticket
```

For docs-only changes, still consider running `go test ./...` if code comments or generated documentation paths changed.

## Review Expectations

When reviewing or implementing, prioritize:

- Data integrity of ticket files.
- Safe filesystem behavior.
- Clear CLI output and exit codes.
- Compatibility with the documented ticket schema.
- Tests that exercise command behavior through realistic filesystem fixtures.
- Documentation alignment with current behavior.

## Working Tree Hygiene

The worktree may contain user changes. Do not revert files you did not change unless explicitly asked.

Generated binaries such as `clinban` and `clinban.test` should not be treated as source files. Do not add them to commits unless the user explicitly requests release artifacts.

## Commit-Sized Change Strategy

Prefer small, coherent changes:

1. Domain/model changes.
2. Store/filesystem changes.
3. CLI command behavior.
4. Tests.
5. Documentation/wiki updates.

Keep refactors scoped to the task. If you discover unrelated issues, report them or create a follow-up plan instead of mixing them into the change.
