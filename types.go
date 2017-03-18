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

type Tx interface {
	Element
	Action() string
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

// Sled holds pointers to the configuration, database, and the ctrie
// data structure.  It has no exported data structures.
type sled struct {
	// cfg               *config.Config
	ct *ctrie
}
