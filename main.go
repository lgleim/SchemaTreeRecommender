package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"sync"
	"sync/atomic"
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
	c := subjectSummaryReader(fileName, &schema.propMap, &schema.typeMap)

	concurrency := 12
	var wg sync.WaitGroup // goroutine coordination
	wg.Add(concurrency)
	var subjectCount uint64
	for i := 0; i < concurrency; i++ {
		go func() {
			for subjectSummary := range c {
				for _, prop := range subjectSummary.properties {
					prop.increment()
				}

				if atomic.AddUint64(&subjectCount, 1); firstN > 0 && subjectCount >= firstN {
					break
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()

	fmt.Println("First Pass:", time.Since(t1))
	PrintMemUsage()

	// second pass
	t1 = time.Now()
	c = subjectSummaryReader(fileName, &schema.propMap, &schema.typeMap)

	schema.updateSortOrder()

	subjectCount = 0
	concurrency = 12
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			for subjectSummary := range c {
				schema.Insert(subjectSummary, false)

				if atomic.AddUint64(&subjectCount, 1); firstN > 0 && subjectCount >= firstN {
					break
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()

	fmt.Println("Second Pass:", time.Since(t1))
	PrintMemUsage()

	return &schema
}

func main() {
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
