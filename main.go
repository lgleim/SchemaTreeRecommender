package main

import (
	"fmt"
	"time"
)

func twoPass(fileName string, firstN uint64) *SchemaTree {
	// first pass: collect I-List and statistics
	t1 := time.Now()
	PrintMemUsage()
	c := subjectSummaryReader(fileName)
	propMap := make(propMap)
	var i uint64

	for subjectSummary := range c {
		for _, propIri := range subjectSummary.properties {
			prop := propMap.get(propIri)
			prop.TotalCount++
		}

		if i++; firstN > 0 && i >= firstN {
			break
		}
	}

	fmt.Println("First Pass:", time.Since(t1))
	PrintMemUsage()

	// second pass
	t1 = time.Now()
	c = subjectSummaryReader(fileName)
	schema := SchemaTree{
		propMap: propMap,
		typeMap: make(typeMap),
		Root:    newSchemaNode(),
		MinSup:  3,
	}

	schema.updateSortOrder()

	i = 0
	for subjectSummary := range c {
		schema.insert(subjectSummary, false)

		if i++; firstN > 0 && i >= firstN {
			break
		}
	}

	fmt.Println("Second Pass:", time.Since(t1))
	PrintMemUsage()

	return &schema
}

func main() {
	// fileName := "latest-truthy.nt.bz2"
	fileName := "100k.nt"
	t1 := time.Now()
	schema := twoPass(fileName, 388)

	// r := &renderer.PNGRenderer{
	// 	OutputFile: "my_graph.png",
	// }
	// r.Render(fmt.Sprint(schema))

	rdftype := schema.propMap["http://www.w3.org/1999/02/22-rdf-syntax-ns#type"]
	memberOf := schema.propMap["http://www.wikidata.org/prop/direct/P463"]
	list := []*iItem{rdftype, memberOf}
	fmt.Println(schema.support(list), schema.Root.Support)

	t1 = time.Now()
	rec := schema.recommendProperty(list)
	fmt.Println(time.Since(t1))

	PrintMemUsage()
	fmt.Println(rec[:10])

	schema.save("schemaTree.bin")
	schema, _ = loadSchemaTree("schemaTree.bin")
	rec = schema.recommendProperty(list)

	PrintMemUsage()
	fmt.Println(rec[:10])

}
