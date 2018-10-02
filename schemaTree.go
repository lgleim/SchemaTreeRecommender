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
	properties.sort()

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

func (tree *schemaTree) support(properties iList) uint32 {
	var support uint32

	if len(properties) == 0 {
		return tree.root.support // empty set occured in all transactions
	}

	properties.sort() // descending by support

	// check all branches that include least frequent term
	for term := properties[len(properties)-1].traversalPointer; term != nil; term = term.nextSameID {
		if term.prefixContains(&properties) {
			support += term.support
		}
	}

	return support
}

// internal! propertyPath *MUST* be sorted in sortOrder (i.e. descending support)
func (node *schemaNode) prefixContains(propertyPath *iList) bool {
	nextP := len(*propertyPath) - 1                        // index of property expected to be seen next
	for cur := node; cur.parent != nil; cur = cur.parent { // walk from leaf towards root

		if cur.ID.sortOrder < (*propertyPath)[nextP].sortOrder { // we already walked past the next expected property
			return false
		}
		if cur.ID == (*propertyPath)[nextP] {
			nextP--
			if nextP < 0 { // we encountered all expected properties!
				return true
			}
		}
	}
	return false
}

func (tree *schemaTree) recommend(properties iList) propertyRecommendations {
	var setSupport uint32
	//tree.root.support // empty set occured in all transactions

	properties.sort() // descending by support

	pSet := make(map[*iItem]bool)
	for _, p := range properties {
		pSet[p] = true
	}

	candidates := make(map[*iItem]uint32)

	var makeCandidates func(startNode *schemaNode)
	makeCandidates = func(startNode *schemaNode) { // head hunter function ;)
		for _, child := range startNode.children {
			candidates[child.ID] += child.support
			makeCandidates(child)
		}
	}

	// walk from each leaf towards root...
	for leaf := properties[len(properties)-1].traversalPointer; leaf != nil; leaf = leaf.nextSameID {
		if leaf.prefixContains(&properties) {
			setSupport += leaf.support // number of occuences of this set of properties in the current branch
			for cur := leaf; cur.parent != nil; cur = cur.parent {
				if !(pSet[cur.ID]) {
					candidates[cur.ID] += leaf.support
				}
			}
			makeCandidates(leaf)
		}
	}

	// TODO: If there are no candidates, consider doing (n-1)-gram smoothing over property subsets

	// now that all candidates have been collected, rank them
	ranked := make([]rankedCandidate, 0, len(candidates))
	for candidate, support := range candidates {
		ranked = append(ranked, rankedCandidate{candidate, float64(support) / float64(setSupport)})
	}

	// sort descending by support
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].probability > ranked[j].probability })

	return ranked
}
