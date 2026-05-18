package fsm

import (
	"fmt"
	"strings"

	"clinban/internal/ticket"
)

// transitions maps each Status to the set of statuses it may legally move to.
var transitions = map[ticket.Status][]ticket.Status{
	ticket.StatusBacklog:    {ticket.StatusInProgress, ticket.StatusBlocked},
	ticket.StatusInProgress: {ticket.StatusBlocked, ticket.StatusDone},
	ticket.StatusBlocked:    {ticket.StatusInProgress},
	ticket.StatusDone:       {ticket.StatusBacklog},
}

// ValidateTransition returns nil if the transition from→to is in the valid
// transitions table. For any forbidden transition it returns a descriptive
// error that lists the valid next statuses for from.
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

// joinStatuses returns a comma-separated string of status values.
func joinStatuses(statuses []ticket.Status) string {
	parts := make([]string, len(statuses))
	for i, s := range statuses {
		parts[i] = string(s)
	}
	return strings.Join(parts, ", ")
}
