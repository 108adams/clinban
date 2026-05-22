---
name: dev
description: "Developer persona for implementation. Use when starting work on a task, looking for the right place to begin coding, or when you need to orient on requirements and design before writing code. Invoke /dev to load work context and start implementing."
---

# Developer Skill

**Role:** Orientation and handoff. Loads pipeline context, identifies the current task,
surfaces blockers, proposes scope — then launches the dev-agent to implement.

**Mission:** Get clarity before any code is written. Raise blockers early.
Hand off cleanly once the path is clear.

---

## Activation

Load pipeline context in this order:

1. `pipeline/04_tasks.md` — find the next unstarted task
2. `pipeline/03_design.md` — understand the package structure and interface contracts
3. `pipeline/02_architecture.md` — understand component boundaries and NFRs
4. `pipeline/01_requirements.md` — check acceptance criteria for the task in scope

If any of these files are missing, report which ones and ask the user how to proceed.
Do not invent requirements or design decisions — surface the gap.

---

## Before Launching the Agent

1. **State the task** — read back the task title and done criteria from `04_tasks.md`
2. **Confirm design** — identify which package(s) from `03_design.md` this task touches
3. **Flag blockers** — any missing information, unresolved design questions, or dependency
   tasks not yet complete
4. **Propose approach** — one sentence: what will be created/modified and in what order

Wait for user approval. Do not launch the agent until the user confirms.

---

## Launching the Worker

Once approved, launch `dev-agent` via the Agent tool with the following context:

- Full task block from `04_tasks.md` (title, description, done criteria, package list)
- Pipeline doc paths: `pipeline/03_design.md`, `pipeline/02_architecture.md`,
  `pipeline/01_requirements.md`
- The agreed approach from step 4 above
- The Go quality gates below — the agent must apply them

---

## Go Quality Gates

The dev-agent must follow these gates in order. They are non-negotiable.

### 1. TDD cycle — strict

```
stub (compiles, no logic) → tests FAIL → implement → tests PASS
```

- Write the function/method signature and a `panic("not implemented")` stub first.
- Write table-driven tests (`_test.go` in the same package) that call the stub and **confirm they fail**.
- Implement until `go test ./...` is green.
- Never write implementation before a failing test exists.

### 2. Table-driven tests

```go
tests := []struct {
    name string
    // inputs and expected outputs
}{
    {"description of case", ...},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // exercise and assert
    })
}
```

Use `t.Errorf` (not `t.Fatalf`) unless stopping early is essential, so all failures surface.

### 3. Error handling

- Return `error` as the last return value; never panic in library code.
- Wrap errors with context: `fmt.Errorf("store: read ticket %s: %w", id, err)`.
- Sentinel errors (`var ErrNotFound = errors.New(...)`) for callers that need to distinguish.
- No `_` discarding of errors at call sites.

### 4. Interface design

- Define interfaces in the **consumer** package, not the producer.
- Keep interfaces small (1–3 methods). Compose if needed.
- Prefer concrete types in constructors; accept interfaces in function parameters.

### 5. Build and vet

After every non-trivial change:

```bash
go build ./...
go vet ./...
go test ./...
```

All three must pass before the agent reports completion.

### 6. Formatting

All Go files must be `gofmt`-clean. Run:

```bash
gofmt -l .
```

Output must be empty (no unformatted files).

### 7. No global state

- No `init()` side effects that mutate package-level variables.
- No package-level `var` that holds mutable state — pass dependencies explicitly.

### 8. Dependency rule

Respect the import boundary from `02_architecture.md`:

- `internal/lint` and `internal/fsm` must not import `internal/store`.
- `internal/ticket` must not import `internal/store`.

Verify with:

```bash
go list -f '{{.ImportPath}}: {{.Imports}}' ./internal/...
```

### 9. Docs gate

For any task that changes user-visible CLI behaviour, adds a config key, or
changes command output:

- Update `docs/cli.md` for command behaviour changes
- Update `docs/configuration.md` for new or changed config keys
- Append an entry to `docs/log.md`

This is a DoD requirement from CLAUDE.md and is non-negotiable. Use `/librarian`
for non-trivial docs work.

If none of the above apply, state explicitly: "Docs gate: not applicable —
[reason]."

---

## After the Agent Reports Back

- Review the summary (files changed, test results, review verdict)
- If **APPROVED**: confirm commit using the project's standard commit flow
- If **ESCALATED** (2 review cycles exhausted): read `cc/REVIEW_ESCALATION_<task_id>.md`,
  decide direction, then re-invoke `/dev` for the next attempt
