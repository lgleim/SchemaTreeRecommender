package main

import (
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"os"
	"runtime"
	"recommender/schematree"
	"strings"
	"sync"
	"sync/atomic"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	testFile := flag.String("testSet", "", "the file to parse")

	logr := log.New(os.Stderr, "", 0)

	// parse commandline arguments/flags
	flag.Parse()
	if *testFile == "" {
		log.Fatalln("A test set must be provided!")
	}

	var wg sync.WaitGroup
	results := make(chan []string, 1000) // collect eval results via channel

	rand.Seed(1)

	var ctr, ctr2 uint64

	slens := make(map[int]int)

	handler := func(s *schematree.SubjectSummary) {
		properties := []string{}
		for p := range s.Properties {
			if strings.HasPrefix(*p.Str, "http://www.wikidata.org/prop/direct/") {
				properties = append(properties, strings.TrimPrefix(*p.Str, "http://www.wikidata.org/prop/direct/"))
			}
		}

		if len(properties) < 4 || len(properties) > 50 {
			return
		}
		atomic.AddUint64(&ctr, 1)
		if ctr%3 != 0 {
			return
		}
		atomic.AddUint64(&ctr2, 1)
		rand.Shuffle(len(properties), func(i, j int) { properties[i], properties[j] = properties[j], properties[i] })

		results <- properties
	}

	go func() {
		wg.Add(1)
		f, err := os.Create(*testFile + ".pSets.json")
		if err != nil {
			log.Fatalln("Could not open .pSets.json file")
		}
		defer f.Close()
		e := json.NewEncoder(f)
		for res := range results {
			slens[len(res)]++
			e.Encode(res)
		}
		wg.Done()
	}()

	tree := schematree.NewSchemaTree()

	subjectCount := schematree.SubjectSummaryReader(*testFile, tree.PropMap, tree.TypeMap, handler, 0)
	logr.Printf("\nEvaluation with total of %v subject sets!\n", subjectCount)
	close(results)
	wg.Wait()
	logr.Println("Total #usableSets:", ctr, ctr2)
	logr.Println(slens)

	return
}
