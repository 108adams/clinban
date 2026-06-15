package lint

import (
	"fmt"

	"github.com/108adams/clinban/internal/ticket"
)

// LintError is a single schema violation found in a ticket file.
type LintError struct {
	// File is the base filename reported to users, not an absolute path.
	File string
	// Field is the schema field associated with the violation.
	Field string
	// Message explains the violation in user-facing language.
	Message string
}

// String returns the canonical one-line representation of the error:
//
//	"0042-fix-login-timeout.md: field 'type': invalid value"
func (e LintError) String() string {
	return fmt.Sprintf("%s: field '%s': %s", e.File, e.Field, e.Message)
}

// Error implements the error interface by returning String.
func (e LintError) Error() string {
	return e.String()
}

// ruleFunc is the signature shared by all rule functions.
type ruleFunc func(t *ticket.Ticket, filename string, allIDs []string) []LintError

// rules is the ordered list of all lint rules.
var rules = []ruleFunc{
	ruleRequiredFields,
	ruleValidStatus,
	ruleValidType,
	ruleTagsNonEmpty,
	ruleIDUnique,
}

// Lint runs all schema rules against t and returns every violation found.
//
// The filename argument should be the base filename used for user-facing error
// output. The allIDs argument is the repository context used for uniqueness
// checks across active and archived tickets.
//
// Precondition: t.ID must be set before calling Lint; ruleRequiredFields will
// report a violation if t.ID is empty. On the read path, store.ReadTicket
// injects the ID from the filename automatically. On the write path (new,
// register), callers must set t.ID explicitly before calling Lint.
//
// Returns an empty (never nil) slice when the ticket is valid.
func Lint(t *ticket.Ticket, filename string, allIDs []string) []LintError {
	result := []LintError{}
	for _, rule := range rules {
		result = append(result, rule(t, filename, allIDs)...)
	}
	return result
}

// ValidateForCommit parses raw bytes, assigns id to the parsed ticket, then
// runs Lint with filename and allIDs.
//
// Returns (nil, nil, parseErr) when ticket.Parse fails.
// Returns (t, lintErrs, nil) otherwise — lintErrs is an empty (never nil)
// slice when the ticket is valid.
func ValidateForCommit(raw []byte, id, filename string, allIDs []string) (*ticket.Ticket, []LintError, error) {
	t, err := ticket.Parse(raw)
	if err != nil {
		return nil, nil, err
	}
	t.ID = id
	return t, Lint(t, filename, allIDs), nil
}
