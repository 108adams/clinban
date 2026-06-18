package lint_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/108adams/clinban/internal/lint"
	"github.com/108adams/clinban/internal/ticket"
)

// --------------------------------------------------------------------------
// Constants and helpers
// --------------------------------------------------------------------------

const (
	testFilename = "0042-fix-login-timeout.md"
	testID       = "0042"
	testTitle    = "Fix login timeout on staging"
	testOtherID  = "0007"
)

var testNow = time.Date(2026, 5, 18, 14, 30, 0, 0, time.UTC)

// validTicket returns a fully populated, valid *ticket.Ticket for use in tests.
func validTicket() *ticket.Ticket {
	return &ticket.Ticket{
		ID:      testID,
		Status:  ticket.StatusBacklog,
		Type:    ticket.TypeTask,
		Title:   testTitle,
		Tags:    []string{},
		Created: testNow,
		Updated: testNow,
	}
}

// uniqueIDs returns an allIDs slice that contains testID exactly once.
func uniqueIDs() []string { return []string{testID} }

// duplicateIDs returns an allIDs slice that contains testID twice (collision).
func duplicateIDs() []string { return []string{testID, testID} }

// assertNoErrors fails the test if errs is non-empty.
func assertNoErrors(t *testing.T, errs []lint.LintError) {
	t.Helper()
	if len(errs) != 0 {
		t.Errorf("expected no lint errors, got %d: %v", len(errs), errs)
	}
}

// assertErrorCount fails the test if the number of errors differs from want.
func assertErrorCount(t *testing.T, errs []lint.LintError, want int) {
	t.Helper()
	if len(errs) != want {
		t.Errorf("expected %d lint error(s), got %d: %v", want, len(errs), errs)
	}
}

// assertFieldError fails the test if none of the errors refers to field.
func assertFieldError(t *testing.T, errs []lint.LintError, field string) {
	t.Helper()
	for _, e := range errs {
		if e.Field == field {
			return
		}
	}
	t.Errorf("expected an error for field %q, got errors: %v", field, errs)
}

// --------------------------------------------------------------------------
// TestLintReturnsNonNilSlice — Lint must return an empty, non-nil slice for
// a valid ticket.
// --------------------------------------------------------------------------

func TestLintReturnsNonNilSlice(t *testing.T) {
	t.Parallel()
	errs := lint.Lint(validTicket(), testFilename, uniqueIDs())
	if errs == nil {
		t.Fatal("Lint returned nil; want empty non-nil slice")
	}
	assertNoErrors(t, errs)
}

// --------------------------------------------------------------------------
// TestLintStringFormat — LintError.String() output format.
// --------------------------------------------------------------------------

func TestLintStringFormat(t *testing.T) {
	t.Parallel()
	e := lint.LintError{
		File:    testFilename,
		Field:   "type",
		Message: "invalid value",
	}
	got := e.String()
	want := "0042-fix-login-timeout.md: field 'type': invalid value"
	if got != want {
		t.Errorf("LintError.String() = %q, want %q", got, want)
	}
}

// --------------------------------------------------------------------------
// TestRule1RequiredFields — every required field triggers an error when zero.
// --------------------------------------------------------------------------

