package main

import (
	"encoding/gob"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Nodes of the Schema FP-Tree
// TODO: determine wether to use hash-maps or arrays to store child notes in the tree
type schemaNode struct {
	ID     *iItem
	parent *schemaNode
	// Children   map[*iItem]*schemaNode
	Children   []*schemaNode
	nextSameID *schemaNode       // node traversal pointer
	Support    uint32            // total frequency of the node in the path
	Types      map[*iType]uint32 //[]*iType    // RDFS class - nonempty only for tail nodes
}

func newRootNode() schemaNode {
	root := "root"
	// return schemaNode{&iItem{&root, 0, 0, nil}, nil, make(map[*iItem]*schemaNode), nil, 0, nil}
	return schemaNode{&iItem{&root, 0, 0, nil}, nil, []*schemaNode{}, nil, 0, nil}
}

func (node *schemaNode) writeGob(e *gob.Encoder) error {
	// ID
	err := e.Encode(node.ID.sortOrder)
	if err != nil {
		return err
	}

	// Support
	err = e.Encode(node.Support)
	if err != nil {
		return err
	}

	// Children
	err = e.Encode(len(node.Children))
	if err != nil {
		return err
	}
	for _, child := range node.Children {
		err = child.writeGob(e)
		if err != nil {
			return err
		}
	}

	// Types
	types := make(map[uintptr]uint32)
	for t, count := range node.Types {
		types[uintptr(unsafe.Pointer(t))] = count
	}
	err = e.Encode(types)
	return err
}

func (node *schemaNode) decodeGob(d *gob.Decoder, props []*iItem, tMap map[uintptr]*iType) error {
	// function scoping to allow for garbage collection
	err := func() error {
		// ID
		var id uint16
		err := d.Decode(&id)
		if err != nil {
			return err
		}
		node.ID = props[int(id)]

		// traversal pointer repopulation
		node.nextSameID = node.ID.traversalPointer
		node.ID.traversalPointer = node

		// Support
		err = d.Decode(&node.Support)
		if err != nil {
			return err
		}

		// Children
		var length int
		err = d.Decode(&length)
		if err != nil {
			return err
		}
		node.Children = make([]*schemaNode, length, length)

		return nil
	}()
	for i := range node.Children {
		node.Children[i] = &schemaNode{nil, node, nil, nil, 0, nil}
		err = node.Children[i].decodeGob(d, props, tMap)
		if err != nil {
			return err
		}
		// node.Children[i].parent = node
	}
	// fixing sort order of schemaNode.children arrays (sorted by *changed* pointer addresses)
	sort.Slice(node.Children, func(i, j int) bool {
		return uintptr(unsafe.Pointer(node.Children[i].ID)) < uintptr(unsafe.Pointer(node.Children[j].ID))
	})

	// Types
	var types map[uintptr]uint32
	err = d.Decode(&types)
	if err != nil {
		return err
	}

	if len(types) > 0 {
		node.Types = make(map[*iType]uint32)
		for t, count := range types {
			node.Types[tMap[t]] = count
		}
	}

	return nil
}

func (node *schemaNode) incrementSupport() {
	atomic.AddUint32(&node.Support, 1)
}

/// structures & logic for handling types annotations in schemaNodes
var typeChan chan struct {
	node  *schemaNode
	types []*iType
}

func (node *schemaNode) insertTypes(types []*iType) {
	typeChan <- struct {
		node  *schemaNode
		types []*iType
	}{node, types}
}

func typeInsertionWorker() {
	for ts := range typeChan {
		// update typ "counts" at tail
		if len(ts.types) > 0 {
			m := ts.node.Types
			if m == nil {
				ts.node.Types = make(map[*iType]uint32)
				m = ts.node.Types
			}
			for _, t := range ts.types {
				m[t]++
			}
		}
	}
}

// thread-safe!
const lockPrime = 97 // arbitrary prime number
var globalItemLocks [lockPrime]*sync.Mutex

// const workerCount = 97 //arbitrary prime number, e.g. 43, 97
// var workers [workerCount]chan *uint32

func (node *schemaNode) getChild(term *iItem) *schemaNode {
	//// hash map based
	// child, ok := node.children[term]
	// if !ok {
	// 	// child not found. create new one:
	// 	child = &schemaNode{term, node, make(map[*iItem]*schemaNode), term.traversalPointer, 0, nil}
	// 	term.traversalPointer = child
	// 	node.children[term] = child
	// }
	// return child

	// binary search for the child
	// NOTE: Dirty reads are acceptable!!! If search misses due to dirty writes, the correct result will still be determined in the synchronized section
	children := node.Children
	i := sort.Search(len(children), func(i int) bool { return uintptr(unsafe.Pointer(children[i].ID)) >= uintptr(unsafe.Pointer(term)) })
	if i < len(children) {
		if child := children[i]; child.ID == term { // MUST be in a temporary variable, since i-th element might change concurrently
			return child
		}
	}

	// We have to add the child, aquire a lock for this term
	globalItemLocks[uintptr(unsafe.Pointer(term))%lockPrime].Lock()

	// search again, since child might meanwhile have been added by other thread or previous search might have missed
	children = node.Children
	i = sort.Search(len(children), func(i int) bool { return uintptr(unsafe.Pointer(children[i].ID)) >= uintptr(unsafe.Pointer(term)) })
	if i < len(children) && children[i].ID == term {
		defer globalItemLocks[uintptr(unsafe.Pointer(term))%lockPrime].Unlock()
		return children[i]
	}

	// child not found, but i is the index where it would be inserted.
	// create a new one...
	newChild := &schemaNode{term, node, []*schemaNode{}, term.traversalPointer, 0, nil}
	term.traversalPointer = newChild

	// ...and insert it at position i
	node.Children = append(node.Children, nil)
	copy(node.Children[i+1:], node.Children[i:])
	node.Children[i] = newChild

	globalItemLocks[uintptr(unsafe.Pointer(term))%lockPrime].Unlock()

	return newChild
}

// internal! propertyPath *MUST* be sorted in sortOrder (i.e. descending support)
// thread-safe!
func (node *schemaNode) prefixContains(propertyPath iList) bool {
	nextP := len(propertyPath) - 1                         // index of property expected to be seen next
	for cur := node; cur.parent != nil; cur = cur.parent { // walk from leaf towards root

		if cur.ID.sortOrder < propertyPath[nextP].sortOrder { // we already walked past the next expected property
			return false
		}
		if cur.ID == propertyPath[nextP] {
			nextP--
			if nextP < 0 { // we encountered all expected properties!
				return true
			}
		}
	}
	return false
}

func (node *schemaNode) graphViz(minSup uint32) string {
	s := ""
	// draw horizontal links
	if node.nextSameID != nil && node.nextSameID.Support >= minSup {
		s += fmt.Sprintf("%v -> %v  [color=blue];\n", node, node.nextSameID)
	}

	// // draw types
	// for k, v := range node.Types {
	// 	s += fmt.Sprintf("%v -> \"%v\" [color=red,label=%v];\n", node, *k.Str, v)
	// }

	// draw children
	for _, child := range node.Children {
		if child.Support >= minSup {
			s += fmt.Sprintf("%v -> %v [label=%v,weight=%v]; ", node, child, child.Support, child.Support)
			s += child.graphViz(minSup)
		}
	}

	return s
}

func (node *schemaNode) String() string {
	return fmt.Sprintf("\"%v (%p)\"", *node.ID.Str, node)
}
