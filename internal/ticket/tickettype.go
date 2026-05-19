package ticket

// Type is the category of work described by a ticket.
//
// Type is intentionally a small controlled vocabulary so list filtering and
// automation can rely on stable values.
type Type string

const (
	// TypeBug identifies corrective work for broken behavior.
	TypeBug Type = "bug"
	// TypeTask identifies routine implementation or maintenance work.
	TypeTask Type = "task"
	// TypeFeature identifies user-visible product capability work.
	TypeFeature Type = "feature"
	// TypeSpike identifies time-boxed research or discovery work.
	TypeSpike Type = "spike"
)

// Valid reports whether t is one of the supported Clinban ticket types.
func (t Type) Valid() bool {
	switch t {
	case TypeBug, TypeTask, TypeFeature, TypeSpike:
		return true
	default:
		return false
	}
}
