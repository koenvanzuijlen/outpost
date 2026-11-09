package server

import "time"

// sortOrder is the direction of a sort, as in the query value order.
type sortOrder string

// The possible values for sortOrder.
const (
	sortOrderAsc  sortOrder = "asc"
	sortOrderDesc sortOrder = "desc"
)

// Valid reports whether o is a known sort order.
func (o sortOrder) Valid() bool {
	return o == sortOrderAsc || o == sortOrderDesc
}

// Reverse returns sortOrderDesc for sortOrderAsc, and sortOrderAsc for every other value.
func (o sortOrder) Reverse() sortOrder {
	if o == sortOrderAsc {
		return sortOrderDesc
	}
	return sortOrderAsc
}

// Apply returns the ascending comparison result c in the direction of o: c for sortOrderAsc, and -c for every other
// value.
func (o sortOrder) Apply(c int) int {
	if o == sortOrderAsc {
		return c
	}
	return -c
}

// compareDatesNilLast compares two optional dates in the given order. Nil dates are last in both orders.
func compareDatesNilLast(a, b *time.Time, order sortOrder) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return 1
	}
	if b == nil {
		return -1
	}
	return order.Apply(a.Compare(*b))
}
