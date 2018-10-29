package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
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
		Root:    newRootNode(),
		MinSup:  3,
	}

	schema.updateSortOrder()

	i = 0
	for subjectSummary := range c {
		schema.Insert(subjectSummary, false)

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
	fileName := flag.String("file", "100k.nt", "the file to parse")
	firstNsubjects := uint64(*flag.Int64("n", 0, "Only parse the first n subjects")) // TODO: handle negative inputs
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")

	// parse commandline arguments/flags
	flag.Parse()

	// write cpu profile to file
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	t1 := time.Now()
	schema := twoPass(*fileName, firstNsubjects)

	// r := &renderer.PNGRenderer{
	// 	OutputFile: "my_graph.png",
	// }
	// r.Render(fmt.Sprint(schema))

	rdftype := schema.propMap["http://www.w3.org/1999/02/22-rdf-syntax-ns#type"]
	memberOf := schema.propMap["http://www.wikidata.org/prop/direct/P463"]
	list := []*iItem{rdftype, memberOf}
	fmt.Println(schema.Support(list), schema.Root.Support)

	t1 = time.Now()
	rec := schema.recommendProperty(list)
	fmt.Println(time.Since(t1))

	PrintMemUsage()
	fmt.Println(rec[:10])

	schema.Save("schemaTree.bin")
	schema, _ = LoadSchemaTree("schemaTree.bin")
	rec = schema.recommendProperty(list)

	PrintMemUsage()
	fmt.Println(rec[:10])

}
