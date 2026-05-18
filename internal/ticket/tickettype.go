package ticket

// Type represents the category of work a ticket describes.
type Type string

const (
	TypeBug     Type = "bug"
	TypeTask    Type = "task"
	TypeFeature Type = "feature"
	TypeSpike   Type = "spike"
)

// Valid reports whether t is one of the four recognised Type values.
func (t Type) Valid() bool {
	switch t {
	case TypeBug, TypeTask, TypeFeature, TypeSpike:
		return true
	default:
		return false
	}
}
