package main

import (
	"fmt"
	"time"

	"github.com/windler/dotgraph/renderer"
)

func twoPass(fileName string) {
	// first pass: collect I-List and statistics
	c := subjectSummaryReader(fileName)
	propMap := make(propMap)
	i := 0
	for subjectSummary := range c {
		for _, propIri := range subjectSummary.properties {
			prop := propMap.get(propIri)
			prop.totalCount++
		}

		if i++; i >= 50 {
			break
		}
	}

	// for _, v := range propMap {
	// 	fmt.Println(*v)
	// }

	// second pass
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

		if i++; i >= 50 {
			r := &renderer.PNGRenderer{
				OutputFile: "my_graph.png",
			}
			r.Render(fmt.Sprint(schema))

			break
		}
	}
}

func main() {
	fileName := "latest-truthy.nt.bz2"
	t1 := time.Now()
	twoPass(fileName)
	fmt.Println(time.Since(t1))
}
