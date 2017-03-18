package sled

import (
	"sync/atomic"
	"unsafe"
)

// rdcssDescriptor is an intermediate struct which communicates the intent to
// replace the value in an I-node and check that the root's generation has not
// changed before committing to the new value.
type rdcssDescriptor struct {
	old       *iNode
	expected  *node
	nv        *iNode
	committed int32
}

func newRdcssDescriptor(o *iNode, m *node, n *iNode) *rdcssDescriptor {
	return &rdcssDescriptor{
		old:      o,
		expected: m,
		nv:       n,
	}
}

func (d *rdcssDescriptor) Load(i int32) bool {
	return atomic.LoadInt32(&d.committed) == i
}

func (d *rdcssDescriptor) Set(i int32) {
	atomic.StoreInt32(&d.committed, i)
}

// // casRoot performs a CAS on the Ctrie root.
// func casRoot(c *Ctrie, ov, nv *iNode) bool {
// 	return atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&c.root)), unsafe.Pointer(ov), unsafe.Pointer(nv))
// }

// func casRoot(r unsafe.Pointer, ov, nv *iNode) bool {
func casRoot(r **iNode, ov, nv *iNode) bool {
	return atomic.CompareAndSwapPointer(upp(unsafe.Pointer(r)), unsafe.Pointer(ov), unsafe.Pointer(nv))
}

func casINode(r **iNode, ov, nv *iNode) bool {
	return atomic.CompareAndSwapPointer(upp(unsafe.Pointer(r)), unsafe.Pointer(ov), unsafe.Pointer(nv))
}

func casMainNode(r **node, ov, nv *node) bool {
	return atomic.CompareAndSwapPointer(upp(unsafe.Pointer(r)), unsafe.Pointer(ov), unsafe.Pointer(nv))
}

func upp(p unsafe.Pointer) *unsafe.Pointer {
	return (*unsafe.Pointer)(unsafe.Pointer(p))
}

// atomic local inode
func alin(root **iNode) *iNode {
	return (*iNode)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(root))))
}

// atomic load node
func aln(main **node) *node {
	return (*node)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(main))))
}

// gcasRead performs a GCAS-linearizable read of the I-node's main node.
func gcasRead(in *iNode, c *ctrie) *node {
	// m := (*node)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&in.main))))
	// prev := (*node)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&m.prev))))
	m := aln(&in.main)
	prev := aln(&m.prev)
	if prev == nil {
		return m
	}
	return gcasComplete(in, m, c.rdcssCompleteAbort())
}

func gcasComplete(i *iNode, m *node, root *iNode) *node {
	for prev := aln(&m.prev); m != nil && prev != nil; prev = aln(&m.prev) {
		switch {
		// Signals GCAS failure. Swap old value back into I-node.
		case prev.failed != nil:
			f := aln(&prev.failed)
			if casMainNode(&i.main, m, f) {
				return f
			}
			m = aln(&i.main)
		case root.gen == i.gen: // Commit GCAS.
			if casMainNode(&m.prev, prev, nil) {
				return m
			}
		default:
			// Generations did not match. Store failed node on prev to signal
			// I-node's main node must be set back to the previous value.
			casMainNode(&m.prev, prev, &node{failed: prev})
			m = aln(&i.main)
		}
	}
	return m //nil
}

// gcas is a generation-compare-and-swap which has semantics similar to RDCSS,
// but it does not create the intermediate object except in the case of
// failures that occur due to the snapshot being taken. This ensures that the
// write occurs only if the Ctrie root generation has remained the same in
// addition to the I-node having the expected value.
func gcas(in *iNode, old, n *node, c *ctrie) bool {
	prevPtr := (*unsafe.Pointer)(unsafe.Pointer(&n.prev))
	atomic.StorePointer(prevPtr, unsafe.Pointer(old))
	if atomic.CompareAndSwapPointer(
		(*unsafe.Pointer)(unsafe.Pointer(&in.main)),
		unsafe.Pointer(old), unsafe.Pointer(n)) {
		gcasComplete(in, n, c.rdcssCompleteAbort())
		return atomic.LoadPointer(prevPtr) == nil
	}
	return false
}
