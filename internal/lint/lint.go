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

// rules is the ordered list of all 6 lint rules.
var rules = []ruleFunc{
	ruleRequiredFields,    // 1
	ruleValidStatus,       // 2
	ruleValidType,         // 3
	ruleIDMatchesFilename, // 4
	ruleTagsNonEmpty,      // 5
	ruleIDUnique,          // 6
}

// Lint runs all schema rules against t and returns every violation found.
//
// The filename argument should be the base filename used for user-facing error
// output and for the rule that checks whether the filename prefix matches the
// ticket ID. The allIDs argument is the repository context used for uniqueness
// checks across active and archived tickets.
//
// Returns an empty (never nil) slice when the ticket is valid.
func Lint(t *ticket.Ticket, filename string, allIDs []string) []LintError {
	result := []LintError{}
	for _, rule := range rules {
		result = append(result, rule(t, filename, allIDs)...)
	}
	return result
}
