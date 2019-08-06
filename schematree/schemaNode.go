package schematree

import (
	"encoding/gob"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"unsafe"
)

// SchemaNode is a nodes of the Schema FP-Tree
type SchemaNode struct {
	ID         *IItem
	parent     *SchemaNode
	Children   []*SchemaNode
	nextSameID *SchemaNode // node traversal pointer
	Support    uint32      // total frequency of the node in the path
}

//newRootNode creates a new root node for a given propMap
func newRootNode(pMap propMap) SchemaNode {
	// return schemaNode{newRootiItem(), nil, make(map[*iItem]*schemaNode), nil, 0, nil}
	return SchemaNode{pMap.get("root"), nil, []*SchemaNode{}, nil, 0}
}

//writeGob encodes the schema node into a binary representation
func (node *SchemaNode) writeGob(e *gob.Encoder) error {
	// ID
	err := e.Encode(node.ID.SortOrder)
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

	return nil
}

// decodeGob decodes the schema node from its binary representation
func (node *SchemaNode) decodeGob(d *gob.Decoder, props []*IItem) error {
	// function scoping to allow for garbage collection
	// err := func() error {
	// ID
	var id uint32
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
	node.Children = make([]*SchemaNode, length, length)
	// node.Children = make(map[*iItem]*SchemaNode)

	// 	return nil
	// }()
	for i := range node.Children {
		node.Children[i] = &SchemaNode{nil, node, nil, nil, 0}
		err = node.Children[i].decodeGob(d, props)
		// for i := 0; i < length; i++ {
		// 	child := &SchemaNode{nil, node, nil, nil, 0, nil}
		// 	err = child.decodeGob(d, props, tMap)
		if err != nil {
			return err
		}
		// node.Children[child.ID] = child
	}
	// // fixing sort order of SchemaNode.children arrays (sorted by *changed* pointer addresses)
	// sort.Slice(node.Children, func(i, j int) bool {
	// 	return uintptr(unsafe.Pointer(node.Children[i].ID)) < uintptr(unsafe.Pointer(node.Children[j].ID))
	// })

	return nil
}

//incrementSupport increments the support of the schema node by one
func (node *SchemaNode) incrementSupport() {
	atomic.AddUint32(&node.Support, 1)
}

// thread-safe!
const lockPrime = 97 // arbitrary prime number
var globalItemLocks [lockPrime]*sync.Mutex
var globalNodeLocks [lockPrime]*sync.RWMutex

// getOrCreateChild returns the child of a node associated to a IItem. If such child does not exist, a new child is created.
func (node *SchemaNode) getOrCreateChild(term *IItem) *SchemaNode {

	// binary search for the child
	globalNodeLocks[uintptr(unsafe.Pointer(node))%lockPrime].RLock()
	children := node.Children
	i := sort.Search(
		len(children),
		func(i int) bool {
			return uintptr(unsafe.Pointer(children[i].ID)) >= uintptr(unsafe.Pointer(term))
		})

	if i < len(children) {
		if child := children[i]; child.ID == term {
			globalNodeLocks[uintptr(unsafe.Pointer(node))%lockPrime].RUnlock()
			return child
		}
	}
	globalNodeLocks[uintptr(unsafe.Pointer(node))%lockPrime].RUnlock()

	// We have to add the child, aquire a lock for this term
	globalNodeLocks[uintptr(unsafe.Pointer(node))%lockPrime].Lock()

	// search again, since child might meanwhile have been added by other thread or previous search might have missed
	children = node.Children
	i = sort.Search(
		len(children),
		func(i int) bool {
			return uintptr(unsafe.Pointer(children[i].ID)) >= uintptr(unsafe.Pointer(term))
		})
	if i < len(node.Children) {
		if child := children[i]; child.ID == term {
			globalNodeLocks[uintptr(unsafe.Pointer(node))%lockPrime].Unlock()
			return child
		}
	}

	// child not found, but i is the index where it would be inserted.
	// create a new one...
	globalItemLocks[uintptr(unsafe.Pointer(term))%lockPrime].Lock()
	newChild := &SchemaNode{term, node, []*SchemaNode{}, term.traversalPointer, 0}
	term.traversalPointer = newChild
	globalItemLocks[uintptr(unsafe.Pointer(term))%lockPrime].Unlock()

	// ...and insert it at position i
	node.Children = append(node.Children, nil)
	copy(node.Children[i+1:], node.Children[i:])
	node.Children[i] = newChild

	globalNodeLocks[uintptr(unsafe.Pointer(node))%lockPrime].Unlock()

	return newChild
}

// prefixContains checks if all properties of a given list are ancestors of a node
// internal! propertyPath *MUST* be sorted in sortOrder (i.e. descending support)
// thread-safe!
func (node *SchemaNode) prefixContains(propertyPath IList) bool {
	nextP := len(propertyPath) - 1                         // index of property expected to be seen next
	for cur := node; cur.parent != nil; cur = cur.parent { // walk from leaf towards root

		if cur.ID.SortOrder < propertyPath[nextP].SortOrder { // we already walked past the next expected property
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

func (node *SchemaNode) graphViz(minSup uint32) string {
	s := ""
	// // draw horizontal links
	// if node.nextSameID != nil && node.nextSameID.Support >= minSup {
	// 	s += fmt.Sprintf("%v -> %v  [color=blue];\n", node, node.nextSameID)
	// }

	// // draw types
	// for k, v := range node.Types {
	// 	s += fmt.Sprintf("%v -> \"%v\" [color=red,label=%v];\n", node, *k.Str, v)
	// }

	// draw children
	for _, child := range node.Children {
		if child.Support >= minSup {
			s += fmt.Sprintf("\"%p\" -> \"%p\" [label=%v;weight=%v];\n", node, child, child.Support, child.ID.TotalCount)
			s += child.graphViz(minSup)
		}
	}

	return s
}

// func (node *SchemaNode) String() string {
// 	return fmt.Sprintf("\"%v (%p id:%p str:%p)\"", *node.ID.Str, node, node.ID, node.ID.Str)
// }
