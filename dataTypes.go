package main

import (
	"fmt"
	"sort"
)

// A struct capturing
// - a string IRI (str) and
// - its support, i.e. its total number of occurrences (totalCount)
// - an integer indicating sort order
type iType struct {
	str        *string
	totalCount uint32
}

type typeMap map[string]*iType

func (m *typeMap) get(iri *string) *iType {
	item, ok := (*m)[*iri]
	if !ok {
		item = &iType{iri, 0}
		(*m)[*iri] = item
	}
	return item
}

// A struct capturing
// - a string IRI (str) and
// - its support, i.e. its total number of occurrences (totalCount)
// - an integer indicating sort order
type iItem struct {
	str              *string
	totalCount       uint32
	sortOrder        uint16
	traversalPointer *schemaNode // node traversal pointer
}

func (m iItem) String() string {
	return fmt.Sprint(m.totalCount, "x\t", *m.str, " (", m.sortOrder, ")")
}

type propMap map[string]*iItem

func (m *propMap) get(iri *string) *iItem { // TODO: Implement sameas Mapping/Resolution to single group identifier upon insert!
	item, ok := (*m)[*iri]
	if !ok {
		item = &iItem{iri, 0, uint16(len(*m)), nil}
		(*m)[*iri] = item
	}
	return item
}

// An array of pointers to IRI structs
type iList []*iItem

// sort the list according to the current iList sort order
func (l *iList) sort() {
	// sort the properties according to the current iList sort order
	sort.Slice(*l, func(i, j int) bool { return (*l)[i].sortOrder < (*l)[j].sortOrder })
}

func (l *iList) toSet() map[*iItem]bool {
	pSet := make(map[*iItem]bool, len(*l))
	for _, p := range *l {
		pSet[p] = true
	}
	return pSet
}

func (l iList) String() string {
	//// list representation (includes duplicates)
	o := "[ "
	for i := 0; i < len(l); i++ {
		o += *(l[i].str) + " "
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
	property    *iItem
	probability float64
}

type propertyRecommendations []rankedPropertyCandidate

func (ps propertyRecommendations) String() string {
	s := ""
	for _, p := range ps {
		s += fmt.Sprintf("%v: %v\n", *p.property.str, p.probability)
	}
	return s
}

type rankedTypeCandidate struct {
	class       *iType
	probability float64
}
type typeRecommendations []rankedTypeCandidate

func (ts typeRecommendations) String() string {
	s := ""
	for _, t := range ts {
		s += fmt.Sprintf("%v: %v\n", *t.class.str, t.probability)
	}
	return s
}
