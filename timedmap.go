package timedmap

import (
	"sync"
	"sync/atomic"
	"time"
)

// Map contains a map with all key-value pairs. It does not automatically clean
// up.
type Map struct {
	mtx       sync.RWMutex
	container map[interface{}]*Element
}

var _ Cleanable = (*Map)(nil)

// Element contains the actual value as interface type and the time when the
// value expires.
type Element struct {
	Value   interface{}
	expires int64 // unixnano
}

// something something low allocations
var nilElement = Element{}

// Expires returns the expiry time in UnixNano. This method is thread-safe.
func (e *Element) Expires() int64 {
	return atomic.LoadInt64(&e.expires)
}

// ExpiryTime returns the expiry time in time.Time. This method is thread-safe.
func (e *Element) ExpiryTime() time.Time {
	return time.Unix(0, e.Expires())
}

// New creates and returns a new instance of Map. This Map does not
// periodically clean up.
func New() *Map {
	return &Map{
		container: make(map[interface{}]*Element),
	}
}

// Set appends a key-value pair to the map or sets the value of
// a key. expiresAfter sets the expire time after the key-value pair
// will automatically be removed from the map.
func (tm *Map) Set(key, value interface{}, expiresAfter time.Duration) {
	tm.mtx.Lock()
	defer tm.mtx.Unlock()

	tm.container[key] = &Element{
		Value:   value,
		expires: time.Now().Add(expiresAfter).UnixNano(),
	}
}

// GetValue returns an interface of the value of a key in the map. The returned
// value is nil if there is no value to the passed key or if the value was
// expired.
func (tm *Map) GetValue(key interface{}) interface{} {
	v, ok := tm.Get(key)
	if ok {
		return v.Value
	}
	return nil
}

// GetExpires returns the expire time of a key-value pair. If the key-value pair
// does not exist in the map or was expired, this will return false.
func (tm *Map) GetExpires(key interface{}) (time.Time, bool) {
	v, ok := tm.Get(key)
	if ok {
		return v.ExpiryTime(), true
	}
	return time.Time{}, false
}

// Contains returns true, if the key exists in the map.
// false will be returned, if there is no value to the
// key or if the key-value pair was expired.
func (tm *Map) Contains(key interface{}) bool {
	_, ok := tm.Get(key)
	return ok
}

// Remove deletes a key-value pair in the map.
func (tm *Map) Remove(key interface{}) {
	tm.mtx.Lock()
	delete(tm.container, key)
	tm.mtx.Unlock()
}

// Extend adds the duration into the expiry time.
func (tm *Map) Extend(key interface{}, d time.Duration) bool {
	v, ok := tm.Get(key)
	if ok {
		atomic.AddInt64(&v.expires, int64(d))
	}
	return ok
}

// Flush deletes all key-value pairs of the map.
func (tm *Map) Flush() {
	tm.mtx.Lock()
	defer tm.mtx.Unlock()

	tm.container = make(map[interface{}]*Element)
}

// Size returns the current number of key-value pairs
// existent in the map.
func (tm *Map) Size() int {
	tm.mtx.RLock()
	defer tm.mtx.RUnlock()

	return len(tm.container)
}

// cleanUp iterates trhough the map and expires all key-value
// pairs which expire time after the current time
func (tm *Map) Cleanup() {
	tm.mtx.Lock()
	defer tm.mtx.Unlock()

	// getting now after mutex to prevent drifting
	now := time.Now().UnixNano()

	for k, v := range tm.container {
		if now > v.expires {
			delete(tm.container, k)
		}
	}
}

// Get returns an element object by key.
func (tm *Map) Get(key interface{}) (*Element, bool) {
	tm.mtx.RLock()
	v, ok := tm.container[key]
	tm.mtx.RUnlock()

	// let the cleaner do the job.
	if !ok || time.Now().UnixNano() > v.Expires() {
		return nil, false
	}

	return v, true
}
