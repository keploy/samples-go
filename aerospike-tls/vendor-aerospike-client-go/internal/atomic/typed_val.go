package atomic

import "sync/atomic"

// TypedVal allows synchronized access to a value
type TypedVal[T any] atomic.Value

// Set updates the value of TypedVal with the passed argument
func (sv *TypedVal[T]) Set(val T) {
	(*atomic.Value)(sv).Store(&val)
}

// Get returns the value inside the TypedVal
func (sv *TypedVal[T]) Get() T {
	res := (*atomic.Value)(sv).Load()
	if res != nil {
		return *res.(*T)
	}

	// return zero value; for pointers, it will be nil
	var t T
	return t
}
