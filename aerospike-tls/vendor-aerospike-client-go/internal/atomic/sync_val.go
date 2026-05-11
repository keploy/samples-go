package atomic

import "sync"

// SyncVal allows synchronized access to a value
type SyncVal[T any] struct {
	val  T
	lock sync.RWMutex
}

// NewSyncVal creates a new instance of SyncVal
func NewSyncVal[T any](val T) *SyncVal[T] {
	return &SyncVal[T]{val: val}
}

// Set updates the value of SyncVal with the passed argument
func (sv *SyncVal[T]) Set(val T) {
	sv.lock.Lock()
	sv.val = val
	sv.lock.Unlock()
}

// Get returns the value inside the SyncVal
func (sv *SyncVal[T]) Get() T {
	sv.lock.RLock()
	val := sv.val
	sv.lock.RUnlock()
	return val
}

// GetSyncedVia returns the value returned by the function f.
func (sv *SyncVal[T]) GetSyncedVia(f func(T) (T, error)) (T, error) {
	sv.lock.RLock()
	defer sv.lock.RUnlock()

	val, err := f(sv.val)
	return val, err
}

// Update gets a function and passes the value of SyncVal to it.
// If the resulting err is nil, it will update the value of SyncVal.
// It will return the resulting error to the caller.
func (sv *SyncVal[T]) Update(f func(T) (T, error)) error {
	sv.lock.Lock()
	defer sv.lock.Unlock()

	val, err := f(sv.val)
	if err == nil {
		sv.val = val
	}
	return err
}

// MapSyncValue returns the value returned by the function f.
func MapSyncValue[T any, U any](sv *SyncVal[T], f func(T) (U, error)) (U, error) {
	sv.lock.RLock()
	defer sv.lock.RUnlock()

	val, err := f(sv.val)
	return val, err
}
