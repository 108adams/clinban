package ticket

// Status is the workflow state stored in a ticket's YAML frontmatter.
//
// Clinban's CLI enforces transitions between status values through package fsm.
// Direct file writers may set any string value, but package lint reports values
// outside this set as schema violations.
type Status string

const (
	// StatusBacklog marks work that is known but not currently being worked.
	StatusBacklog Status = "backlog"
	// StatusInProgress marks work that is actively being worked.
	StatusInProgress Status = "in-progress"
	// StatusBlocked marks work that cannot currently proceed.
	StatusBlocked Status = "blocked"
	// StatusDone marks completed work that may be archived.
	StatusDone Status = "done"
)

// Valid reports whether s is one of the supported Clinban status values.
func (s Status) Valid() bool {
	switch s {
	case StatusBacklog, StatusInProgress, StatusBlocked, StatusDone:
		return true
	default:
		return false
	}
}
