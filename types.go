package sled

// Element is interface for accessing keys and values when iterating. For effectuate memory
// utilization elements are implemented as a pool of re-usable structures and therefore
// must be closed after use.
type Element interface {
	Key() string
	Value() interface{}
	Close()
}

// IoMode represents the mode of a sled as either ReadWrite, or ReadOnly.
type IoMode uint

const (
	ReadOnly IoMode = iota
	ReadWrite
)