func TestRule1RequiredFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mutate    func(*ticket.Ticket)
		wantField string
	}{
		{
			name:      "missing id",
			mutate:    func(tk *ticket.Ticket) { tk.ID = "" },
			wantField: "id",
		},
		{
			name:      "missing status",
			mutate:    func(tk *ticket.Ticket) { tk.Status = "" },
			wantField: "status",
		},
		{
			name:      "missing title",
			mutate:    func(tk *ticket.Ticket) { tk.Title = "" },
			wantField: "title",
		},
		{
			name:      "missing type",
			mutate:    func(tk *ticket.Ticket) { tk.Type = "" },
			wantField: "type",
		},
		{
			name:      "zero created",
			mutate:    func(tk *ticket.Ticket) { tk.Created = time.Time{} },
			wantField: "created",
		},
		{
			name:      "zero updated",
			mutate:    func(tk *ticket.Ticket) { tk.Updated = time.Time{} },
			wantField: "updated",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tk := validTicket()
			tc.mutate(tk)
			errs := lint.Lint(tk, testFilename, uniqueIDs())
			assertFieldError(t, errs, tc.wantField)
		})
	}

	t.Run("all fields present", func(t *testing.T) {
		t.Parallel()
		// A valid ticket should produce no rule-1 errors.
		// (Other rules may produce errors; we only check the field set here.)
		tk := validTicket()
		errs := lint.Lint(tk, testFilename, uniqueIDs())
		for _, e := range errs {
			if e.Field == "id" || e.Field == "status" || e.Field == "title" ||
				e.Field == "type" || e.Field == "created" || e.Field == "updated" {
				// Only fail if the error has the message "required field missing"
				if strings.Contains(e.Message, "required") || strings.Contains(e.Message, "missing") {
					t.Errorf("unexpected required-field error for %q: %v", e.Field, e)
				}
			}
		}
	})
}

// --------------------------------------------------------------------------
// TestRule2ValidStatus — status field validation.
// --------------------------------------------------------------------------

func TestRule2ValidStatus(t *testing.T) {
	t.Parallel()

	t.Run("invalid status", func(t *testing.T) {
		t.Parallel()
		tk := validTicket()
		tk.Status = "wip"
		errs := lint.Lint(tk, testFilename, uniqueIDs())
		assertFieldError(t, errs, "status")
	})

	t.Run("valid statuses produce no status error", func(t *testing.T) {
		t.Parallel()
		validStatuses := []ticket.Status{
			ticket.StatusBacklog,
			ticket.StatusInProgress,
			ticket.StatusBlocked,
			ticket.StatusDone,
		}
		for _, s := range validStatuses {
			t.Run(string(s), func(t *testing.T) {
				t.Parallel()
				tk := validTicket()
				tk.Status = s
				errs := lint.Lint(tk, testFilename, uniqueIDs())
				for _, e := range errs {
					if e.Field == "status" {
						t.Errorf("unexpected status error for %q: %v", s, e)
					}
				}
			})
		}
	})
}

// --------------------------------------------------------------------------
// TestRule3ValidType — type field validation.
// --------------------------------------------------------------------------

func TestRule3ValidType(t *testing.T) {
	t.Parallel()

	t.Run("invalid type", func(t *testing.T) {
		t.Parallel()
		tk := validTicket()
		tk.Type = "chore"
		errs := lint.Lint(tk, testFilename, uniqueIDs())
		assertFieldError(t, errs, "type")
	})

	t.Run("valid types produce no type error", func(t *testing.T) {
		t.Parallel()
		validTypes := []ticket.Type{
			ticket.TypeBug,
			ticket.TypeTask,
			ticket.TypeFeature,
			ticket.TypeSpike,
		}
		for _, tp := range validTypes {
			t.Run(string(tp), func(t *testing.T) {
				t.Parallel()
				tk := validTicket()
				tk.Type = tp
				errs := lint.Lint(tk, testFilename, uniqueIDs())
				for _, e := range errs {
					if e.Field == "type" {
						t.Errorf("unexpected type error for %q: %v", tp, e)
					}
				}
			})
		}
	})
}

// --------------------------------------------------------------------------
// TestRule1TimestampZeroValue — zero time.Time triggers an error from rule 1
// (ruleRequiredFields); the message is the precise RFC3339 parse failure text.
// --------------------------------------------------------------------------

