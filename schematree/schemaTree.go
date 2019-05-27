package schematree

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"time"
	"unsafe"

	gzip "github.com/klauspost/pgzip"
)

type SchemaTree struct {
	PropMap propMap
	TypeMap typeMap
	Root    schemaNode
	MinSup  uint32
}

// NewSchemaTree returns a newly allocated and initialized schema tree
func NewSchemaTree() (tree *SchemaTree) {
	pMap := make(propMap)
	tree = &SchemaTree{
		PropMap: pMap,
		TypeMap: make(typeMap),
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

// WritePropFreqs writes all Properties together with their Support to the given File as CSV
func (tree SchemaTree) WritePropFreqs(file string) {
	f, err := os.Create(file)
	if err != nil {
		log.Fatalln("Could not open file to writePropFreqs!")
	}
	defer f.Close()

	f.WriteString("URI;Frequency\n")
	for uri, item := range tree.PropMap {
		f.WriteString(fmt.Sprintf("%v;%v\n", uri, item.TotalCount))
	}
}

func (tree SchemaTree) String() string {
	var minSupport uint32 = 100000
	s := "digraph schematree { newrank=true; labelloc=b; color=blue; fontcolor=blue; style=dotted;\n"

	s += tree.Root.graphViz(minSupport)

	cluster := ""

	for _, prop := range tree.PropMap {
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
func (tree *SchemaTree) Insert(s *SubjectSummary) {
	// properties := s.properties
	// sort the properties according to the current iList sort order & deduplicate
	// properties.sortAndDeduplicate()

	properties := make(IList, len(s.Properties), len(s.Properties))
	i := 0
	for p := range s.Properties {
		properties[i] = p
		i++
	}
	properties.Sort()

	// fmt.Println(properties)

	// insert sorted property-list into actual schemaTree
	node := &tree.Root
	node.incrementSupport()
	for _, prop := range properties {
		node = node.getChild(prop) // recurse, i.e., node.getChild(prop).insert(properties[1:], types)
		node.incrementSupport()
	}

	// update class "counts" at tail
	node.insertTypes(s.Types)
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
	iList := make(IList, len(tree.PropMap))
	i := 0
	for _, v := range tree.PropMap {
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
		v.SortOrder = uint32(i)
	}
}

// Support returns the total cooccurrence-frequency of the given property list
func (tree *SchemaTree) Support(properties IList) uint32 {
	var support uint32

	if len(properties) == 0 {
		return tree.Root.Support // empty set occured in all transactions
	}

	properties.Sort() // descending by support

	// check all branches that include least frequent term
	for term := properties[len(properties)-1].traversalPointer; term != nil; term = term.nextSameID {
		if term.prefixContains(properties) {
			support += term.Support
		}
	}

	return support
}

func (tree *SchemaTree) RecommendProperty(properties IList) (ranked propertyRecommendations) {

	if len(properties) > 0 {

		properties.Sort() // descending by support

		pSet := properties.toSet()

		candidates := make(map[*IItem]uint32)

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
		ranked = make([]RankedPropertyCandidate, len(candidates), len(candidates))
		for candidate, support := range candidates {
			ranked[i] = RankedPropertyCandidate{candidate, float64(support) / setSup}
			i++
		}

		// sort descending by support
		sort.Slice(ranked, func(i, j int) bool { return ranked[i].Probability > ranked[j].Probability })
	} else {
		// TODO: Race condition on propMap: fatal error: concurrent map iteration and map write
		// fmt.Println(tree.Root.Support)
		setSup := float64(tree.Root.Support) // empty set occured in all transactions
		ranked = make([]RankedPropertyCandidate, len(tree.PropMap), len(tree.PropMap))
		for _, prop := range tree.PropMap {
			ranked[int(prop.SortOrder)] = RankedPropertyCandidate{prop, float64(prop.TotalCount) / setSup}
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
	props := make([]*IItem, len(tree.PropMap), len(tree.PropMap))
	for _, p := range tree.PropMap {
		props[int(p.SortOrder)] = p
	}
	err = e.Encode(props)
	if err != nil {
		return err
	}

	// encode typeMap
	types := make(map[uintptr]*iType, len(tree.TypeMap))
	for _, t := range tree.TypeMap {
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
	var props []*IItem
	err = d.Decode(&props)
	if err != nil {
		return nil, err
	}
	for sortOrder, item := range props {
		item.SortOrder = uint32(sortOrder)
		tree.PropMap[*item.Str] = item
	}
	fmt.Printf("%v properties... ", len(props))

	// decode typeMap
	var types map[uintptr]*iType
	err = d.Decode(&types)
	if err != nil {
		return nil, err
	}
	for _, t := range types {
		tree.TypeMap[*t.Str] = t
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
		tree.Root.ID = tree.PropMap.get("root")
	}

	if err != nil {
		fmt.Printf("Encountered error while decoding the file: %v\n", err)
		return nil, err
	}

	fmt.Println(time.Since(t1))
	return tree, err
}

// first pass: collect I-List and statistics
func (tree *SchemaTree) firstPass(fileName string, firstN uint64) {
	if _, err := os.Stat(fileName + ".firstPass.bin"); os.IsNotExist(err) {
		counter := func(s *SubjectSummary) {
			for prop := range s.Properties {
				prop.increment()
			}
		}

		t1 := time.Now()
		subjectCount := SubjectSummaryReader(fileName, tree.PropMap, tree.TypeMap, counter, firstN)

		fmt.Printf("%v subjects, %v properties, %v types\n", subjectCount, len(tree.PropMap), len(tree.TypeMap))

		// f, _ := os.Create(fileName + ".propMap")
		// gob.NewEncoder(f).Encode(schema.propMap)
		// f.Close()
		// f, _ = os.Create(fileName + ".typeMap")
		// gob.NewEncoder(f).Encode(schema.typeMap)
		// f.Close()

		tree.updateSortOrder()

		fmt.Println("First Pass:", time.Since(t1))
		PrintMemUsage()

		const MaxUint32 = uint64(^uint32(0))
		if subjectCount > MaxUint32 {
			fmt.Print("\n#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#\n\n")
			fmt.Printf("WARNING: uint32 OVERFLOW - Processed %v subjects but tree can only track support up to %v!\n", subjectCount, MaxUint32)
			fmt.Print("\n#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#\n\n")
		}

		err = tree.Save(fileName + ".firstPass.bin")
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		// f1, err1 := os.Open(fileName + ".propMap")
		// f2, err2 := os.Open(fileName + ".typeMap")

		// if err1 == nil && err2 == nil {
		// 	fmt.Print("Loading type- and propertyMap directly from corresponding gobs: ")
		// 	tmp := NewSchemaTree()
		// 	gob.NewDecoder(f1).Decode(&tmp.propMap)
		// 	gob.NewDecoder(f2).Decode(&tmp.typeMap)
		// 	tmp.updateSortOrder()
		// 	*schema = *tmp
		// 	fmt.Printf("%v properties, %v types\n", len(tmp.propMap), len(tmp.typeMap))
		// } else {
		tmp, err := LoadSchemaTree(fileName + ".firstPass.bin")
		if err != nil {
			log.Fatalln(err)
		}
		*tree = *tmp
		// }
	}
}

// build schema tree
func (tree *SchemaTree) secondPass(fileName string, firstN uint64) {
	tree.updateSortOrder() // duplicate -- legacy compatability

	inserter := func(s *SubjectSummary) {
		tree.Insert(s)
	}

	// go countTreeNodes(schema)

	t1 := time.Now()
	SubjectSummaryReader(fileName, tree.PropMap, tree.TypeMap, inserter, firstN)

	fmt.Println("Second Pass:", time.Since(t1))
	PrintMemUsage()
	// PrintLockStats()
}

// TwoPass constructs a SchemaTree from the firstN subjects of the given NTriples file using a two-pass approach
func (tree *SchemaTree) TwoPass(fileName string, firstN uint64) {
	// go func() {
	// 	for true {
	// 		time.Sleep(10 * time.Second)
	// 		PrintMemUsage()
	// 	}
	// }()
	tree.firstPass(fileName, firstN)
	tree.secondPass(fileName, firstN)
}
