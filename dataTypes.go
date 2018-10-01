package main

import "fmt"

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

func (p iList) String() string {
	//// list representation (includes duplicates)
	o := "[ "
	for i := 0; i < len(p); i++ {
		o += *(p[i].str) + " "
	}
	return o + "]"

	//// couter presentation (loses order)
	// ctr := make(map[string]int)
	// for i := 0; i < len(p); i++ {
	// 	ctr[p[i].str]++
	// }
	// return fmt.Sprint(ctr)
}
