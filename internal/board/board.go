package board

import (
	"strconv"

	"github.com/108adams/clinban/internal/ticket"
)

// StatusOrder ranks statuses for board display. Lower values sort earlier.
//
// Order: in-progress (0) → blocked (1) → backlog (2) → done (3).
var StatusOrder = map[ticket.Status]int{
	ticket.StatusInProgress: 0,
	ticket.StatusBlocked:    1,
	ticket.StatusBacklog:    2,
	ticket.StatusDone:       3,
}

// Less reports whether ticket a should sort before ticket b in board order.
//
// Primary key: StatusOrder rank (lower sorts earlier).
// Secondary key: numeric ticket ID, ascending.
// Unknown statuses receive rank 0 (the map zero value).
func Less(a, b *ticket.Ticket) bool {
	oa := StatusOrder[a.Status]
	ob := StatusOrder[b.Status]
	if oa != ob {
		return oa < ob
	}
	na, _ := strconv.Atoi(a.ID)
	nb, _ := strconv.Atoi(b.ID)
	return na < nb
}
