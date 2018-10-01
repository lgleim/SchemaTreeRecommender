package main

import (
	"fmt"
	"sort"
)

// Nodes of the Schema FP-Tree
type schemaNode struct {
	ID         *iItem
	parent     *schemaNode
	children   []*schemaNode
	nextSameID *schemaNode       // node traversal pointer
	support    uint32            // total frequency of the node in the path
	types      map[*iType]uint32 //[]*iType    // RDFS class - nonempty only for tail nodes
}

func (node *schemaNode) graphViz(minSup uint32) string {
	s := ""
	// draw horizontal links
	if node.nextSameID != nil && node.nextSameID.support >= minSup {
		s += fmt.Sprintf("%v -> %v  [color=blue];\n", node, node.nextSameID)
	}

	// draw types
	for k, v := range node.types {
		s += fmt.Sprintf("%v -> \"%v\" [color=red,label=%v];\n", node, *k.str, v)
	}

	// draw children
	for _, child := range node.children {
		if child.support >= minSup {
			s += fmt.Sprintf("%v -> %v [label=%v,weight=%v]; ", node, child, child.support, child.support)
			s += child.graphViz(minSup)
		}
	}

	return s
}

func (node *schemaNode) String() string {
	if node.ID == nil {
		return "root"
	}
	return fmt.Sprintf("\"%v (%p)\"", *node.ID.str, node)
}

func (node *schemaNode) getChild(term *iItem) *schemaNode {
	// theoretically runtime complexity could be improved by using binary search on sorted child array. Limited by Go's lack of pointer arithmetic. Sort on e.g. child id lookups likely slower then trivial linear search (via pointer equivalence)
	for _, child := range node.children {
		if child.ID == term {
			return child
		}
	}
	// child not found. create new one:
	newChild := &schemaNode{term, node, []*schemaNode{}, term.traversalPointer, 0, nil}
	term.traversalPointer = newChild
	node.children = append(node.children, newChild)
	// TODO: Maintain nextSameID pointers
	return newChild
}

type schemaTree struct {
	propMap propMap
	typeMap typeMap
	root    schemaNode
	minSup  uint32
}

func (tree schemaTree) String() string {
	s := "digraph schematree {\n"
	s += tree.root.graphViz(tree.minSup)
	return s + "}"
}

func (tree *schemaTree) insert(s *subjectSummary, updateSupport bool) {
	// map list of types to corresponding set of iType items
	types := make([]*iType, 0, len(s.types))
	for _, typeIri := range s.types {
		item := tree.typeMap.get(typeIri)

		alreadyInserted := false
		for _, e := range types { // TODO: Ineffizient/ Unnötig?
			if e == item {
				alreadyInserted = true
				break
			}
		}
		if !alreadyInserted {
			types = append(types, item)
			if updateSupport {
				item.totalCount++
			}
		}
	}

	// map properties to corresponding iList items
	properties := make(iList, 0, len(s.properties))
	for _, propIri := range s.properties {
		item := tree.propMap.get(propIri)

		alreadyInserted := false
		for _, e := range properties { // TODO: Ineffizient
			if e == item {
				alreadyInserted = true
				break
			}
		}
		if !alreadyInserted {
			properties = append(properties, item)
			if updateSupport {
				item.totalCount++
			}
		}
	}

	// sort the properties according to the current iList sort order
	sort.Slice(properties, func(i, j int) bool { return properties[i].sortOrder < properties[j].sortOrder })

	// insert sorted property-list into actual schemaTree
	node := &tree.root
	node.support++
	for _, prop := range properties {
		node = node.getChild(prop) // recurse, i.e., node.getChild(prop).insert(properties[1:], types)
		node.support++
	}

	// update typ "counts" at tail
	//node.types = append(node.types, types...) // TODO: make this a counting structure (map)
	if len(types) > 0 {
		if node.types == nil {
			node.types = make(map[*iType]uint32)
		}
		for _, t := range types {
			node.types[t]++
		}
	}
}

func (tree *schemaTree) reorganize() {
	tree.updateSortOrder()

	// TODO: implement actual tree reorganization
}

// update iList according to actual frequencies
// calling this directly WILL BREAK non-empty schema trees
// Runtime: O(n*log(n)), Memory: O(n)
func (tree *schemaTree) updateSortOrder() {
	// make a list of all known properties
	// Runtime: O(n), Memory: O(n)
	iList := make(iList, len(tree.propMap))
	i := 0
	for _, v := range tree.propMap { // ignore key iri!
		iList[i] = v
		i++
	}

	// sort by descending support. In case of equal support, lexicographically
	// Runtime: O(n*log(n)), Memory: -
	sort.Slice(
		iList,
		func(i, j int) bool {
			if (*(iList[i])).totalCount != (*(iList[j])).totalCount {
				return (*(iList[i])).totalCount > (*(iList[j])).totalCount
			}
			return *((*(iList[i])).str) < *((*(iList[j])).str)
		})

	// update term's internal sortOrder
	// Runtime: O(n), Memory: -
	for i, v := range iList {
		v.sortOrder = uint16(i)
	}
}
