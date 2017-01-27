package sled

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/Workiva/go-datastructures/trie/ctrie"
)

// Create a new Sled object with optional custom configuration.
func New() *mem_sled {
	ct := ctrie.New(nil)
	return &mem_sled{ct}
}

// Assigns value to key, replacing any previous values.
func (s *mem_sled) Set(key string, value interface{}) error {
	s.ct.Insert([]byte(key), value)
	return nil
}

// SetNil is exclusive Set.  It only assigns the value to the key,
// if the key is not already set.  It returns true if the assignment succeed.
func (s *mem_sled) SetIfNil(key string, value interface{}) bool {
	if _, existed := s.ct.Lookup([]byte(key)); !existed {
		s.Set(key, value)
		return true
	}
	return false
}

// Get return the value stored for the given key, or nil if no value was found.
func (s *mem_sled) Get(key string, v interface{}) error {
	val, ok := s.ct.Lookup([]byte(key))
	if !ok {
		return errors.New("key does not exist")
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New(fmt.Sprintf("invalid type error %s", reflect.TypeOf(v)))
	}
	rv.Elem().Set(reflect.ValueOf(val))
	return nil
}

// Delete removes a key and value, and returns it's previous value with
// an existed flag that will be true if the key was not empty.
func (s *mem_sled) Delete(key string) (value interface{}, existed bool) {
	value, existed = s.ct.Remove([]byte(key))
	return
}

// Snapshot returns a single point in time image of the Sled.
// Snapshot is fast and non blocking.
func (s *mem_sled) Snapshot() CRUD {
	return &mem_sled{s.ct.Snapshot()}
}

var elePool = sync.Pool{
	New: func() interface{} {
		return &ele{}
	},
}

// Iterator returns the key value pair for each key in the sled.
// It takes an optional cancel channel which can be closed to stop iterating.
// The key and value are returned in an 'Element' interface.
// For performance reasons, the caller must call Close() after using
// the Element returned by Iterate
func (s *mem_sled) Iterate(cancel <-chan struct{}) <-chan Element {
	out := make(chan Element)
	c := make(chan struct{})
	go func() {
		defer close(out)
		for e := range s.ct.Iterator(c) {
			entry := elePool.Get().(*ele)
			entry.k = string(e.Key)
			entry.v = e.Value
			entry.c = func() {
				elePool.Put(entry)
			}
			select {
			case out <- entry:
			case <-cancel:
				close(c)
			}
		}

	}()
	return out
}
