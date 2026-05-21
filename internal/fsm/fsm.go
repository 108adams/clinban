package fsm

import (
	"fmt"
	"strings"

	"github.com/108adams/clinban/internal/ticket"
)

// transitions maps each Status to the set of statuses it may legally move to.
var transitions = map[ticket.Status][]ticket.Status{
	ticket.StatusBacklog:    {ticket.StatusInProgress, ticket.StatusBlocked},
	ticket.StatusInProgress: {ticket.StatusBlocked, ticket.StatusDone},
	ticket.StatusBlocked:    {ticket.StatusInProgress},
	ticket.StatusDone:       {ticket.StatusBacklog},
}

// ValidateTransition reports whether a ticket may move from one status to
// another.
//
// A nil error means the transition is explicitly allowed by Clinban's workflow.
// A non-nil error means the transition is forbidden; the error message includes
// the valid next statuses for from. Self-transitions are not valid here. The CLI
// move command handles no-op self-transitions before calling ValidateTransition.
func ValidateTransition(from, to ticket.Status) error {
	nexts, ok := transitions[from]
	if ok {
		for _, s := range nexts {
			if s == to {
				return nil
			}
		}
	}

	// Build the error message with valid next statuses.
	validList := joinStatuses(nexts)
	return fmt.Errorf(
		"cannot transition from %q to %q; valid transitions: %s",
		from, to, validList,
	)
}

// NextStatus returns the next forward status for the push command.
// Returns ("", false) when from is the terminal push status (done) or unknown.
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

// joinStatuses returns a comma-separated string of status values.
func joinStatuses(statuses []ticket.Status) string {
	parts := make([]string, len(statuses))
	for i, s := range statuses {
		parts[i] = string(s)
	}
	return strings.Join(parts, ", ")
}
