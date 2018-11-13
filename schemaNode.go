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

// TODO improve thread safeness
func (node *schemaNode) insertTypes(types []*iType) {
	// update typ "counts" at tail
	if len(types) > 0 {
		globalNodeLocks[uintptr(unsafe.Pointer(node))%lockPrime].Lock()
		if node.Types == nil {
			node.Types = make(map[*iType]uint32)
		}
		for _, t := range types {
			node.Types[t]++
		}
		globalNodeLocks[uintptr(unsafe.Pointer(node))%lockPrime].Unlock()
	}
}

// thread-safe!
const lockPrime = 97 // arbitrary prime number
var globalNodeLocks [97]sync.RWMutex

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

	globalNodeLocks[uintptr(unsafe.Pointer(node))%lockPrime].RLock()
	// binary search for the child
	i := sort.Search(len(node.Children), func(i int) bool { return uintptr(unsafe.Pointer(node.Children[i].ID)) >= uintptr(unsafe.Pointer(term)) })
	if i < len(node.Children) && node.Children[i].ID == term {
		defer globalNodeLocks[uintptr(unsafe.Pointer(node))%lockPrime].RUnlock()
		return node.Children[i]
	}

	globalNodeLocks[uintptr(unsafe.Pointer(node))%lockPrime].RUnlock()

	// We have to add the child, aquire a write lock
	globalNodeLocks[uintptr(unsafe.Pointer(node))%lockPrime].Lock()

	// search again, since child might meanwhile have been added by other thread
	i = sort.Search(len(node.Children), func(i int) bool { return uintptr(unsafe.Pointer(node.Children[i].ID)) >= uintptr(unsafe.Pointer(term)) })
	if i < len(node.Children) && node.Children[i].ID == term {
		defer globalNodeLocks[uintptr(unsafe.Pointer(node))%lockPrime].Unlock()
		return node.Children[i]
	}

	// child not found, but i is the index where it would be inserted.
	// create a new one...
	newChild := &schemaNode{term, node, []*schemaNode{}, term.traversalPointer, 0, nil}
	term.traversalPointer = newChild

	// ...and insert it at position i
	node.Children = append(node.Children, nil)
	copy(node.Children[i+1:], node.Children[i:])
	node.Children[i] = newChild

	globalNodeLocks[uintptr(unsafe.Pointer(node))%lockPrime].Unlock()

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

	// draw types
	for k, v := range node.Types {
		s += fmt.Sprintf("%v -> \"%v\" [color=red,label=%v];\n", node, *k.Str, v)
	}

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
