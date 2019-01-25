package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"

	"schematree"
)

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
	var schema *schematree.SchemaTree

	if *loadBinary != "" {
		var err error
		schema, err = schematree.LoadSchemaTree(*loadBinary)
		if err != nil {
			fmt.Println(err)
		}
	} else {
		schema = schematree.NewSchemaTree()
		schema.TwoPass(*fileName, uint64(*firstNsubjects))

		schematree.PrintMemUsage()

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
		schematree.Serve(schema, *serveOnPort)
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
