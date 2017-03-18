package sled

import (
	"bytes"
	"sync/atomic"
	"unsafe"
)

func (c *ctrie) traverse(i *iNode, ch chan<- *entry, cancel <-chan struct{}) error {
	main := gcasRead(i, c)
	switch {
	case main.cNode != nil:
		for _, br := range main.cNode.array {
			switch b := br.(type) {
			case *iNode:
				if err := c.traverse(b, ch, cancel); err != nil {
					return err
				}
			case *sNode:
				select {
				case ch <- b.entry:
				case <-cancel:
					return ErrCanceled{}
				}
			}
		}
	case main.lNode != nil:
		for _, e := range main.lNode.Map(func(sn interface{}) interface{} {
			return sn.(*sNode).entry
		}) {
			select {
			case ch <- e.(*entry):
			case <-cancel:
				return ErrCanceled{}
			}
		}
	}
	return nil
}

func (c *ctrie) assertReadWrite() {
	if c.readOnly {
		panic("Cannot modify read-only snapshot")
	}
}

func (c *ctrie) insert(entry *entry) {
	root := c.readRoot()
	if !c.iinsert(root, entry, 0, nil, root.gen) {
		c.insert(entry)
	}
}

func (c *ctrie) lookup(entry *entry) (interface{}, bool) {
	root := c.readRoot()
	result, exists, ok := c.ilookup(root, entry, 0, nil, root.gen)
	for !ok {
		return c.lookup(entry)
	}
	return result, exists
}

func (c *ctrie) remove(entry *entry) (interface{}, bool) {
	root := c.readRoot()
	result, exists, ok := c.iremove(root, entry, 0, nil, root.gen)
	for !ok {
		return c.remove(entry)
	}
	return result, exists
}

func (c *ctrie) hash(k []byte) uint64 {
	hasher := c.hashFactory()
	hasher.Write(k)
	return hasher.Sum64()
}

// If the relevant bit is not in the bitmap, then a copy of the
// cNode with the new entry is created. The linearization point is
// a successful CAS.
func (c *ctrie) nobit(cn *cNode, gen *generation, entry *entry, lev uint) (uint64, *node) {
	flag, pos := flagPos(entry.hash, lev, cn.bmp)
	if cn.bmp&flag != 0 {
		return pos, nil
	}
	rn := cn.renewif(gen, c)
	return pos, &node{cNode: rn.inserted(pos, flag, &sNode{entry}, gen)}
}

func (c *ctrie) branchinode(main *node, in *iNode, i *iNode, entry *entry, lev uint, parent *iNode, startGen *generation) bool {
	// If the branch is an I-node, then iinsert is called recursively.
	if startGen == in.gen {
		return c.iinsert(in, entry, lev+w, i, startGen)
	}
	if gcas(i, main, &node{cNode: main.cNode.renewed(startGen, c)}, c) {
		return c.iinsert(i, entry, lev, parent, startGen)
	}
	return false
}

func (c *ctrie) branchsnode(main *node, sn *sNode, i *iNode, entry *entry, lev uint, pos uint64) bool {
	if bytes.Equal(sn.Key, entry.Key) {
		// If the key in the S-node is equal to the key being inserted,
		// then the C-node is replaced with its updated version with a new
		// S-node. The linearization point is a successful CAS.
		ncn := &node{cNode: main.cNode.updated(pos, &sNode{entry}, i.gen)}
		return gcas(i, main, ncn, c)
	}
	// If the branch is an S-node and its key is not equal to the
	// key being inserted, then the Ctrie has to be extended with
	// an additional level. The C-node is replaced with its updated
	// version, created using the updated function that adds a new
	// I-node at the respective position. The new Inode has its
	// main node pointing to a C-node with both keys. The
	// linearization point is a successful CAS.
	rn := main.cNode.renewif(i.gen, c)
	nsn := &sNode{entry}
	nin := &iNode{main: newNode(sn, sn.hash, nsn, nsn.hash, lev+w, i.gen), gen: i.gen}
	ncn := &node{cNode: rn.updated(pos, nin, i.gen)}
	return gcas(i, main, ncn, c)
}

