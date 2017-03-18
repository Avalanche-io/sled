package sled

import (
	"bytes"
	"sync/atomic"
	"unsafe"
)

// node is either a cNode, tNode, lNode, or failed node which makes up an
// I-node.
type node struct {
	cNode  *cNode
	tNode  *tNode
	lNode  *lNode
	failed *node

	// prev is set as a failed main node when we attempt to CAS and the
	// I-node's generation does not match the root generation. This signals
	// that the GCAS failed and the I-node's main node must be set back to the
	// previous value.
	prev *node
}

// newNode is a recursive constructor which creates a new node. This
// node will consist of cNodes as long as the hashcode chunks of the two
// keys are equal at the given level. If the level exceeds 2^w, an lNode is
// created.
func newNode(x *sNode, xhc uint64, y *sNode, yhc uint64, lev uint, gen *generation) *node {
	if lev < exp2 {
		xidx := (xhc >> lev) & 0x3f
		yidx := (yhc >> lev) & 0x3f
		bmp := uint64((1 << xidx) | (1 << yidx))

		if xidx == yidx {
			// Recurse when indexes are equal.
			main := newNode(x, xhc, y, yhc, lev+w, gen)
			iNode := &iNode{main: main, gen: gen}
			return &node{cNode: &cNode{bmp, []branch{iNode}, gen}}
		}
		if xidx < yidx {
			return &node{cNode: &cNode{bmp, []branch{x, y}, gen}}
		}
		return &node{cNode: &cNode{bmp, []branch{y, x}, gen}}
	}
	l := emptyList.Add(x).Add(y)
	return &node{lNode: &lNode{l}}
}

// iNode is an indirection node. I-nodes remain present in the Ctrie even as
// nodes above and below change. Thread-safety is achieved in part by
// performing CAS operations on the I-node instead of the internal node array.
type iNode struct {
	main *node
	gen  *generation

	// rdcss is set during an RDCSS operation. The I-node is actually a wrapper
	// around the descriptor in this case so that a single type is used during
	// CAS operations on the root.
	rdcss *rdcssDescriptor
}

// copyToGen returns a copy of this I-node copied to the given generation.
func (i *iNode) copyToGen(gen *generation, c *ctrie) *iNode {
	nin := &iNode{gen: gen}
	main := gcasRead(i, c)
	atomic.StorePointer(
		(*unsafe.Pointer)(unsafe.Pointer(&nin.main)), unsafe.Pointer(main))
	return nin
}

// cNode is an internal main node containing a bitmap and the array with
// references to branch nodes. A branch node is either another I-node or a
// singleton S-node.
type cNode struct {
	bmp   uint64
	array []branch
	gen   *generation
}

// inserted returns a copy of this cNode with the new entry at the given
// position.
func (c *cNode) inserted(pos, flag uint64, br branch, gen *generation) *cNode {
	length := uint64(len(c.array))
	bmp := c.bmp
	array := make([]branch, length+1)
	copy(array, c.array)
	array[pos] = br
	for i, x := pos, uint64(0); x < length-pos; i++ {
		array[i+1] = c.array[i]
		x++
	}
	ncn := &cNode{bmp: bmp | flag, array: array, gen: gen}
	return ncn
}

// updated returns a copy of this cNode with the entry at the given index
// updated.
func (c *cNode) updated(pos uint64, br branch, gen *generation) *cNode {
	array := make([]branch, len(c.array))
	copy(array, c.array)
	array[pos] = br
	ncn := &cNode{bmp: c.bmp, array: array, gen: gen}
	return ncn
}

// removed returns a copy of this cNode with the entry at the given index
// removed.
func (c *cNode) removed(pos, flag uint64, gen *generation) *cNode {
	length := uint64(len(c.array))
	bmp := c.bmp
	array := make([]branch, length-1)
	for i := uint64(0); i < pos; i++ {
		array[i] = c.array[i]
	}
	for i, x := pos, uint64(0); x < length-pos-1; i++ {
		array[i] = c.array[i+1]
		x++
	}
	ncn := &cNode{bmp: bmp ^ flag, array: array, gen: gen}
	return ncn
}

func (n *cNode) renewif(gen *generation, c *ctrie) *cNode {
	if n.gen != gen {
		return n.renewed(gen, c)
	}
	return n
}

// renewed returns a copy of this cNode with the I-nodes below it copied to the
// given generation.
func (n *cNode) renewed(gen *generation, c *ctrie) *cNode {
	array := make([]branch, len(n.array))
	for i, br := range n.array {
		switch t := br.(type) {
		case *iNode:
			array[i] = t.copyToGen(gen, c)
		default:
			array[i] = br
		}
	}
	return &cNode{bmp: n.bmp, array: array, gen: gen}
}

// tNode is tomb node which is a special node used to ensure proper ordering
// during removals.
type tNode struct {
	*sNode
}

// untombed returns the S-node contained by the T-node.
func (t *tNode) untombed() *sNode {
	return &sNode{&entry{Key: t.Key, hash: t.hash, Value: t.Value}}
}

// lNode is a list node which is a leaf node used to handle hashcode
// collisions by keeping such keys in a persistent list.
type lNode struct {
	lister
}

// entry returns the first S-node contained in the L-node.
func (l *lNode) entry() *sNode {
	head, _ := l.Head()
	return head.(*sNode)
}

// lookup returns the value at the given entry in the L-node or returns false
// if it's not contained.
func (l *lNode) lookup(e *entry) (interface{}, bool) {
	found, ok := l.Find(func(sn interface{}) bool {
		return bytes.Equal(e.Key, sn.(*sNode).Key)
	})
	if !ok {
		return nil, false
	}
	return found.(*sNode).Value, true
}

// inserted creates a new L-node with the added entry.
func (l *lNode) inserted(e *entry) *lNode {
	return &lNode{l.Add(&sNode{e})}
}

// removed creates a new L-node with the entry removed.
func (l *lNode) removed(e *entry) *lNode {
	idx := l.FindIndex(func(sn interface{}) bool {
		return bytes.Equal(e.Key, sn.(*sNode).Key)
	})
	if idx < 0 {
		return l
	}
	nl, _ := l.Remove(uint(idx))
	return &lNode{nl}
}

// length returns the L-node list length.
func (l *lNode) length() uint {
	return l.Length()
}

// sNode is a singleton node which contains a single key and value.
type sNode struct {
	*entry
}
