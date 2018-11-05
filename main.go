package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

func twoPass(fileName string, firstN uint64) *SchemaTree {
	// first pass: collect I-List and statistics
	t1 := time.Now()

	schema := SchemaTree{
		propMap: make(propMap),
		typeMap: make(typeMap),
		Root:    newRootNode(),
		MinSup:  3,
	}

	PrintMemUsage()

	counter := func(s subjectSummary) {
		for _, prop := range s.properties {
			prop.increment()
		}
	}
	subjectSummaryReader(fileName, &schema.propMap, &schema.typeMap, counter, firstN)

	fmt.Println("First Pass:", time.Since(t1))
	PrintMemUsage()

	// second pass
	t1 = time.Now()

	schema.updateSortOrder()

	inserter := func(s subjectSummary) {
		schema.Insert(&s, false)
	}
	subjectSummaryReader(fileName, &schema.propMap, &schema.typeMap, inserter, firstN)

	fmt.Println("Second Pass:", time.Since(t1))
	PrintMemUsage()

	return &schema
}

func main() {
	fileName := flag.String("file", "10M.nt.gz", "the file to parse")
	firstNsubjects := flag.Int64("n", 0, "Only parse the first n subjects") // TODO: handle negative inputs
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile := flag.String("memprofile", "", "write memory profile to `file`")

	// parse commandline arguments/flags
	flag.Parse()

	// write cpu profile to file
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	t1 := time.Now()
	schema := twoPass(*fileName, uint64(*firstNsubjects))

	// r := &renderer.PNGRenderer{
	// 	OutputFile: "my_graph.png",
	// }
	// r.Render(fmt.Sprint(schema))

	rdftype := schema.propMap.get("http://www.w3.org/1999/02/22-rdf-syntax-ns#type")
	memberOf := schema.propMap.get("http://www.wikidata.org/prop/direct/P463")
	list := []*iItem{rdftype, memberOf}
	fmt.Println(schema.Support(list), schema.Root.Support)

	t1 = time.Now()
	rec := schema.recommendProperty(list)
	fmt.Println(time.Since(t1))

	PrintMemUsage()
	fmt.Println(rec[:10])

	// schema.Save("schemaTree.bin")
	// schema, _ = LoadSchemaTree("schemaTree.bin")
	// rec = schema.recommendProperty(list)

	// PrintMemUsage()
	// fmt.Println(rec[:10])

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
		f.Close()
	}
}
