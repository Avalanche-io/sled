package sled

import (
	"bytes"
	"sync/atomic"
	"unsafe"
)

// toContracted ensures that every I-node except the root points to a C-node
// with at least one branch. If a given C-Node has only a single S-node below
// it and is not at the root level, a T-node which wraps the S-node is
// returned.
func toContracted(cn *cNode, lev uint) *node {
	if lev > 0 && len(cn.array) == 1 {
		branch := cn.array[0]
		switch branch.(type) {
		case *sNode:
			return entomb(branch.(*sNode))
		default:
			return &node{cNode: cn}
		}
	}
	return &node{cNode: cn}
}

// toCompressed compacts the C-node as a performance optimization.
func toCompressed(cn *cNode, lev uint) *node {
	tmpArray := make([]branch, len(cn.array))
	for i, sub := range cn.array {
		switch sub.(type) {
		case *iNode:
			inode := sub.(*iNode)
			mainPtr := (*unsafe.Pointer)(unsafe.Pointer(&inode.main))
			main := (*node)(atomic.LoadPointer(mainPtr))
			tmpArray[i] = resurrect(inode, main)
		case *sNode:
			tmpArray[i] = sub
		default:
			panic("invalid state")
		}
	}

	return toContracted(&cNode{bmp: cn.bmp, array: tmpArray}, lev)
}

func entomb(m *sNode) *node {
	return &node{tNode: &tNode{m}}
}

func resurrect(iNode *iNode, main *node) branch {
	if main.tNode != nil {
		return main.tNode.untombed()
	}
	return iNode
}

func clean(i *iNode, lev uint, c *ctrie) bool {
	main := gcasRead(i, c)
	if main.cNode != nil {
		return gcas(i, main, toCompressed(main.cNode, lev), c)
	}
	return true
}

func cleanReadOnly(tn *tNode, lev uint, p *iNode, c *ctrie, e *entry) (val interface{}, exists bool, ok bool) {
	if !c.readOnly {
		clean(p, lev-5, c)
		return nil, false, false
	}
	if tn.hash == e.hash && bytes.Equal(tn.Key, e.Key) {
		return tn.Value, true, true
	}
	return nil, false, true
}

func cleanParent(p, i *iNode, hc uint64, lev uint, c *ctrie, startGen *generation) {
	n := aln(&i.main)
	pn := aln(&p.main)
	// var (
	// 	un  = (*unsafe.Pointer)(unsafe.Pointer(&i.main))

	// 	n     = (*node)(atomic.LoadPointer(un))
	// 	pMainPtr = (*unsafe.Pointer)(unsafe.Pointer(&p.main))
	// 	pMain    = (*node)(atomic.LoadPointer(pMainPtr))
	// )
	if pn.cNode != nil {
		flag, pos := flagPos(hc, lev, pn.cNode.bmp)
		if pn.cNode.bmp&flag != 0 {
			sub := pn.cNode.array[pos]
			if sub == i && n.tNode != nil {
				ncn := pn.cNode.updated(pos, resurrect(i, n), i.gen)
				if !gcas(p, pn, toContracted(ncn, lev), c) && c.readRoot().gen == startGen {
					cleanParent(p, i, hc, lev, c, startGen)
				}
			}
		}
	}
}
