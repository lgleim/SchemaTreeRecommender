package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
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

func newSchemaNode() schemaNode {
	// return schemaNode{nil, nil, make(map[*iItem]*schemaNode), nil, 0, nil}
	root := "root"
	return schemaNode{&iItem{&root, 0, 0, nil}, nil, []*schemaNode{}, nil, 0, nil}
}

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
	i := sort.Search(len(node.Children), func(i int) bool { return uintptr(unsafe.Pointer(node.Children[i])) >= uintptr(unsafe.Pointer(term)) })
	if i < len(node.Children) && node.Children[i].ID == term {
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

	return newChild
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

type SchemaTree struct {
	propMap propMap
	typeMap typeMap
	Root    schemaNode
	MinSup  uint32
}

func (tree SchemaTree) String() string {
	s := "digraph schematree {\n"
	s += tree.Root.graphViz(tree.MinSup)
	return s + "}"
}

func (tree *SchemaTree) insert(s *subjectSummary, updateSupport bool) {
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
				item.TotalCount++
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
				item.TotalCount++
			}
		}
	}

	// sort the properties according to the current iList sort order
	properties.sort()

	// insert sorted property-list into actual schemaTree
	node := &tree.Root
	node.Support++
	for _, prop := range properties {
		node = node.getChild(prop) // recurse, i.e., node.getChild(prop).insert(properties[1:], types)
		node.Support++
	}

	// update typ "counts" at tail
	//node.types = append(node.types, types...) // TODO: make this a counting structure (map)
	if len(types) > 0 {
		if node.Types == nil {
			node.Types = make(map[*iType]uint32)
		}
		for _, t := range types {
			node.Types[t]++
		}
	}
}

func (tree *SchemaTree) reorganize() {
	tree.updateSortOrder()

	// TODO: implement actual tree reorganization
}

// update iList according to actual frequencies
// calling this directly WILL BREAK non-empty schema trees
// Runtime: O(n*log(n)), Memory: O(n)
func (tree *SchemaTree) updateSortOrder() {
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
			if (*(iList[i])).TotalCount != (*(iList[j])).TotalCount {
				return (*(iList[i])).TotalCount > (*(iList[j])).TotalCount
			}
			return *((*(iList[i])).Str) < *((*(iList[j])).Str)
		})

	// update term's internal sortOrder
	// Runtime: O(n), Memory: -
	for i, v := range iList {
		v.SortOrder = uint16(i)
	}
}

func (tree *SchemaTree) support(properties iList) uint32 {
	var support uint32

	if len(properties) == 0 {
		return tree.Root.Support // empty set occured in all transactions
	}

	properties.sort() // descending by support

	// check all branches that include least frequent term
	for term := properties[len(properties)-1].traversalPointer; term != nil; term = term.nextSameID {
		if term.prefixContains(&properties) {
			support += term.Support
		}
	}

	return support
}

// internal! propertyPath *MUST* be sorted in sortOrder (i.e. descending support)
func (node *schemaNode) prefixContains(propertyPath *iList) bool {
	nextP := len(*propertyPath) - 1                        // index of property expected to be seen next
	for cur := node; cur.parent != nil; cur = cur.parent { // walk from leaf towards root

		if cur.ID.SortOrder < (*propertyPath)[nextP].SortOrder { // we already walked past the next expected property
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

func (tree *SchemaTree) recommendProperty(properties iList) propertyRecommendations {
	var setSupport uint32
	//tree.root.support // empty set occured in all transactions

	properties.sort() // descending by support

	pSet := properties.toSet()

	candidates := make(map[*iItem]uint32)

	var makeCandidates func(startNode *schemaNode)
	makeCandidates = func(startNode *schemaNode) { // head hunter function ;)
		for _, child := range startNode.Children {
			candidates[child.ID] += child.Support
			makeCandidates(child)
		}
	}

	// walk from each leaf towards root...l
	for leaf := properties[len(properties)-1].traversalPointer; leaf != nil; leaf = leaf.nextSameID {
		if leaf.prefixContains(&properties) {
			setSupport += leaf.Support // number of occuences of this set of properties in the current branch
			for cur := leaf; cur.parent != nil; cur = cur.parent {
				if !(pSet[cur.ID]) {
					candidates[cur.ID] += leaf.Support
				}
			}
			makeCandidates(leaf)
		}
	}

	// TODO: If there are no candidates, consider doing (n-1)-gram smoothing over property subsets

	// now that all candidates have been collected, rank them
	ranked := make([]rankedPropertyCandidate, 0, len(candidates))
	for candidate, support := range candidates {
		ranked = append(ranked, rankedPropertyCandidate{candidate, float64(support) / float64(setSupport)})
	}

	// sort descending by support
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].Probability > ranked[j].Probability })

	return ranked
}

// func (tree *schemaTree) recommendType(properties iList) typeRecommendations {
// 	var setSupport uint32
// 	//tree.root.support // empty set occured in all transactions

