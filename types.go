package sled

// A key/value pair interface, for use in range operations and signals.
type Element interface {
	Key() string
	Value() interface{}
	Close()
}

type IoMode uint

const (
	ReadOnly IoMode = iota
	ReadWrite
)
