# Developer Tasks
_Produced by: techlead (inline)_
_Date: 2026-05-21_
_Status: draft_
_Input: pipeline/03_design.md_

## Task List

### TASK-001: Add NextStatus to internal/fsm

- **Description:** Add a `NextStatus(from ticket.Status) (ticket.Status, bool)` function to `internal/fsm/fsm.go` that returns the next forward status for the push command.
- **Module(s):** `internal/fsm/fsm.go`, `internal/fsm/fsm_test.go`
- **Done criteria:**
  - [ ] `NextStatus` function exists in `internal/fsm`
  - [ ] `NextStatus(StatusBacklog)` returns `(StatusInProgress, true)`
  - [ ] `NextStatus(StatusInProgress)` returns `(StatusDone, true)`
  - [ ] `NextStatus(StatusBlocked)` returns `(StatusInProgress, true)`
  - [ ] `NextStatus(StatusDone)` returns `("", false)`
  - [ ] Table-driven test covers all four cases
  - [ ] `go test ./internal/fsm/...` passes
  - [ ] `go vet ./internal/fsm/...` clean
- **Depends on:** none

---

### TASK-002: Add cmd/clinban/push.go

- **Description:** Create a new `push` command that advances a ticket one step in the forward direction using `fsm.NextStatus`.
- **Module(s):** `cmd/clinban/push.go` (new), `cmd/clinban/push_test.go` (new)
- **Done criteria:**
  - [ ] `push.go` defines `pushCmd` with `Use: "push <id>"`, `Args: cobra.ExactArgs(1)`, `SilenceUsage: true`
  - [ ] `runPush` finds ticket by ID (exit 1 + stderr "ticket not found" on missing)
  - [ ] `runPush` calls `fsm.NextStatus`; when `ok == false`, prints `"ticket <id> is already at the final status (<status>)\n"` to stdout and returns nil (exit 0)
  - [ ] When `ok == true`: updates `t.Status` and `t.Updated`, writes ticket, prints `"ticket <id> moved to <next>\n"` to stdout
  - [ ] `TestPushFromBacklog`: backlog ticket → exits 0, stdout "moved to in-progress", file has `status: in-progress`
  - [ ] `TestPushFromInProgress`: in-progress ticket → exits 0, stdout "moved to done"
  - [ ] `TestPushFromBlocked`: blocked ticket → exits 0, stdout "moved to in-progress"
  - [ ] `TestPushFromDone`: done ticket → exits 0, stdout "final status"
  - [ ] `TestPushTicketNotFound`: unknown id → exits 1, stderr "not found"
  - [ ] `go test ./cmd/clinban/...` passes
  - [ ] `go vet ./cmd/clinban/...` clean
- **Depends on:** TASK-001

---

## Dependency Order

```
TASK-001 (internal/fsm NextStatus)
    └── TASK-002 (cmd/clinban/push.go)
```
