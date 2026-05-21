# Implementation Design
_Produced by: techlead (inline)_
_Date: 2026-05-21_
_Status: draft_
_Input: ticket 0006 — clinban push_

## Scope note

Two packages touched:
- `internal/fsm` — new `NextStatus` function defining the forward progression
- `cmd/clinban` — new `push.go` command file

No changes to store, ticket, or any other package.

---

## Module Structure

### internal/fsm — fsm.go

**Add:**
```go
// NextStatus returns the next forward status for the push command.
// Returns ("", false) when from is the terminal push status (done).
func NextStatus(from ticket.Status) (ticket.Status, bool) {
    switch from {
    case ticket.StatusBacklog:
        return ticket.StatusInProgress, true
    case ticket.StatusInProgress:
        return ticket.StatusDone, true
    case ticket.StatusBlocked:
        return ticket.StatusInProgress, true
    default:
        return "", false
    }
}
```

**Forward progression rationale:**
- `backlog` → `in-progress`: work starts
- `in-progress` → `done`: work completes (skip `blocked`; that is a lateral/exception state)
- `blocked` → `in-progress`: blocker resolved, work resumes
- `done` → none: terminal state for push; `done→backlog` is a reopen, not a push-forward

**Interface contract:**
- Accepts any `ticket.Status`; unknown values fall through to `default` returning `("", false)`
- Never panics; pure function

---

### cmd/clinban — push.go (new file)

```go
package main

import (
    "errors"
    "fmt"
    "os"
    "time"

    "github.com/spf13/cobra"

    "github.com/108adams/clinban/internal/fsm"
    "github.com/108adams/clinban/internal/store"
)

var pushCmd = &cobra.Command{
    Use:          "push <id>",
    Short:        "Advance a ticket to its next status",
    SilenceUsage: true,
    Args:         cobra.ExactArgs(1),
    RunE:         runPush,
}

func init() {
    rootCmd.AddCommand(pushCmd)
}

func runPush(_ *cobra.Command, args []string) error {
    id := args[0]

    path, _, err := st.FindByID(id)
    if err != nil {
        if errors.Is(err, store.ErrNotFound) {
            fmt.Fprintln(os.Stderr, "ticket not found")
            return ExitError{Code: 1, Err: fmt.Errorf("ticket not found")}
        }
        return fmt.Errorf("push: find ticket: %w", err)
    }

    t, err := st.ReadTicket(path)
    if err != nil {
        return fmt.Errorf("push: read ticket: %w", err)
    }

    next, ok := fsm.NextStatus(t.Status)
    if !ok {
        fmt.Fprintf(os.Stdout, "ticket %s is already at the final status (%s)\n", t.ID, t.Status)
        return nil
    }

    t.Status = next
    t.Updated = time.Now()
    if err := st.WriteTicket(t, path); err != nil {
        return fmt.Errorf("push: write ticket: %w", err)
    }

    fmt.Fprintf(os.Stdout, "ticket %s moved to %s\n", t.ID, next)
    return nil
}
```

**Interface contract:**
- `push <id>` advances the ticket one step in the forward direction
- Exit 0 in all non-error cases (including "already done" — per ticket spec)
- Exit 1 only on ticket-not-found or I/O error
- Output on stdout: `"ticket 0001 moved to in-progress\n"` or `"ticket 0001 is already at the final status (done)\n"`
- No archive handling: `push` never transitions `done→backlog`; archiving is a separate command

---

## Test Strategy

**`internal/fsm` unit tests (table-driven):**
```go
{"backlog → in-progress", StatusBacklog, StatusInProgress, true},
{"in-progress → done",    StatusInProgress, StatusDone, true},
{"blocked → in-progress", StatusBlocked, StatusInProgress, true},
{"done → none",           StatusDone, "", false},
```

**`cmd/clinban` subprocess tests:**
1. `TestPushFromBacklog`: ticket in `backlog` → exits 0, stdout contains "moved to in-progress", file has `status: in-progress`
2. `TestPushFromInProgress`: ticket in `in-progress` → exits 0, stdout contains "moved to done", file has `status: done`
3. `TestPushFromBlocked`: ticket in `blocked` → exits 0, stdout contains "moved to in-progress"
4. `TestPushFromDone`: ticket in `done` → exits 0, stdout contains "final status"
5. `TestPushTicketNotFound`: unknown id → exits 1, stderr contains "not found"

---

## Dependency Order

```
TASK-001 (internal/fsm NextStatus)
    └── TASK-002 (cmd/clinban/push.go)
```