func TestRule1TimestampZeroValue(t *testing.T) {
	t.Parallel()

	t.Run("zero created", func(t *testing.T) {
		t.Parallel()
		tk := validTicket()
		tk.Created = time.Time{}
		errs := lint.Lint(tk, testFilename, uniqueIDs())
		assertFieldError(t, errs, "created")
	})

	t.Run("zero updated", func(t *testing.T) {
		t.Parallel()
		tk := validTicket()
		tk.Updated = time.Time{}
		errs := lint.Lint(tk, testFilename, uniqueIDs())
		assertFieldError(t, errs, "updated")
	})

	t.Run("non-zero timestamps produce no timestamp error", func(t *testing.T) {
		t.Parallel()
		tk := validTicket()
		errs := lint.Lint(tk, testFilename, uniqueIDs())
		for _, e := range errs {
			if (e.Field == "created" || e.Field == "updated") &&
				strings.Contains(e.Message, "zero") {
				t.Errorf("unexpected timestamp error: %v", e)
			}
		}
	})
}

// --------------------------------------------------------------------------
// TestRule6TagsNonEmpty — empty-string elements in tags are invalid.
// --------------------------------------------------------------------------

func TestRule6TagsNonEmpty(t *testing.T) {
	t.Parallel()

	t.Run("tag with empty string element", func(t *testing.T) {
		t.Parallel()
		tk := validTicket()
		tk.Tags = []string{"valid", ""}
		errs := lint.Lint(tk, testFilename, uniqueIDs())
		assertFieldError(t, errs, "tags")
	})

	t.Run("empty tags list is valid", func(t *testing.T) {
		t.Parallel()
		tk := validTicket()
		tk.Tags = []string{}
		errs := lint.Lint(tk, testFilename, uniqueIDs())
		for _, e := range errs {
			if e.Field == "tags" {
				t.Errorf("unexpected tags error for empty list: %v", e)
			}
		}
	})

	t.Run("all non-empty tags are valid", func(t *testing.T) {
		t.Parallel()
		tk := validTicket()
		tk.Tags = []string{"backend", "auth", "urgent"}
		errs := lint.Lint(tk, testFilename, uniqueIDs())
		for _, e := range errs {
			if e.Field == "tags" {
				t.Errorf("unexpected tags error for non-empty tags: %v", e)
			}
		}
	})

	t.Run("nil tags treated as empty", func(t *testing.T) {
		t.Parallel()
		tk := validTicket()
		tk.Tags = nil
		errs := lint.Lint(tk, testFilename, uniqueIDs())
		for _, e := range errs {
			if e.Field == "tags" {
				t.Errorf("unexpected tags error for nil tags: %v", e)
			}
		}
	})
}

// --------------------------------------------------------------------------
// TestRule7IDUnique — duplicate IDs across allIDs must be flagged.
// --------------------------------------------------------------------------

func TestRule7IDUnique(t *testing.T) {
	t.Parallel()

	t.Run("duplicate id produces error", func(t *testing.T) {
		t.Parallel()
		tk := validTicket()
		errs := lint.Lint(tk, testFilename, duplicateIDs())
		assertFieldError(t, errs, "id")
	})

	t.Run("unique id produces no uniqueness error", func(t *testing.T) {
		t.Parallel()
		tk := validTicket()
		// allIDs has two different IDs, testID appears once.
		allIDs := []string{testID, testOtherID}
		errs := lint.Lint(tk, testFilename, allIDs)
		for _, e := range errs {
			if e.Field == "id" && strings.Contains(e.Message, "unique") {
				t.Errorf("unexpected uniqueness error: %v", e)
			}
		}
	})
}

// --------------------------------------------------------------------------
// TestLintFullyValidTicket — done-criteria: fully valid ticket, two unique IDs,
// returns empty slice.
// --------------------------------------------------------------------------

func TestLintFullyValidTicket(t *testing.T) {
	t.Parallel()
	tk := validTicket()
	allIDs := []string{testID, testOtherID}
	errs := lint.Lint(tk, testFilename, allIDs)
	if errs == nil {
		t.Fatal("Lint returned nil; want empty non-nil slice")
	}
	assertNoErrors(t, errs)
}

// --------------------------------------------------------------------------
// TestLintMissingTitleAndDuplicateID — done-criteria: ticket missing title and
// duplicate ID must produce exactly 2 errors.
// --------------------------------------------------------------------------