// 	properties.sort() // descending by support

// 	pSet := properties.toSet()

// 	candidates := make(map[*iItem]uint32)

// 	var makeCandidates func(startNode *schemaNode)
// 	makeCandidates = func(startNode *schemaNode) { // head hunter function ;)
// 		for _, child := range startNode.children {
// 			candidates[child.ID] += child.support
// 			makeCandidates(child)
// 		}
// 	}

// 	// walk from each leaf towards root...l
// 	for leaf := properties[len(properties)-1].traversalPointer; leaf != nil; leaf = leaf.nextSameID {
// 		if leaf.prefixContains(&properties) {
// 			setSupport += leaf.support // number of occuences of this set of properties in the current branch
// 			for cur := leaf; cur.parent != nil; cur = cur.parent {
// 				if !(pSet[cur.ID]) {
// 					candidates[cur.ID] += leaf.support
// 				}
// 			}
// 			makeCandidates(leaf)
// 		}
// 	}

// 	// TODO: If there are no candidates, consider doing (n-1)-gram smoothing over property subsets

// 	// now that all candidates have been collected, rank them
// 	ranked := make([]rankedCandidate, 0, len(candidates))
// 	for candidate, support := range candidates {
// 		ranked = append(ranked, rankedCandidate{candidate, float64(support) / float64(setSupport)})
// 	}

// 	// sort descending by support
// 	sort.Slice(ranked, func(i, j int) bool { return ranked[i].probability > ranked[j].probability })

// 	return ranked
// }

func (tree *SchemaTree) save(filePath string) error {
	t1 := time.Now()
	fmt.Printf("Writing schema to file %v... ", filePath)

	// // Via Sereal lib since it supports serialization of object references, including circular references.
	// // See https://github.com/Sereal/Sereal
	// e := sereal.NewEncoder()
	// // e.Compression = sereal.SnappyCompressor{Incremental: true}
	// serialized, err := e.Marshal(tree)
	// err = ioutil.WriteFile(filePath, serialized, 0644)

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	err = gob.NewEncoder(f).Encode(tree)

	if err == nil {
		fmt.Printf("done (%v)\n", time.Since(t1))
	} else {
		fmt.Printf("Saving schema failed with error: %v\n", err)
	}

	return err
}

func loadSchemaTree(filePath string) (*SchemaTree, error) {
	fmt.Printf("Loading schema (from file %v): ", filePath)
	t1 := time.Now()

	// serialized, err := ioutil.ReadFile(filePath)
	f, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Encountered error while trying to open the file: %v\n", err)
		return nil, err
	}

	tree := new(SchemaTree)
	// err = sereal.Unmarshal(serialized, tree)
	err = gob.NewDecoder(f).Decode(tree)
	if err != nil {
		fmt.Printf("Encountered error while decoding the file: %v\n", err)
		return nil, err
	}

	fmt.Println(time.Since(t1))

	fmt.Printf("Restructuring schema tree: ")
	t1 = time.Now()

	// reinstantiate propMap and typeMap
	tree.propMap = make(propMap)
	tree.typeMap = make(typeMap)

	var wg sync.WaitGroup // goroutine coordination

	// fixes parent pointers & children sort order
	var parallelFix func(node *schemaNode, parent *schemaNode)
	parallelFix = func(node *schemaNode, parent *schemaNode) {
		//recurse
		wg.Add(len(node.Children))
		for _, child := range node.Children {
			go parallelFix(child, node)
		}

		// parent link reconstruction
		node.parent = parent

		// fixing sort order of schemaNode.children arrays (sorted by *changed* pointer addresses)
		sort.Slice(node.Children, func(i, j int) bool {
			return uintptr(unsafe.Pointer(node.Children[i])) < uintptr(unsafe.Pointer(node.Children[j]))
		})
		wg.Done()
	}

	// repopulates propMap and typeMap & deduplicates the corresponding iItems and iTypes
	var serialFix func(node *schemaNode)
	serialFix = func(node *schemaNode) {
		// property deduplication & traversal pointer repopulation
		if prop, ok := tree.propMap[*node.ID.Str]; ok {
			node.ID = prop
			node.nextSameID = prop.traversalPointer
			prop.traversalPointer = node
		} else {
			tree.propMap[*node.ID.Str] = node.ID
			node.nextSameID = nil
			node.ID.traversalPointer = node
		}

		// type deduplication
		for class, support := range node.Types {
			if dedupClass, ok := tree.typeMap[*class.Str]; ok {
				delete(node.Types, class)
				node.Types[dedupClass] = support
			} else {
				tree.typeMap[*class.Str] = class
			}
		}

		// recurse
		for _, child := range node.Children {
			serialFix(child)
		}
	}

	wg.Add(1)
	go parallelFix(&tree.Root, nil)
	serialFix(&tree.Root)

	wg.Wait()
	fmt.Println(time.Since(t1))

	return tree, err
}
