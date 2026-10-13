package server

// ptrIfNonZero returns a pointer to the given value if it is non-zero, otherwise it returns nil.
func ptrIfNonZero[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}

// derefOrZero returns the value p points to, or the zero value of T if p is nil.
func derefOrZero[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}
