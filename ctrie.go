package sled

import (
	"hash"
	"hash/fnv"
)

// HashFactory returns a new Hash64 used to hash keys.
type HashFactory func() hash.Hash64

func defaultHashFactory() hash.Hash64 {
	return fnv.New64a()
}

// Ctrie is a concurrent, lock-free hash trie. By default, keys are hashed
// using FNV-1a unless a HashFactory is provided to New.
type ctrie struct {
	root        *iNode
	readOnly    bool
	hashFactory HashFactory
}

// generation demarcates Ctrie snapshots. We use a heap-allocated reference
// instead of an integer to avoid integer overflows. Struct must have a field
// on it since two distinct zero-size variables may have the same address in
// memory.
type generation struct{ _ int }

// branch is either an iNode or sNode.
type branch interface{}

// Entry contains a Ctrie key-value pair.
type entry struct {
	Key   []byte
	Value interface{}
	hash  uint64
}

// New creates an empty Ctrie which uses the provided HashFactory for key
// hashing. If nil is passed in, it will default to FNV-1a hashing.
func newCtrie(hashFactory HashFactory) *ctrie {
	if hashFactory == nil {
		hashFactory = defaultHashFactory
	}
	root := &iNode{main: &node{cNode: &cNode{}}}
	return makectrie(root, hashFactory, false)
}

func makectrie(root *iNode, hashFactory HashFactory, readOnly bool) *ctrie {
	return &ctrie{
		root:        root,
		hashFactory: hashFactory,
		readOnly:    readOnly,
	}
}

// Insert adds the key-value pair to the Ctrie, replacing the existing value if
// the key already exists.
func (c *ctrie) Insert(key []byte, value interface{}) {
	c.assertReadWrite()
	c.insert(&entry{
		Key:   key,
		Value: value,
		hash:  c.hash(key),
	})
}

// Lookup returns the value for the associated key or returns false if the key
// doesn't exist.
func (c *ctrie) Lookup(key []byte) (interface{}, bool) {
	return c.lookup(&entry{Key: key, hash: c.hash(key)})
}

// Remove deletes the value for the associated key, returning true if it was
// removed or false if the entry doesn't exist.
func (c *ctrie) Remove(key []byte) (interface{}, bool) {
	c.assertReadWrite()
	return c.remove(&entry{Key: key, hash: c.hash(key)})
}

// Snapshot returns a stable, point-in-time snapshot of the Ctrie.
func (c *ctrie) Snapshot(mode IoMode) *ctrie {
	if mode != ReadOnly {
		for {
			root := c.readRoot()
			main := gcasRead(root, c)
			if c.rdcssRoot(root, main, root.copyToGen(&generation{}, c)) {
				return makectrie(c.readRoot().copyToGen(&generation{}, c), c.hashFactory, c.readOnly)
			}
		}
	}
	if c.readOnly {
		return c
	}
	for {
		root := c.readRoot()
		main := gcasRead(root, c)
		if c.rdcssRoot(root, main, root.copyToGen(&generation{}, c)) {
			return makectrie(c.readRoot(), c.hashFactory, true)
		}
	}

}

// ReadOnlySnapshot returns a stable, point-in-time snapshot of the Ctrie which
// is read-only. Write operations on a read-only snapshot will panic.
// func (c *ctrie) ReadOnlySnapshot() *ctrie {
// 	if c.readOnly {
// 		return c
// 	}
// 	for {
// 		root := c.readRoot()
// 		main := gcasRead(root, c)
// 		if c.rdcssRoot(root, main, root.copyToGen(&generation{}, c)) {
// 			return newCtrie(c.readRoot(), c.hashFactory, true)
// 		}
// 	}
// }

// Clear removes all keys from the Ctrie.
func (c *ctrie) Clear() {
	for {
		root := c.readRoot()
		gen := &generation{}
		newRoot := &iNode{
			main: &node{cNode: &cNode{array: make([]branch, 0), gen: gen}},
			gen:  gen,
		}
		if c.rdcssRoot(root, gcasRead(root, c), newRoot) {
			return
		}
	}
}

// Iterate returns a channel which yields the entries of the ctrie. If a
// cancel channel is provided, closing it will terminate and close the iterator
// channel. Note that if a cancel channel is not used and not every entry is
// read from the iterator, a goroutine will leak.
func (c *ctrie) Iterate(cancel <-chan struct{}) <-chan *entry {
	ch := make(chan *entry)
	snapshot := c.Snapshot(ReadOnly)
	go func() {
		snapshot.traverse(snapshot.readRoot(), ch, cancel)
		close(ch)
	}()
	return ch
}

// Size returns the number of keys in the Ctrie.
func (c *ctrie) Size() uint {
	// TODO: The size operation can be optimized further by caching the size
	// information in main nodes of a read-only Ctrie – this reduces the
	// amortized complexity of the size operation to O(1) because the size
	// computation is amortized across the update operations that occurred
	// since the last snapshot.
	size := uint(0)
	for _ = range c.Iterate(nil) {
		size++
	}
	return size
}