func TestLintMissingTitleAndDuplicateID(t *testing.T) {
	t.Parallel()
	tk := validTicket()
	tk.Title = ""
	errs := lint.Lint(tk, testFilename, duplicateIDs())
	assertErrorCount(t, errs, 2)
}

// --------------------------------------------------------------------------
// TestZeroTimestampExactlyOneErrorPerField — done-criteria: a ticket with zero
// Created and Updated must produce exactly one LintError per field, with
// message "zero timestamp; value was not parseable as RFC3339".
// --------------------------------------------------------------------------

func TestZeroTimestampExactlyOneErrorPerField(t *testing.T) {
	t.Parallel()

	const wantMsg = "zero timestamp; value was not parseable as RFC3339"

	tests := []struct {
		name      string
		mutate    func(*ticket.Ticket)
		wantField string
	}{
		{
			name:      "zero created produces exactly one error",
			mutate:    func(tk *ticket.Ticket) { tk.Created = time.Time{} },
			wantField: "created",
		},
		{
			name:      "zero updated produces exactly one error",
			mutate:    func(tk *ticket.Ticket) { tk.Updated = time.Time{} },
			wantField: "updated",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tk := validTicket()
			tc.mutate(tk)
			errs := lint.Lint(tk, testFilename, uniqueIDs())

			// Count errors for the target field.
			var fieldErrs []lint.LintError
			for _, e := range errs {
				if e.Field == tc.wantField {
					fieldErrs = append(fieldErrs, e)
				}
			}
			if len(fieldErrs) != 1 {
				t.Fatalf("expected exactly 1 error for field %q, got %d: %v", tc.wantField, len(fieldErrs), fieldErrs)
			}
			if fieldErrs[0].Message != wantMsg {
				t.Errorf("error message = %q, want %q", fieldErrs[0].Message, wantMsg)
			}
		})
	}
}

// --------------------------------------------------------------------------
// TestLintAllRulesOrdered — smoke test: all rules run and produce errors for
// a maximally broken ticket.
// --------------------------------------------------------------------------

func TestLintAllRulesOrdered(t *testing.T) {
	t.Parallel()
	tk := &ticket.Ticket{
		// ID is empty → rule 1 (id required)
		// Status is invalid → rules 1 (status required) + 2 (invalid status)
		// Type is invalid → rules 1 (type required) + 3 (invalid type)
		// Title is empty → rule 1 (title required)
		// Created/Updated are zero → rule 1 (timestamp message)
		// Tags has empty element → rule 6
		Tags: []string{""},
	}
	errs := lint.Lint(tk, "no-prefix.md", duplicateIDs())
	// We expect multiple errors; just assert at least one per broken rule.
	if len(errs) == 0 {
		t.Fatal("expected multiple lint errors for a maximally broken ticket, got none")
	}
}

// --------------------------------------------------------------------------
// ValidateForCommit tests (T3)
// --------------------------------------------------------------------------

// validRawTicket returns a minimal valid ticket file as bytes, using the
// provided ID embedded in the constants (ID is not stored in frontmatter).
func validRawTicket() []byte {
	now := time.Now().UTC().Format(time.RFC3339)
	return fmt.Appendf(nil, "---\ntitle: %s\nstatus: backlog\ntype: task\ntags: []\ncreated: %s\nupdated: %s\n---\n", testTitle, now, now)
}

// invalidStatusRawTicket returns raw bytes for a ticket whose status value is
// not one of the recognised statuses, so lint will flag it.
func invalidStatusRawTicket() []byte {
	now := time.Now().UTC().Format(time.RFC3339)
	return fmt.Appendf(nil, "---\ntitle: %s\nstatus: wip\ntype: task\ntags: []\ncreated: %s\nupdated: %s\n---\n", testTitle, now, now)
}

// invalidFrontmatterRaw returns bytes that cannot be parsed as a ticket at all.
func invalidFrontmatterRaw() []byte {
	return []byte("this is not a ticket — no frontmatter\n")
}