func (c *ctrie) cinsert(main *node, i *iNode, entry *entry, lev uint, parent *iNode, startGen *generation) bool {
	var ncn *node
	var pos uint64
	if pos, ncn = c.nobit(main.cNode, i.gen, entry, lev); ncn != nil {
		return gcas(i, main, ncn, c)
	}
	// If the relevant bit is present in the bitmap, then its corresponding
	// branch is read from the array.
	branch := main.cNode.array[pos]
	switch n := branch.(type) {
	case *iNode:
		return c.branchinode(main, n, i, entry, lev, parent, startGen)
	case *sNode:
		return c.branchsnode(main, n, i, entry, lev, pos)
	default:
		panic("Ctrie is in an invalid state")
	}
}

// iinsert attempts to insert the entry into the Ctrie. If false is returned,
// the operation should be retried.
func (c *ctrie) iinsert(i *iNode, entry *entry, lev uint, parent *iNode, startGen *generation) bool {
	// Linearization point.
	main := gcasRead(i, c)
	switch {
	case main.cNode != nil:
		return c.cinsert(main, i, entry, lev, parent, startGen)
	case main.tNode != nil:
		clean(parent, lev-w, c)
	case main.lNode != nil:
		return gcas(i, main, &node{lNode: main.lNode.inserted(entry)}, c)
	default:
		panic("Ctrie is in an invalid state")
	}
	return false
}

// ilookup attempts to fetch the entry from the Ctrie. The first two return
// values are the entry value and whether or not the entry was contained in the
// Ctrie. The last bool indicates if the operation succeeded. False means it
// should be retried.
func (c *ctrie) ilookup(i *iNode, entry *entry, lev uint, parent *iNode, startGen *generation) (interface{}, bool, bool) {
	// Linearization point.
	main := gcasRead(i, c)
	switch {
	case main.cNode != nil:
		cn := main.cNode
		flag, pos := flagPos(entry.hash, lev, cn.bmp)
		if cn.bmp&flag == 0 {
			// If the bitmap does not contain the relevant bit, a key with the
			// required hashcode prefix is not present in the trie.
			return nil, false, true
		}
		// Otherwise, the relevant branch at index pos is read from the array.
		branch := cn.array[pos]
		switch branch.(type) {
		case *iNode:
			// If the branch is an I-node, the ilookup procedure is called
			// recursively at the next level.
			in := branch.(*iNode)
			if c.readOnly || startGen == in.gen {
				return c.ilookup(in, entry, lev+w, i, startGen)
			}
			if gcas(i, main, &node{cNode: cn.renewed(startGen, c)}, c) {
				return c.ilookup(i, entry, lev, parent, startGen)
			}
			return nil, false, false
		case *sNode:
			// If the branch is an S-node, then the key within the S-node is
			// compared with the key being searched – these two keys have the
			// same hashcode prefixes, but they need not be equal. If they are
			// equal, the corresponding value from the S-node is
			// returned and a NOTFOUND value otherwise.
			sn := branch.(*sNode)
			if bytes.Equal(sn.Key, entry.Key) {
				return sn.Value, true, true
			}
			return nil, false, true
		default:
			panic("Ctrie is in an invalid state")
		}
	case main.tNode != nil:
		return cleanReadOnly(main.tNode, lev, parent, c, entry)
	case main.lNode != nil:
		// Hash collisions are handled using L-nodes, which are essentially
		// persistent linked lists.
		val, ok := main.lNode.lookup(entry)
		return val, ok, true
	default:
		panic("Ctrie is in an invalid state")
	}
}

