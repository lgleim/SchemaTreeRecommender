package main

import (
	"fmt"
	"time"
)

func twoPass(fileName string, firstN uint64) {
	// first pass: collect I-List and statistics
	t1 := time.Now()
	c := subjectSummaryReader(fileName)
	propMap := make(propMap)
	var i uint64

	for subjectSummary := range c {
		for _, propIri := range subjectSummary.properties {
			prop := propMap.get(propIri)
			prop.totalCount++
		}

		if i++; firstN > 0 && i >= firstN {
			fmt.Println(subjectSummary)
			break
		}
	}

	fmt.Println("First Pass:", time.Since(t1))

	// second pass
	t1 = time.Now()
	c = subjectSummaryReader(fileName)
	schema := schemaTree{
		propMap: propMap,
		typeMap: make(typeMap),
		root:    schemaNode{nil, nil, []*schemaNode{}, nil, 0, nil},
		minSup:  3,
	}

	schema.updateSortOrder()

	i = 0
	for subjectSummary := range c {
		schema.insert(subjectSummary, false)

		if i++; firstN > 0 && i >= firstN {
			fmt.Println(subjectSummary)
			break
		}
	}

	fmt.Println("Second Pass:", time.Since(t1))

	// r := &renderer.PNGRenderer{
	// 	OutputFile: "my_graph.png",
	// }
	// r.Render(fmt.Sprint(schema))

	rdftype := propMap["http://www.w3.org/1999/02/22-rdf-syntax-ns#type"]
	memberOf := propMap["http://www.wikidata.org/prop/direct/P463"]
	list := []*iItem{rdftype, memberOf}
	fmt.Println(schema.support(list), schema.root.support)

	t1 = time.Now()
	rec := schema.recommend(list)
	fmt.Println(time.Since(t1))
	fmt.Println(rec[:10])
}

func main() {
	// fileName := "latest-truthy.nt.bz2"
	fileName := "100k.nt"
	t1 := time.Now()
	twoPass(fileName, 388)
	fmt.Println(time.Since(t1))
}
