package fsm_test

import (
	"strings"
	"testing"

	"github.com/108adams/clinban/internal/fsm"
	"github.com/108adams/clinban/internal/ticket"
)

// validTransitionCases lists every transition that must return nil.
// Design reference: pipeline/03_design.md — internal/fsm valid transitions table.
var validTransitionCases = []struct {
	name string
	from ticket.Status
	to   ticket.Status
}{
	{name: "backlog to in-progress", from: ticket.StatusBacklog, to: ticket.StatusInProgress},
	{name: "backlog to blocked", from: ticket.StatusBacklog, to: ticket.StatusBlocked},
	{name: "in-progress to blocked", from: ticket.StatusInProgress, to: ticket.StatusBlocked},
	{name: "in-progress to done", from: ticket.StatusInProgress, to: ticket.StatusDone},
	{name: "blocked to in-progress", from: ticket.StatusBlocked, to: ticket.StatusInProgress},
	{name: "done to backlog", from: ticket.StatusDone, to: ticket.StatusBacklog},
}

// invalidTransitionCases lists every transition that must return a non-nil error.
// Self-transitions and all cross-transitions not in the valid table are included.
var invalidTransitionCases = []struct {
	name           string
	from           ticket.Status
	to             ticket.Status
	wantValidInMsg []string // substrings the error must include (valid next statuses)
}{
	// Self-transitions (4 cases)
	{
		name:           "backlog to backlog (self)",
		from:           ticket.StatusBacklog,
		to:             ticket.StatusBacklog,
		wantValidInMsg: []string{"in-progress", "blocked"},
	},
	{
		name:           "in-progress to in-progress (self)",
		from:           ticket.StatusInProgress,
		to:             ticket.StatusInProgress,
		wantValidInMsg: []string{"blocked", "done"},
	},
	{
		name:           "blocked to blocked (self)",
		from:           ticket.StatusBlocked,
		to:             ticket.StatusBlocked,
		wantValidInMsg: []string{"in-progress"},
	},
	{
		name:           "done to done (self)",
		from:           ticket.StatusDone,
		to:             ticket.StatusDone,
		wantValidInMsg: []string{"backlog"},
	},
	// Cross-invalid transitions (6 cases)
	{
		name:           "backlog to done",
		from:           ticket.StatusBacklog,
		to:             ticket.StatusDone,
		wantValidInMsg: []string{"in-progress", "blocked"},
	},
	{
		name:           "in-progress to backlog",
		from:           ticket.StatusInProgress,
		to:             ticket.StatusBacklog,
		wantValidInMsg: []string{"blocked", "done"},
	},
	{
		name:           "blocked to backlog",
		from:           ticket.StatusBlocked,
		to:             ticket.StatusBacklog,
		wantValidInMsg: []string{"in-progress"},
	},
	{
		name:           "blocked to done",
		from:           ticket.StatusBlocked,
		to:             ticket.StatusDone,
		wantValidInMsg: []string{"in-progress"},
	},
	{
		name:           "done to in-progress",
		from:           ticket.StatusDone,
		to:             ticket.StatusInProgress,
		wantValidInMsg: []string{"backlog"},
	},
	{
		name:           "done to blocked",
		from:           ticket.StatusDone,
		to:             ticket.StatusBlocked,
		wantValidInMsg: []string{"backlog"},
	},
}

func TestValidateTransition_ValidCases(t *testing.T) {
	t.Parallel()

	for _, tc := range validTransitionCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := fsm.ValidateTransition(tc.from, tc.to)
			if err != nil {
				t.Errorf("ValidateTransition(%q, %q) = %v, want nil", tc.from, tc.to, err)
			}
		})
	}
}

