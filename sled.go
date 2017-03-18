package sled

import (
	"errors"
	"reflect"
	"sync"
)

// Create a new Sled object with optional custom configuration.
func New() Sled {
	ct := newCtrie(nil)
	return &sled{ct}
}

type sled struct {
	ct *ctrie
}

type ele struct {
	k string
	v interface{}
	c func()
}

func (e *ele) Close() {
	if e.c != nil {
		e.c()
	}
}

func (e *ele) Key() string {
	return e.k
}

func (e *ele) Value() interface{} {
	return e.v
}

// Assigns value to key, replacing any previous values.
func (s *sled) Set(key string, value interface{}) error {
	s.ct.Insert([]byte(key), value)
	return nil
}

// SetNil is exclusive Set.  It only assigns the value to the key,
// if the key is not already set.  It returns true if the assignment succeed.
func (s *sled) SetIfNil(key string, value interface{}) bool {
	if _, existed := s.ct.Lookup([]byte(key)); !existed {
		s.Set(key, value)
		return true
	}
	return false
}

// Get return the value stored for the given key, or nil if no value was found.
func (s *sled) Get(key string, v interface{}) error {
	val, ok := s.ct.Lookup([]byte(key))
	if !ok {
		return errors.New("key does not exist")
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return errors.New("argument must be a pointer")
	} else if rv.IsNil() {
		return errors.New("argument is nil")
	}
	rv.Elem().Set(reflect.ValueOf(val))
	return nil
}

// Delete removes a key and value, and returns it's previous value with
// an existed flag that will be true if the key was not empty.
func (s *sled) Delete(key string) (value interface{}, existed bool) {
	value, existed = s.ct.Remove([]byte(key))
	return
}

// Close releases all sled resources.
func (s *sled) Close() error {
	return nil
}

// Snapshot returns a single point in time image of the Sled.
// Snapshot is fast and non blocking.
func (s *sled) Snapshot(mode IoMode) Sled {
	return &sled{s.ct.Snapshot(mode)}
}

var elePool = sync.Pool{
	New: func() interface{} {
		return &ele{}
	},
}

// Iterator returns the key value pair for each key in the sled.
// It takes an optional cancel channel which can be closed to stop iterating.
// The key and value are returned in an 'Element' interface.
// For performance reasons, the caller must call Close() on
// the Element returned.
func (s *sled) Iterate(cancel <-chan struct{}) <-chan Element {
	out := make(chan Element)
	go func() {
		defer close(out)
		for e := range s.ct.Iterate(cancel) {
			entry := elePool.Get().(*ele)
			entry.k = string(e.Key)
			entry.v = e.Value
			entry.c = func() {
				elePool.Put(entry)
			}
			select {
			case out <- entry:
			case <-cancel:
				return
			}
		}

	}()
	return out
}
