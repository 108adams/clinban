package ticket

// Status represents the workflow state of a ticket.
type Status string

const (
	StatusBacklog    Status = "backlog"
	StatusInProgress Status = "in-progress"
	StatusBlocked    Status = "blocked"
	StatusDone       Status = "done"
)

// Valid reports whether s is one of the four recognised Status values.
func (s Status) Valid() bool {
	switch s {
	case StatusBacklog, StatusInProgress, StatusBlocked, StatusDone:
		return true
	default:
		return false
	}
}