func TestValidateTransition_InvalidCases(t *testing.T) {
	t.Parallel()

	for _, tc := range invalidTransitionCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := fsm.ValidateTransition(tc.from, tc.to)
			if err == nil {
				t.Fatalf("ValidateTransition(%q, %q) = nil, want non-nil error", tc.from, tc.to)
			}

			msg := err.Error()

			// The error must mention the 'from' and 'to' statuses.
			if !strings.Contains(msg, string(tc.from)) {
				t.Errorf("error message %q does not contain 'from' status %q", msg, tc.from)
			}
			if !strings.Contains(msg, string(tc.to)) {
				t.Errorf("error message %q does not contain 'to' status %q", msg, tc.to)
			}

			// The error must mention each of the valid next statuses.
			for _, want := range tc.wantValidInMsg {
				if !strings.Contains(msg, want) {
					t.Errorf("error message %q does not contain expected valid status %q", msg, want)
				}
			}
		})
	}
}

// TestValidateTransition_ErrorMessageFormat verifies the canonical error format
// documented in the task spec:
//
//	cannot transition from "blocked" to "done"; valid transitions: in-progress
func TestValidateTransition_ErrorMessageFormat(t *testing.T) {
	t.Parallel()

	err := fsm.ValidateTransition(ticket.StatusBlocked, ticket.StatusDone)
	if err == nil {
		t.Fatal("expected non-nil error for blocked→done")
	}

	got := err.Error()
	const wantFmt = `cannot transition from "blocked" to "done"; valid transitions: in-progress`
	if got != wantFmt {
		t.Errorf("error message:\n  got:  %q\n  want: %q", got, wantFmt)
	}
}

// TestValidateTransition_ErrorMessageFormat_MultipleNextStatuses verifies that
// when a status has more than one valid next status, all are listed.
func TestValidateTransition_ErrorMessageFormat_MultipleNextStatuses(t *testing.T) {
	t.Parallel()

	// backlog → done is invalid; valid transitions from backlog are in-progress, blocked
	err := fsm.ValidateTransition(ticket.StatusBacklog, ticket.StatusDone)
	if err == nil {
		t.Fatal("expected non-nil error for backlog→done")
	}

	msg := err.Error()
	if !strings.Contains(msg, "in-progress") {
		t.Errorf("error message %q does not contain 'in-progress'", msg)
	}
	if !strings.Contains(msg, "blocked") {
		t.Errorf("error message %q does not contain 'blocked'", msg)
	}
}

// TestValidateTransition_CountValid asserts exactly 6 valid transitions exist.
func TestValidateTransition_CountValid(t *testing.T) {
	t.Parallel()

	const wantCount = 6
	if len(validTransitionCases) != wantCount {
		t.Errorf("validTransitionCases has %d entries, want %d", len(validTransitionCases), wantCount)
	}
}

// TestValidateTransition_CountInvalid asserts exactly 10 invalid transitions are tested.
func TestValidateTransition_CountInvalid(t *testing.T) {
	t.Parallel()

	const wantCount = 10
	if len(invalidTransitionCases) != wantCount {
		t.Errorf("invalidTransitionCases has %d entries, want %d", len(invalidTransitionCases), wantCount)
	}
}

// TestNextStatus covers all four documented cases for the push forward-progression.
func TestNextStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		from     ticket.Status
		wantNext ticket.Status
		wantOK   bool
	}{
		{name: "backlog to in-progress", from: ticket.StatusBacklog, wantNext: ticket.StatusInProgress, wantOK: true},
		{name: "in-progress to done", from: ticket.StatusInProgress, wantNext: ticket.StatusDone, wantOK: true},
		{name: "blocked to in-progress", from: ticket.StatusBlocked, wantNext: ticket.StatusInProgress, wantOK: true},
		{name: "done is terminal", from: ticket.StatusDone, wantNext: "", wantOK: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotNext, gotOK := fsm.NextStatus(tc.from)

			if gotOK != tc.wantOK {
				t.Errorf("NextStatus(%q): ok = %v, want %v", tc.from, gotOK, tc.wantOK)
			}
			if gotNext != tc.wantNext {
				t.Errorf("NextStatus(%q): next = %q, want %q", tc.from, gotNext, tc.wantNext)
			}
		})
	}
}
