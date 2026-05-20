package lint

import (
	"fmt"
	"strings"

	"github.com/108adams/clinban/internal/ticket"
)

// ruleRequiredFields checks that all required fields are non-zero.
// Rule 1: id, status, title, type, created, updated are non-zero.
func ruleRequiredFields(t *ticket.Ticket, filename string, _ []string) []LintError {
	var errs []LintError

	if t.ID == "" {
		errs = append(errs, LintError{File: filename, Field: "id", Message: "required field missing"})
	}
	if string(t.Status) == "" {
		errs = append(errs, LintError{File: filename, Field: "status", Message: "required field missing"})
	}
	if t.Title == "" {
		errs = append(errs, LintError{File: filename, Field: "title", Message: "required field missing"})
	}
	if string(t.Type) == "" {
		errs = append(errs, LintError{File: filename, Field: "type", Message: "required field missing"})
	}
	if t.Created.IsZero() {
		errs = append(errs, LintError{File: filename, Field: "created", Message: "zero timestamp; value was not parseable as RFC3339"})
	}
	if t.Updated.IsZero() {
		errs = append(errs, LintError{File: filename, Field: "updated", Message: "zero timestamp; value was not parseable as RFC3339"})
	}

	return errs
}

// ruleValidStatus checks that the status field holds a recognised value.
// Rule 2: status is one of the valid Status constants.
// Only runs if status is non-empty (rule 1 already flags the empty case).
func ruleValidStatus(t *ticket.Ticket, filename string, _ []string) []LintError {
	if string(t.Status) == "" {
		return nil
	}
	if !t.Status.Valid() {
		return []LintError{{
			File:    filename,
			Field:   "status",
			Message: fmt.Sprintf("invalid value %q; must be one of: backlog, in-progress, blocked, done", t.Status),
		}}
	}
	return nil
}

// ruleValidType checks that the type field holds a recognised value.
// Rule 3: type is one of the valid Type constants.
// Only runs if type is non-empty (rule 1 already flags the empty case).
func ruleValidType(t *ticket.Ticket, filename string, _ []string) []LintError {
	if string(t.Type) == "" {
		return nil
	}
	if !t.Type.Valid() {
		return []LintError{{
			File:    filename,
			Field:   "type",
			Message: fmt.Sprintf("invalid value %q; must be one of: bug, task, feature, spike", t.Type),
		}}
	}
	return nil
}

// ruleIDMatchesFilename checks that the numeric prefix of filename matches t.ID.
// Rule 4: extract leading digits from filename, compare with t.ID.
func ruleIDMatchesFilename(t *ticket.Ticket, filename string, _ []string) []LintError {
	// Extract the leading run of digit characters from the base filename.
	prefix := leadingDigits(filename)
	if prefix == "" || prefix != t.ID {
		return []LintError{{
			File:    filename,
			Field:   "id",
			Message: fmt.Sprintf("id %q does not match numeric prefix of filename %q", t.ID, filename),
		}}
	}
	return nil
}

// leadingDigits returns the longest prefix of s consisting solely of ASCII digit characters.
func leadingDigits(s string) string {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return s[:i]
}

// ruleTagsNonEmpty checks that every element in the tags list is a non-empty string.
// Rule 6: no zero-value (empty-string) entries in tags.
func ruleTagsNonEmpty(t *ticket.Ticket, filename string, _ []string) []LintError {
	for i, tag := range t.Tags {
		if strings.TrimSpace(tag) == "" {
			return []LintError{{
				File:    filename,
				Field:   "tags",
				Message: fmt.Sprintf("element %d is an empty string", i),
			}}
		}
	}
	return nil
}

// ruleIDUnique checks that t.ID appears exactly once in allIDs.
// Rule 7: flag if count > 1.
func ruleIDUnique(t *ticket.Ticket, filename string, allIDs []string) []LintError {
	count := 0
	for _, id := range allIDs {
		if id == t.ID {
			count++
		}
	}
	if count > 1 {
		return []LintError{{
			File:    filename,
			Field:   "id",
			Message: fmt.Sprintf("id %q is not unique; found %d occurrences across active and archive", t.ID, count),
		}}
	}
	return nil
}
