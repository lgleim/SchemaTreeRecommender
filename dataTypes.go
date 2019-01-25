package schematree

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
)

// Class statistics
// - a string IRI (str) and
// - its support, i.e. its total number of occurrences (totalCount)
type iType struct {
	Str        *string
	TotalCount uint64
}

func (t *iType) increment() {
	atomic.AddUint64(&t.TotalCount, 1)
}

type typeMap map[string]*iType

var typeMapLock sync.Mutex

// thread-safe
func (m *typeMap) get(iri string) (item *iType) {
	item, ok := (*m)[iri]
	if !ok {
		typeMapLock.Lock()
		defer typeMapLock.Unlock()

		// recheck existence - might have been created by other thread
		if item, ok = (*m)[iri]; ok {
			return
		}

		item = &iType{&iri, 0}
		(*m)[iri] = item
	}
	return
}

// A struct capturing
// - a string IRI (str) and
// - its support, i.e. its total number of occurrences (totalCount)
// - an integer indicating sort order
type IItem struct {
	Str              *string
	TotalCount       uint64
	sortOrder        uint32
	traversalPointer *schemaNode // node traversal pointer
}

func (p *IItem) increment() {
	atomic.AddUint64(&p.TotalCount, 1)
}

func (p IItem) String() string {
	return fmt.Sprint(p.TotalCount, "x\t", *p.Str, " (", p.sortOrder, ")")
}

type propMap map[string]*IItem

var propMapLock sync.Mutex

// thread-safe
func (m propMap) get(iri string) (item *IItem) { // TODO: Implement sameas Mapping/Resolution to single group identifier upon insert!
	item, ok := m[iri]
	if !ok {
		propMapLock.Lock()
		defer propMapLock.Unlock()

		// recheck existence - might have been created by other thread
		if item, ok = m[iri]; ok {
			return
		}

		item = &IItem{&iri, 0, uint32(len(m)), nil}
		m[iri] = item
	}
	return
}

// An array of pointers to IRI structs
type IList []*IItem

// Sort the list according to the current iList Sort order
func (l IList) Sort() {
	sort.Slice(l, func(i, j int) bool { return l[i].sortOrder < l[j].sortOrder })
}

// inplace sorting and deduplication.
func (l *IList) sortAndDeduplicate() {
	ls := *l

	ls.Sort()

	// inplace deduplication
	j := 0
	for i := 1; i < len(ls); i++ {
		if ls[j] == ls[i] {
			continue
		}
		j++
		ls[i], ls[j] = ls[j], ls[i]
	}
	*l = ls[:j+1]
}

func (l IList) toSet() map[*IItem]bool {
	pSet := make(map[*IItem]bool, len(l))
	for _, p := range l {
		pSet[p] = true
	}
	return pSet
}

func (l IList) String() string {
	//// list representation (includes duplicates)
	o := "[ "
	for i := 0; i < len(l); i++ {
		o += *(l[i].Str) + " "
	}
	return o + "]"

	//// couter presentation (loses order)
	// ctr := make(map[string]int)
	// for i := 0; i < len(p); i++ {
	// 	ctr[p[i].str]++
	// }
	// return fmt.Sprint(ctr)
}

// struct to rank suggestions
type rankedPropertyCandidate struct {
	Property    *IItem
	Probability float64
}

type propertyRecommendations []rankedPropertyCandidate

func (ps propertyRecommendations) String() string {
	s := ""
	for _, p := range ps {
		s += fmt.Sprintf("%v: %v\n", *p.Property.Str, p.Probability)
	}
	return s
}

type rankedTypeCandidate struct {
	Class       *iType
	Probability float64
}
type typeRecommendations []rankedTypeCandidate

func (ts typeRecommendations) String() string {
	s := ""
	for _, t := range ts {
		s += fmt.Sprintf("%v: %v\n", *t.Class.Str, t.Probability)
	}
	return s
}
