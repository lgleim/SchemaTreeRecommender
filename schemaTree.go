package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
	"unsafe"

	gzip "github.com/klauspost/pgzip"
)

type SchemaTree struct {
	propMap propMap
	typeMap typeMap
	Root    schemaNode
	MinSup  uint32
}

// NewSchemaTree returns a newly allocated and initialized schema tree
func NewSchemaTree() (tree *SchemaTree) {
	pMap := make(propMap)
	tree = &SchemaTree{
		propMap: pMap,
		typeMap: make(typeMap),
		Root:    newRootNode(pMap),
		MinSup:  3,
	}
	tree.init()
	return
}

// Init initializes the datastructure for usage
func (tree *SchemaTree) init() {
	for i := range globalItemLocks {
		globalItemLocks[i] = &sync.Mutex{}
		globalNodeLocks[i] = &sync.RWMutex{}
	}
	// // initialize support counter workers
	// for i := range workers {
	// 	if workers[i] == nil {
	// 		workers[i] = make(chan *uint32) // TODO: buffering likely break transactional consistency of schema tree
	// 		go supportCounter(workers[i])   // dispatch worker coroutine
	// 	}
	// }

	if typeChan == nil {
		typeChan = make(chan struct {
			node  *schemaNode
			types []*iType
		})
		go typeInsertionWorker()
	}
}

func (tree *SchemaTree) destroy() {
	if typeChan != nil {
		close(typeChan)
	}

	// // destroy support counter workers
	// for _, wrkr := range workers {
	// 	if wrkr != nil {
	// 		close(wrkr)
	// 	}
	// }
}

func (tree SchemaTree) String() string {
	var minSupport uint32 = 100000
	s := "digraph schematree { newrank=true; labelloc=b; color=blue; fontcolor=blue; style=dotted;\n"

	s += tree.Root.graphViz(minSupport)

	cluster := ""

	for _, prop := range tree.propMap {
		cluster = ""
		for node := prop.traversalPointer; node != nil; node = node.nextSameID {
			if node.Support >= minSupport {
				cluster += fmt.Sprintf("\"%p\"; ", node)
			}
		}
		if cluster != "" {
			s += fmt.Sprintf("subgraph \"cluster_%v\" { rank=same; label=\"%v\"; %v}\n", prop.Str, *prop.Str, cluster)
		}
	}

	s += "\n"

	return s + "}"
}

// thread-safe
func (tree *SchemaTree) Insert(s *subjectSummary) {
	properties := s.properties

	// sort the properties according to the current iList sort order & deduplicate
	properties.sortAndDeduplicate()

	// fmt.Println(properties)

	// insert sorted property-list into actual schemaTree
	node := &tree.Root
	node.incrementSupport()
	for _, prop := range properties {
		node = node.getChild(prop) // recurse, i.e., node.getChild(prop).insert(properties[1:], types)
		node.incrementSupport()
	}

	// update class "counts" at tail
	node.insertTypes(s.types)
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
	for _, v := range tree.propMap {
		iList[i] = v
		i++
	}

	// sort by descending support. In case of equal support, lexicographically
	// Runtime: O(n*log(n)), Memory: -
	sort.Slice(
		iList,
		func(i, j int) bool {
			if iList[i].TotalCount != iList[j].TotalCount {
				return iList[i].TotalCount > iList[j].TotalCount
			}
			return *(iList[i].Str) < *(iList[j].Str)
		})

	// update term's internal sortOrder
	// Runtime: O(n), Memory: -
	for i, v := range iList {
		v.sortOrder = uint32(i)
	}
}

// Support returns the total cooccurrence-frequency of the given property list
func (tree *SchemaTree) Support(properties iList) uint32 {
	var support uint32

	if len(properties) == 0 {
		return tree.Root.Support // empty set occured in all transactions
	}

	properties.sort() // descending by support

	// check all branches that include least frequent term
	for term := properties[len(properties)-1].traversalPointer; term != nil; term = term.nextSameID {
		if term.prefixContains(properties) {
			support += term.Support
		}
	}

	return support
}

