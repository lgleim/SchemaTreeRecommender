package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"recommender/schematree"
	"recommender/server"
	"recommender/strategy"
	"runtime"
	"runtime/pprof"
	"runtime/trace"

	"github.com/spf13/cobra"
)

func main() {

	// Program initialization actions
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Setup the variables where all flags will reside.
	var cpuprofile, memprofile, traceFile string // used globally
	var firstNsubjects int64                     // used by build-tree
	var writeOutPropertyFreqs bool               // used by build-tree
	var strategyName string                      // used by serve
	var serveOnPort int                          // used by serve

	// writeOutPropertyFreqs := flag.Bool("writeOutPropertyFreqs", false, "set this to write the frequency of all properties to a csv after first pass or schematree loading")

	// root command
	cmdRoot := &cobra.Command{
		Use: "recommender",

		// Execute global pre-run activities such as profiling.
		PersistentPreRun: func(cmd *cobra.Command, args []string) {

			// write cpu profile to file - open file and start profiling
			if cpuprofile != "" {
				f, err := os.Create(cpuprofile)
				if err != nil {
					log.Fatal("could not create CPU profile: ", err)
				}
				if err := pprof.StartCPUProfile(f); err != nil {
					log.Fatal("could not start CPU profile: ", err)
				}
			}

			// write trace execution to file - open file and start tracing
			if traceFile != "" {
				f, err := os.Create(traceFile)
				if err != nil {
					log.Fatal("could not create trace file: ", err)
				}
				if err := trace.Start(f); err != nil {
					log.Fatal("could not start tracing: ", err)
				}
			}

		},

		// Close whatever profiling was running globally.
		PersistentPostRun: func(cmd *cobra.Command, args []string) {

			// write cpu profile to file - stop profiling
			if cpuprofile != "" {
				pprof.StopCPUProfile()
			}

			// write memory profile to file
			if memprofile != "" {
				f, err := os.Create(memprofile)
				if err != nil {
					log.Fatal("could not create memory profile: ", err)
				}
				runtime.GC() // get up-to-date statistics
				if err := pprof.WriteHeapProfile(f); err != nil {
					log.Fatal("could not write memory profile: ", err)
				}
				f.Close()
			}

			// write trace execution to file - stop tracing
			if traceFile != "" {
				trace.Stop()
			}

		},
	}

	// flags for root command
	cmdRoot.PersistentFlags().StringVar(&cpuprofile, "cpuprofile", "", "write cpu profile to `file`")
	cmdRoot.PersistentFlags().StringVar(&memprofile, "memprofile", "", "write memory profile to `file`")
	cmdRoot.PersistentFlags().StringVar(&traceFile, "trace", "", "write execution trace to `file`")

	// subcommand build-tree
	cmdBuildTree := &cobra.Command{
		Use:   "build-tree <dataset>",
		Short: "Build the SchemaTree model",
		Long: "A SchemaTree model will be built using the file provided in <dataset>. Two output files will be" +
			" generated in the same directory as <dataset> and with suffixed names, namely:" +
			" '<dataset>.firstPass.bin' and '<dataset>.schemaTree.bin'",
		Args: cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			inputDataset := &args[0]

			// Create the tree output file by using the input dataset.
			schema := schematree.NewSchemaTree()
			schema.TwoPass(*inputDataset, uint64(firstNsubjects))
			schema.Save(*inputDataset + ".schemaTree.bin")
			schematree.PrintMemUsage()

			if writeOutPropertyFreqs {
				propFreqsPath := *inputDataset + ".propertyFreqs.csv"
				schema.WritePropFreqs(propFreqsPath)
				fmt.Printf("Wrote PropertyFreqs to %s\n", propFreqsPath)
			}

		},
	}

	// flags for build-tree subcommand
	// cmdBuildTree.Flags().StringVarP(&inputDataset, "dataset", "d", "", "`path` to the dataset file to parse")
	// cmdBuildTree.MarkFlagRequired("dataset")
	cmdBuildTree.Flags().Int64VarP(&firstNsubjects, "first", "n", 0, "only parse the first `n` subjects") // TODO: handle negative inputs
	cmdBuildTree.Flags().BoolVarP(
		&writeOutPropertyFreqs, "write-frequencies", "f", false,
		"write all property frequencies to a csv file named '<dataset>.propertyFreqs.csv' after the SchemaTree is built",
	)

	// subcommand serve
	cmdServe := &cobra.Command{
		Use:   "serve <tree>",
		Short: "Serve a SchemaTree model via an HTTP Server",
		Long: "Load the schematree binary stored in path given by <tree> and then serve it using an" +
			" HTTP Server. Available endpoints are given on startup.",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			treeBinary := &args[0]

			// Load the schematree from the binary file.
			schema, err := schematree.LoadSchemaTree(*treeBinary)
			if err != nil {
				log.Panicln(err)
			}
			schematree.PrintMemUsage()

			// Fetch the strategy by name. (TODO)
			workflow := strategy.MakePresetWorkflow(strategyName, schema)

			// Initiate the HTTP server. Make it stop on <Enter> press.
			router := server.SetupEndpoints(schema, workflow)
			go http.ListenAndServe(fmt.Sprintf("0.0.0.0:%v", serveOnPort), router)
			fmt.Printf("Now listening on port %v\n", serveOnPort)
			waitForReturn()
		},
	}

	// flags for serve subcommand
	// cmdBuildTree.Flags().StringVarP(&treeBinary, "tree", "t", "", "read stored schematree from `file`")
	// cmdBuildTree.MarkFlagRequired("load")
	cmdServe.Flags().IntVarP(&serveOnPort, "port", "p", 8080, "`port` of http server")
	cmdServe.Flags().StringVarP(&strategyName, "strategy", "s", "direct", "`name` of strategy to use")

	// subcommand visualize
	cmdBuildDot := &cobra.Command{
		Use:   "build-dot <tree>",
		Short: "Build a DOT file from a schematree binary",
		Long: "Load the schematree binary stored in path given by <tree> and build a DOT file using" +
			" the GraphViz toolbox.\n" +
			"Will create a file in the same directory as <tree>, with the name: '<tree>.dot'\n",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			treeBinary := &args[0]

			// Load the schematree from the binary file.
			schema, err := schematree.LoadSchemaTree(*treeBinary)
			if err != nil {
				log.Panicln(err)
			}
			schematree.PrintMemUsage()

			// Write the dot file and open it with visualizer.
			// TODO: output a GraphViz visualization of the tree to `tree.png
			// TODO: Println could show the real file name
			f, err := os.Create(*treeBinary + ".dot")
			if err == nil {
				defer f.Close()
				f.WriteString(fmt.Sprint(schema))
				fmt.Println("Run e.g. `dot -Tsvg tree.dot -o tree.svg` to visualize!")
			}

		},
	}

	// putting the command hierarchy together
	cmdRoot.AddCommand(cmdBuildTree)
	cmdRoot.AddCommand(cmdServe)
	cmdRoot.AddCommand(cmdBuildDot)

	// Start the CLI application
	cmdRoot.Execute()

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
