package board_test

import (
	"sort"
	"testing"

	"github.com/108adams/clinban/internal/board"
	"github.com/108adams/clinban/internal/ticket"
)

// helper builds a minimal *ticket.Ticket with just the fields Less uses.
func makeTicket(id string, status ticket.Status) *ticket.Ticket {
	return &ticket.Ticket{ID: id, Status: status}
}

// TestStatusOrder verifies the StatusOrder map has exactly the four expected
// ranks in the canonical order.
func TestStatusOrder(t *testing.T) {
	t.Parallel()

	want := map[ticket.Status]int{
		ticket.StatusInProgress: 0,
		ticket.StatusBlocked:    1,
		ticket.StatusBacklog:    2,
		ticket.StatusDone:       3,
	}

	for status, wantRank := range want {
		if got := board.StatusOrder[status]; got != wantRank {
			t.Errorf("StatusOrder[%q] = %d, want %d", status, got, wantRank)
		}
	}

	if len(board.StatusOrder) != len(want) {
		t.Errorf("StatusOrder has %d entries, want %d", len(board.StatusOrder), len(want))
	}
}

// TestLessGroupOrder verifies that Less produces the canonical group ordering:
// in-progress < blocked < backlog < done.
func TestLessGroupOrder(t *testing.T) {
	t.Parallel()

	type pair struct {
		a, b   *ticket.Ticket
		wantLT bool // a Less b should be true
	}

	tests := []struct {
		name   string
		a, b   *ticket.Ticket
		wantLT bool
	}{
		{
			name:   "in-progress before blocked",
			a:      makeTicket("0001", ticket.StatusInProgress),
			b:      makeTicket("0002", ticket.StatusBlocked),
			wantLT: true,
		},
		{
			name:   "in-progress before backlog",
			a:      makeTicket("0001", ticket.StatusInProgress),
			b:      makeTicket("0002", ticket.StatusBacklog),
			wantLT: true,
		},
		{
			name:   "in-progress before done",
			a:      makeTicket("0001", ticket.StatusInProgress),
			b:      makeTicket("0002", ticket.StatusDone),
			wantLT: true,
		},
		{
			name:   "blocked before backlog",
			a:      makeTicket("0001", ticket.StatusBlocked),
			b:      makeTicket("0002", ticket.StatusBacklog),
			wantLT: true,
		},
		{
			name:   "blocked before done",
			a:      makeTicket("0001", ticket.StatusBlocked),
			b:      makeTicket("0002", ticket.StatusDone),
			wantLT: true,
		},
		{
			name:   "backlog before done",
			a:      makeTicket("0001", ticket.StatusBacklog),
			b:      makeTicket("0002", ticket.StatusDone),
			wantLT: true,
		},
		// Reversed pairs — a should NOT be less than b.
		{
			name:   "blocked NOT before in-progress",
			a:      makeTicket("0002", ticket.StatusBlocked),
			b:      makeTicket("0001", ticket.StatusInProgress),
			wantLT: false,
		},
		{
			name:   "done NOT before backlog",
			a:      makeTicket("0002", ticket.StatusDone),
			b:      makeTicket("0001", ticket.StatusBacklog),
			wantLT: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := board.Less(tc.a, tc.b)
			if got != tc.wantLT {
				t.Errorf("Less(%q/%s, %q/%s) = %v, want %v",
					tc.a.ID, tc.a.Status, tc.b.ID, tc.b.Status, got, tc.wantLT)
			}
		})
	}
}

// TestLessIDTiebreak verifies that within the same status group Less sorts
// by ascending numeric ID.
func TestLessIDTiebreak(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		a, b   *ticket.Ticket
		wantLT bool
	}{
		{
			name:   "same status lower ID sorts first",
			a:      makeTicket("0001", ticket.StatusBacklog),
			b:      makeTicket("0002", ticket.StatusBacklog),
			wantLT: true,
		},
		{
			name:   "same status higher ID does not sort first",
			a:      makeTicket("0002", ticket.StatusBacklog),
			b:      makeTicket("0001", ticket.StatusBacklog),
			wantLT: false,
		},
		{
			name:   "same status same ID — not less than self",
			a:      makeTicket("0005", ticket.StatusInProgress),
			b:      makeTicket("0005", ticket.StatusInProgress),
			wantLT: false,
		},
		{
			name:   "in-progress group ID tiebreak",
			a:      makeTicket("0010", ticket.StatusInProgress),
			b:      makeTicket("0020", ticket.StatusInProgress),
			wantLT: true,
		},
		{
			name:   "done group ID tiebreak",
			a:      makeTicket("0003", ticket.StatusDone),
			b:      makeTicket("0001", ticket.StatusDone),
			wantLT: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := board.Less(tc.a, tc.b)
			if got != tc.wantLT {
				t.Errorf("Less(%q/%s, %q/%s) = %v, want %v",
					tc.a.ID, tc.a.Status, tc.b.ID, tc.b.Status, got, tc.wantLT)
			}
		})
	}
}

// TestSliceStableIntegration verifies that using board.Less with sort.SliceStable
// produces the expected full ordering: in-progress → blocked → backlog → done,
// ascending ID within each group.
func TestSliceStableIntegration(t *testing.T) {
	t.Parallel()

	tickets := []*ticket.Ticket{
		makeTicket("0004", ticket.StatusDone),
		makeTicket("0001", ticket.StatusBacklog),
		makeTicket("0003", ticket.StatusBlocked),
		makeTicket("0002", ticket.StatusInProgress),
		makeTicket("0006", ticket.StatusBacklog),
		makeTicket("0005", ticket.StatusInProgress),
	}

	sort.SliceStable(tickets, func(i, j int) bool {
		return board.Less(tickets[i], tickets[j])
	})

	want := []struct {
		id     string
		status ticket.Status
	}{
		{"0002", ticket.StatusInProgress},
		{"0005", ticket.StatusInProgress},
		{"0003", ticket.StatusBlocked},
		{"0001", ticket.StatusBacklog},
		{"0006", ticket.StatusBacklog},
		{"0004", ticket.StatusDone},
	}

	if len(tickets) != len(want) {
		t.Fatalf("got %d tickets, want %d", len(tickets), len(want))
	}

	for i, w := range want {
		got := tickets[i]
		if got.ID != w.id || got.Status != w.status {
			t.Errorf("position %d: got {%s, %s}, want {%s, %s}",
				i, got.ID, got.Status, w.id, w.status)
		}
	}
}

// TestLessUnknownStatusSortsLast verifies that an unknown status value (not in
// StatusOrder) maps to zero rank (same as in-progress) — it falls back to 0 via
// the map zero value, so it sorts as if it were in-progress.  This is a
// correctness guard, not an API guarantee.
func TestLessUnknownStatus(t *testing.T) {
	t.Parallel()

	unknown := makeTicket("0001", ticket.Status("unknown"))
	inProgress := makeTicket("0002", ticket.StatusInProgress)

	// Both get rank 0 from the map; tiebreak is by ID: 0001 < 0002 so Less == true.
	if !board.Less(unknown, inProgress) {
		t.Errorf("expected unknown status (rank 0) with lower ID to sort before in-progress with higher ID")
	}
}