// iremove attempts to remove the entry from the Ctrie. The first two return
// values are the entry value and whether or not the entry was contained in the
// Ctrie. The last bool indicates if the operation succeeded. False means it
// should be retried.
func (c *ctrie) iremove(i *iNode, entry *entry, lev uint, parent *iNode, startGen *generation) (interface{}, bool, bool) {
	// Linearization point.
	main := gcasRead(i, c)
	switch {
	case main.cNode != nil:
		cn := main.cNode
		flag, pos := flagPos(entry.hash, lev, cn.bmp)
		if cn.bmp&flag == 0 {
			// If the bitmap does not contain the relevant bit, a key with the
			// required hashcode prefix is not present in the trie.
			return nil, false, true
		}
		// Otherwise, the relevant branch at index pos is read from the array.
		branch := cn.array[pos]
		switch branch.(type) {
		case *iNode:
			// If the branch is an I-node, the iremove procedure is called
			// recursively at the next level.
			in := branch.(*iNode)
			if startGen == in.gen {
				return c.iremove(in, entry, lev+w, i, startGen)
			}
			if gcas(i, main, &node{cNode: cn.renewed(startGen, c)}, c) {
				return c.iremove(i, entry, lev, parent, startGen)
			}
			return nil, false, false
		case *sNode:
			// If the branch is an S-node, its key is compared against the key
			// being removed.
			sn := branch.(*sNode)
			if !bytes.Equal(sn.Key, entry.Key) {
				// If the keys are not equal, the NOTFOUND value is returned.
				return nil, false, true
			}
			//  If the keys are equal, a copy of the current node without the
			//  S-node is created. The contraction of the copy is then created
			//  using the toContracted procedure. A successful CAS will
			//  substitute the old C-node with the copied C-node, thus removing
			//  the S-node with the given key from the trie – this is the
			//  linearization point
			ncn := cn.removed(pos, flag, i.gen)
			cntr := toContracted(ncn, lev)
			if gcas(i, main, cntr, c) {
				if parent != nil {
					main = gcasRead(i, c)
					if main.tNode != nil {
						cleanParent(parent, i, entry.hash, lev-w, c, startGen)
					}
				}
				return sn.Value, true, true
			}
			return nil, false, false
		default:
			panic("Ctrie is in an invalid state")
		}
	case main.tNode != nil:
		clean(parent, lev-w, c)
		return nil, false, false
	case main.lNode != nil:
		nln := &node{lNode: main.lNode.removed(entry)}
		if nln.lNode.length() == 1 {
			nln = entomb(nln.lNode.entry())
		}
		if gcas(i, main, nln, c) {
			val, ok := main.lNode.lookup(entry)
			return val, ok, true
		}
		return nil, false, true
	default:
		panic("Ctrie is in an invalid state")
	}
}

// cas/rdcss methods should be move the rdcss file

// readRoot performs a linearizable read of the Ctrie root. This operation is
// prioritized so that if another thread performs a GCAS on the root, a
// deadlock does not occur.
func (c *ctrie) readRoot() *iNode {
	r := alin(&c.root)
	if r.rdcss != nil {
		return c.rdcssComplete()
	}
	return r
}

// rdcssRoot performs a RDCSS on the Ctrie root. This is used to create a
// snapshot of the Ctrie by copying the root I-node and setting it to a new
// generation.
func (c *ctrie) rdcssRoot(old *iNode, expected *node, nv *iNode) bool {
	desc := &iNode{rdcss: newRdcssDescriptor(old, expected, nv)}
	if c.casRoot(old, desc) {
		c.rdcssComplete()
		// return atomic.LoadInt32(&desc.rdcss.committed) == 1
		return desc.rdcss.Load(1)
	}
	return false
}

// rdcssComplete commits the RDCSS operation.
func (c *ctrie) rdcssComplete() *iNode {
	for {
		r := (*iNode)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&c.root))))
		if r.rdcss == nil {
			return r
		}

		var (
			desc = r.rdcss
			ov   = desc.old
			exp  = desc.expected
			nv   = desc.nv
		)

		oldeMain := gcasRead(ov, c)
		if oldeMain == exp {
			// Commit the RDCSS.
			if c.casRoot(r, nv) {
				atomic.StoreInt32(&desc.committed, 1)
				return nv
			}
			continue
		}
		if c.casRoot(r, ov) {
			return ov
		}
		continue
	}
}

func (c *ctrie) rdcssCompleteAbort() *iNode {
	for {

		r := (*iNode)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&c.root))))
		if r.rdcss == nil {
			return r
		}

		if c.casRoot(r, r.rdcss.old) {
			return r.rdcss.old
		}

	}
}

// casRoot performs a CAS on the Ctrie root.
func (c *ctrie) casRoot(ov, nv *iNode) bool {
	c.assertReadWrite()
	return casRoot(&c.root, ov, nv)
}
