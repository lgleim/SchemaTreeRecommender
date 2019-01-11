package main

import (
	"bufio"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"time"
)

// first pass: collect I-List and statistics
func (schema *SchemaTree) firstPass(fileName string, firstN uint64) {
	if _, err := os.Stat(fileName + ".firstPass.bin"); os.IsNotExist(err) {
		counter := func(s *subjectSummary) {
			for prop := range s.properties {
				prop.increment()
			}
		}

		t1 := time.Now()
		subjectCount := subjectSummaryReader(fileName, schema.propMap, schema.typeMap, counter, firstN)

		fmt.Printf("%v properties, %v types\n", len(schema.propMap), len(schema.typeMap))

		f, _ := os.Create(fileName + ".propMap")
		gob.NewEncoder(f).Encode(schema.propMap)
		f.Close()
		f, _ = os.Create(fileName + ".typeMap")
		gob.NewEncoder(f).Encode(schema.typeMap)
		f.Close()

		schema.updateSortOrder()

		fmt.Println("First Pass:", time.Since(t1))
		PrintMemUsage()
		if subjectCount != uint64(schema.Root.Support) {
			fmt.Println("#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#")
			fmt.Printf("WARNING: uint32 OVERFLOW - Processed %v subjects but root support is only %v!\n", subjectCount, schema.Root.Support)
			fmt.Println("#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#!#")
		}

		err = schema.Save(fileName + ".firstPass.bin")
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		// f1, err1 := os.Open(fileName + ".propMap")
		// f2, err2 := os.Open(fileName + ".typeMap")

		// if err1 == nil && err2 == nil {
		// 	fmt.Print("Loading type- and propertyMap directly from corresponding gobs: ")
		// 	tmp := NewSchemaTree()
		// 	gob.NewDecoder(f1).Decode(&tmp.propMap)
		// 	gob.NewDecoder(f2).Decode(&tmp.typeMap)
		// 	tmp.updateSortOrder()
		// 	*schema = *tmp
		// 	fmt.Printf("%v properties, %v types\n", len(tmp.propMap), len(tmp.typeMap))
		// } else {
		tmp, err := LoadSchemaTree(fileName + ".firstPass.bin")
		if err != nil {
			log.Fatalln(err)
		}
		*schema = *tmp
		// }
	}
}

// build schema tree
func (schema *SchemaTree) secondPass(fileName string, firstN uint64) {
	schema.updateSortOrder() // duplicate -- legacy compatability

	inserter := func(s *subjectSummary) {
		schema.Insert(s)
	}

	// go countTreeNodes(schema)

	t1 := time.Now()
	subjectSummaryReader(fileName, schema.propMap, schema.typeMap, inserter, firstN)

	fmt.Println("Second Pass:", time.Since(t1))
	PrintMemUsage()
	// PrintLockStats()
}

func (schema *SchemaTree) twoPass(fileName string, firstN uint64) {
	// go func() {
	// 	for true {
	// 		time.Sleep(10 * time.Second)
	// 		PrintMemUsage()
	// 	}
	// }()
	schema.firstPass(fileName, firstN)
	schema.secondPass(fileName, firstN)
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	fileName := flag.String("file", "experiments/10M.nt.gz", "the file to parse")
	firstNsubjects := flag.Int64("n", 0, "Only parse the first n subjects") // TODO: handle negative inputs
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile := flag.String("memprofile", "", "write memory profile to `file`")
	traceFile := flag.String("trace", "", "write execution trace to `file`")
	loadBinary := flag.String("load", "", "read stored schematree from `file`")
	visualize := flag.Bool("viz", false, "output a GraphViz visualization of the tree to `tree.png`")
	serveRest := flag.Bool("api", false, "specifying this flag enables the rest api")
	serveOnPort := flag.Int("port", 8080, "the port the rest interface will be served on. Use in conjunction with -api")

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

	// write cpu profile to file
	if *memprofile != "" {
		defer func() {
			f, err := os.Create(*memprofile)
			if err != nil {
				log.Fatal("could not create memory profile: ", err)
			}
			runtime.GC() // get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatal("could not write memory profile: ", err)
			}
			f.Close()
		}()
	}

	// write cpu profile to file
	if *traceFile != "" {
		f, err := os.Create(*traceFile)
		if err != nil {
			log.Fatal("could not create trace file: ", err)
		}
		if err := trace.Start(f); err != nil {
			log.Fatal("could not start tracing: ", err)
		}
		defer trace.Stop()
	}

	// t1 := time.Now()
	var schema *SchemaTree

	if *loadBinary != "" {
		var err error
		schema, err = LoadSchemaTree(*loadBinary)
		if err != nil {
			fmt.Println(err)
		}
	} else {
		schema = NewSchemaTree()
		schema.twoPass(*fileName, uint64(*firstNsubjects))

		PrintMemUsage()

		schema.Save(*fileName + ".schemaTree.bin")
	}

	if *visualize {
		f, err := os.Create("tree.dot")
		if err == nil {
			defer f.Close()
			f.WriteString(fmt.Sprint(schema))
			fmt.Println("Run e.g. `dot -Tsvg tree.dot -o tree.svg` to visualize!")
		}
	}

	if *serveRest {
		serve(schema, *serveOnPort)
		waitForReturn()
	}
}

func waitForReturn() {
	buf := bufio.NewReader(os.Stdin)
	fmt.Print("> ")
	sentence, err := buf.ReadBytes('\n')
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(string(sentence))
	}
}
