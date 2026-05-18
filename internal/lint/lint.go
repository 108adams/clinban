package lint

import (
	"fmt"

	"clinban/internal/ticket"
)

// LintError represents a single schema violation found in a ticket file.
type LintError struct {
	File    string // base filename only, not a full path
	Field   string
	Message string
}

// String returns the canonical one-line representation of the error:
//
//	"0042-fix-login-timeout.md: field 'type': invalid value"
func (e LintError) String() string {
	return fmt.Sprintf("%s: field '%s': %s", e.File, e.Field, e.Message)
}

// Error implements the error interface so LintError can be used as an error value.
func (e LintError) Error() string {
	return e.String()
}

// ruleFunc is the signature shared by all rule functions.
type ruleFunc func(t *ticket.Ticket, filename string, allIDs []string) []LintError

// rules is the ordered list of all 7 lint rules.
var rules = []ruleFunc{
	ruleRequiredFields,    // 1
	ruleValidStatus,       // 2
	ruleValidType,         // 3
	ruleIDMatchesFilename, // 4
	ruleTimestampsNonZero, // 5
	ruleTagsNonEmpty,      // 6
	ruleIDUnique,          // 7
}

// Lint runs all 7 rules against the ticket in order and returns every violation
// found. filename is the base filename (used in error output and for rule 4).
// allIDs is the full list of IDs across active + archive (used for rule 7).
//
// Returns an empty (never nil) slice when the ticket is valid.
func Lint(t *ticket.Ticket, filename string, allIDs []string) []LintError {
	result := []LintError{}
	for _, rule := range rules {
		result = append(result, rule(t, filename, allIDs)...)
	}
	return result
}