func (tree *SchemaTree) recommendProperty(properties iList) (ranked propertyRecommendations) {

	if len(properties) > 0 {

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

		// the least frequent property from the list is farthest from the root
		rarestProperty := properties[len(properties)-1]

		var setSupport uint64
		// walk from each "leaf" instance of that property towards the root...
		for leaf := rarestProperty.traversalPointer; leaf != nil; leaf = leaf.nextSameID { // iterate all instances for that property
			if leaf.prefixContains(properties) {
				setSupport += uint64(leaf.Support) // number of occuences of this set of properties in the current branch

				// walk up
				for cur := leaf; cur.parent != nil; cur = cur.parent {
					if !(pSet[cur.ID]) {
						candidates[cur.ID] += leaf.Support
					}
				}
				// walk down
				makeCandidates(leaf)
			}
		}

		// TODO: If there are no candidates, consider doing (n-1)-gram smoothing over property subsets

		// now that all candidates have been collected, rank them
		i := 0
		setSup := float64(setSupport)
		ranked = make([]rankedPropertyCandidate, len(candidates), len(candidates))
		for candidate, support := range candidates {
			ranked[i] = rankedPropertyCandidate{candidate, float64(support) / setSup}
			i++
		}

		// sort descending by support
		sort.Slice(ranked, func(i, j int) bool { return ranked[i].Probability > ranked[j].Probability })
	} else {
		fmt.Println(tree.Root.Support)
		setSup := float64(tree.Root.Support) // empty set occured in all transactions
		ranked = make([]rankedPropertyCandidate, len(tree.propMap), len(tree.propMap))
		for _, prop := range tree.propMap {
			ranked[prop.sortOrder] = rankedPropertyCandidate{prop, float64(prop.TotalCount) / setSup}
		}
	}

	return
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

// Save stores a binarized version of the schematree to the given filepath
func (tree *SchemaTree) Save(filePath string) error {
	t1 := time.Now()
	fmt.Printf("Writing schema to file %v... ", filePath)

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := gzip.NewWriter(f)
	defer w.Close()

	e := gob.NewEncoder(w)

	// encode propMap
	props := make([]*iItem, len(tree.propMap), len(tree.propMap))
	for _, p := range tree.propMap {
		props[p.sortOrder] = p
	}
	err = e.Encode(props)
	if err != nil {
		return err
	}

	// encode typeMap
	types := make(map[uintptr]*iType, len(tree.typeMap))
	for _, t := range tree.typeMap {
		types[uintptr(unsafe.Pointer(t))] = t
	}
	err = e.Encode(types)
	if err != nil {
		return err
	}

	// encode MinSup
	err = e.Encode(tree.MinSup)
	if err != nil {
		return err
	}

	// encode root
	err = tree.Root.writeGob(e)

	if err == nil {
		fmt.Printf("done (%v)\n", time.Since(t1))
	} else {
		fmt.Printf("Saving schema failed with error: %v\n", err)
	}

	return err
}

// LoadSchemaTree loads a binarized SchemaTree from disk
func LoadSchemaTree(filePath string) (*SchemaTree, error) {
	// Alternatively via GobDecoder(...): https://stackoverflow.com/a/12854659

	fmt.Printf("Loading schema (from file %v): ", filePath)
	t1 := time.Now()

	/// file handling
	f, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Encountered error while trying to open the file: %v\n", err)
		return nil, err
	}

	r, err := gzip.NewReader(f)
	if err != nil {
		fmt.Printf("Encountered error while trying to decompress the file: %v\n", err)
		return nil, err
	}
	defer r.Close()

	/// decoding
	tree := NewSchemaTree()
	d := gob.NewDecoder(r)

	// decode propMap
	var props []*iItem
	err = d.Decode(&props)
	if err != nil {
		return nil, err
	}
	for sortOrder, item := range props {
		item.sortOrder = uint32(sortOrder)
		tree.propMap[*item.Str] = item
	}
	fmt.Printf("%v properties... ", len(props))

	// decode typeMap
	var types map[uintptr]*iType
	err = d.Decode(&types)
	if err != nil {
		return nil, err
	}
	for _, t := range types {
		tree.typeMap[*t.Str] = t
	}
	fmt.Printf("%v types... ", len(types))

	// decode MinSup
	err = d.Decode(&tree.MinSup)
	if err != nil {
		return nil, err
	}

	// decode Root
	fmt.Printf("decoding tree...")
	err = tree.Root.decodeGob(d, props, types)

	// legacy import bug workaround
	if *tree.Root.ID.Str != "root" {
		fmt.Println("WARNING!!! Encountered legacy root node import bug - root node counts will be incorrect!")
		tree.Root.ID = tree.propMap.get("root")
	}

	if err != nil {
		fmt.Printf("Encountered error while decoding the file: %v\n", err)
		return nil, err
	}

	fmt.Println(time.Since(t1))
	return tree, err
}
