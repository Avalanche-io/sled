package sled

// Sled is an interface for sled key value store types.
type Sled interface {
	Set(key string, v interface{}) error
	SetIfNil(string, interface{}) bool
	Get(key string, v interface{}) error
	Delete(string) (interface{}, bool)
	Close() error
	Iterate(<-chan struct{}) <-chan Element
	Snapshot(IoMode) Sled
}
