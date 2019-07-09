package schematree

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// A struct capturing
// - a string IRI (str) and
// - its support, i.e. its total number of occurrences (totalCount)
// - an integer indicating sort order
type IItem struct {
	Str              *string
	TotalCount       uint64
	SortOrder        uint32
	traversalPointer *SchemaNode // node traversal pointer
}

func (p *IItem) increment() {
	atomic.AddUint64(&p.TotalCount, 1)
}

var typePrefix = "t#"

func (p *IItem) IsType() bool {
	return strings.HasPrefix(*p.Str, typePrefix)
}

func (p *IItem) IsProp() bool {
	return !strings.HasPrefix(*p.Str, typePrefix)
}

func (p IItem) String() string {
	return fmt.Sprint(p.TotalCount, "x\t", *p.Str, " (", p.SortOrder, ")")
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

func (p propMap) count() (int, int) {
	props := 0
	types := 0
	for _, item := range p {
		if item.IsType() {
			types++
		} else {
			props++
		}
	}
	return props, types
}

// An array of pointers to IRI structs
type IList []*IItem

// Sort the list according to the current iList Sort order
func (l IList) Sort() {
	sort.Slice(l, func(i, j int) bool { return l[i].SortOrder < l[j].SortOrder })
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

func (l IList) count() (int, int) {
	props := 0
	types := 0
	for _, item := range l {
		if item.IsType() {
			types++
		} else {
			props++
		}
	}
	return props, types
}

func (l IList) removeTypes() IList {
	nl := IList{}
	for _, item := range l {
		if !item.IsType() {
			nl = append(nl, item)
		}
	}
	return nl
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