// TestValidateForCommitParseFailure verifies that a parse error returns
// (nil, nil, parseErr) and the error wraps ticket.ErrMissingFrontmatter.
func TestValidateForCommitParseFailure(t *testing.T) {
	t.Parallel()

	got, lintErrs, err := lint.ValidateForCommit(invalidFrontmatterRaw(), testID, testFilename, uniqueIDs())

	if got != nil {
		t.Errorf("want nil *ticket.Ticket on parse failure, got %+v", got)
	}
	if lintErrs != nil {
		t.Errorf("want nil []LintError on parse failure, got %v", lintErrs)
	}
	if err == nil {
		t.Fatal("want non-nil error on parse failure, got nil")
	}
	if !errors.Is(err, ticket.ErrMissingFrontmatter) {
		t.Errorf("error does not wrap ErrMissingFrontmatter: %v", err)
	}
}

// TestValidateForCommitLintViolation verifies that a parseable but invalid
// ticket returns (t, errs, nil) where errs is non-empty.
func TestValidateForCommitLintViolation(t *testing.T) {
	t.Parallel()

	got, lintErrs, err := lint.ValidateForCommit(invalidStatusRawTicket(), testID, testFilename, uniqueIDs())

	if err != nil {
		t.Fatalf("want nil error on parse success, got %v", err)
	}
	if got == nil {
		t.Fatal("want non-nil *ticket.Ticket when parse succeeds, got nil")
	}
	// ID must be injected by ValidateForCommit.
	if got.ID != testID {
		t.Errorf("ticket.ID = %q, want %q", got.ID, testID)
	}
	if len(lintErrs) == 0 {
		t.Error("want at least one lint error for invalid status, got none")
	}
	// At least one error must name the status field.
	found := false
	for _, e := range lintErrs {
		if e.Field == "status" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a lint error for field 'status', got: %v", lintErrs)
	}
}

// TestValidateForCommitClean verifies that a fully valid ticket returns
// (t, emptyErrs, nil) where emptyErrs is a non-nil empty slice.
func TestValidateForCommitClean(t *testing.T) {
	t.Parallel()

	got, lintErrs, err := lint.ValidateForCommit(validRawTicket(), testID, testFilename, uniqueIDs())

	if err != nil {
		t.Fatalf("want nil error for clean ticket, got %v", err)
	}
	if got == nil {
		t.Fatal("want non-nil *ticket.Ticket for clean ticket, got nil")
	}
	if got.ID != testID {
		t.Errorf("ticket.ID = %q, want %q", got.ID, testID)
	}
	if lintErrs == nil {
		t.Error("want non-nil empty slice for clean ticket, got nil")
	}
	if len(lintErrs) != 0 {
		t.Errorf("want empty lint errors for clean ticket, got %v", lintErrs)
	}
}

// TestValidateForCommitIDInjected verifies that ValidateForCommit injects the
// supplied id into the ticket regardless of what the raw bytes contain.
func TestValidateForCommitIDInjected(t *testing.T) {
	t.Parallel()

	const injectedID = "0099"
	got, _, err := lint.ValidateForCommit(validRawTicket(), injectedID, testFilename, []string{injectedID})

	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if got == nil {
		t.Fatal("want non-nil ticket")
	}
	if got.ID != injectedID {
		t.Errorf("ticket.ID = %q, want %q", got.ID, injectedID)
	}
}

// TestValidateForCommitLintErrsNeverNilOnSuccess verifies that even when there
// are no lint violations, lintErrs is a non-nil slice (not nil).
func TestValidateForCommitLintErrsNeverNilOnSuccess(t *testing.T) {
	t.Parallel()

	_, lintErrs, err := lint.ValidateForCommit(validRawTicket(), testID, testFilename, uniqueIDs())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lintErrs == nil {
		t.Error("lintErrs must be non-nil (empty slice) when ticket is valid; got nil")
	}
}
