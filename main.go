package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"recommender/configuration"
	"recommender/glossary"
	"recommender/preparation"
	"recommender/schematree"
	"recommender/server"
	"recommender/strategy"
	"time"

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
	var measureTime bool                         // used globally
	var firstNsubjects int64                     // used by build-tree
	var writeOutPropertyFreqs bool               // used by build-tree
	var serveOnPort int                          // used by serve
	var workflowFile string                      // used by serve
	var everyNthSubject uint                     // used by split-dataset:1-in-n

	// Setup helper variables
	var timeCheckpoint time.Time // used globally

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

			// measure time - start measuring the time
			//   The measurements are done in such a way to not include the time for the profiles operations.
			if measureTime == true {
				timeCheckpoint = time.Now()
			}

		},

		// Close whatever profiling was running globally.
		PersistentPostRun: func(cmd *cobra.Command, args []string) {

			// measure time - stop time measurement and print the measurements
			if measureTime == true {
				fmt.Println("Execution Time:", time.Since(timeCheckpoint))
			}

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

	// global flags for root command
	cmdRoot.PersistentFlags().StringVar(&cpuprofile, "cpuprofile", "", "write cpu profile to `file`")
	cmdRoot.PersistentFlags().StringVar(&memprofile, "memprofile", "", "write memory profile to `file`")
	cmdRoot.PersistentFlags().StringVar(&traceFile, "trace", "", "write execution trace to `file`")
	cmdRoot.PersistentFlags().BoolVarP(&measureTime, "time", "t", false, "measure time of command execution")

	// subcommand build-tree
	cmdBuildTree := &cobra.Command{
		Use:   "build-tree <dataset>",
		Short: "Build the SchemaTree model",
		Long: "A SchemaTree model will be built using the file provided in <dataset>." +
			" The dataset should be a N-Triple of Items.\nTwo output files will be" +
			" generated in the same directory as <dataset> and with suffixed names, namely:" +
			" '<dataset>.firstPass.bin' and '<dataset>.schemaTree.bin'",
		Args: cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			inputDataset := &args[0]

			// Create the tree output file by using the input dataset.
			schema := schematree.Create(*inputDataset, uint64(firstNsubjects), false, 0)

			if writeOutPropertyFreqs {
				propFreqsPath := *inputDataset + ".propertyFreqs.csv"
				schema.WritePropFreqs(propFreqsPath)
				fmt.Printf("Wrote PropertyFreqs to %s\n", propFreqsPath)
			}

		},
	}
	// cmdBuildTree.Flags().StringVarP(&inputDataset, "dataset", "d", "", "`path` to the dataset file to parse")
	// cmdBuildTree.MarkFlagRequired("dataset")
	cmdBuildTree.Flags().Int64VarP(&firstNsubjects, "first", "n", 0, "only parse the first `n` subjects") // TODO: handle negative inputs
	cmdBuildTree.Flags().BoolVarP(
		&writeOutPropertyFreqs, "write-frequencies", "f", false,
		"write all property frequencies to a csv file named '<dataset>.propertyFreqs.csv' after the SchemaTree is built",
	)

	// subcommand build-tree
	cmdBuildTreeTyped := &cobra.Command{
		Use:   "build-tree-typed <dataset>",
		Short: "Build the SchemaTree model with types",
		Long: "A SchemaTree model will be built using the file provided in <dataset>." +
			" The dataset should be a N-Triple of Items.\nTwo output files will be" +
			" generated in the same directory as <dataset> and with suffixed names, namely:" +
			" '<dataset>.firstPass.bin' and '<dataset>.schemaTree.typed.bin'",
		Args: cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			inputDataset := &args[0]

			// Create the tree output file by using the input dataset.
			schema := schematree.Create(*inputDataset, uint64(firstNsubjects), true, 0)

			if writeOutPropertyFreqs {
				propFreqsPath := *inputDataset + ".propertyFreqs.csv"
				schema.WritePropFreqs(propFreqsPath)
				fmt.Printf("Wrote PropertyFreqs to %s\n", propFreqsPath)

				typeFreqsPath := *inputDataset + ".typeFreqs.csv"
				schema.WriteTypeFreqs(typeFreqsPath)
				fmt.Printf("Wrote PropertyFreqs to %s\n", typeFreqsPath)
			}

		},
	}
	// cmdBuildTree.Flags().StringVarP(&inputDataset, "dataset", "d", "", "`path` to the dataset file to parse")
	// cmdBuildTree.MarkFlagRequired("dataset")
	cmdBuildTreeTyped.Flags().Int64VarP(&firstNsubjects, "first", "n", 0, "only parse the first `n` subjects") // TODO: handle negative inputs
	cmdBuildTreeTyped.Flags().BoolVarP(
		&writeOutPropertyFreqs, "write-frequencies", "f", false,
		"write all property frequencies to a csv file named '<dataset>.propertyFreqs.csv' after the SchemaTree is built",
	)

	// subcommand build-glossary
	cmdBuildGlossary := &cobra.Command{
		Use:   "build-glossary <dataset>",
		Short: "Build the Glossary that maps properties to multi-lingual descriptions",
		Long: "A Glossary will be built using the file provided in <dataset>. The input" +
			" file should be a N-Triple of Property entries.\nThe output file will be" +
			" generated in the same directory as <dataset> with the name:" +
			" '<dataset>.glossary.bin'",
		Args: cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			inputDataset := &args[0]

			// Build the glossary
			glos, stats, err := glossary.BuildGlossary(*inputDataset)
			if err != nil {
				log.Panicln(err)
			}

			// Store it in the same directory with 'glossary.bin' extension
			glos.WriteToFile(*inputDataset + ".glossary.bin")
			fmt.Printf("%+v\n", stats)
			//glos.OutputStats()
		},
	}

	// subcommand serve
	cmdServe := &cobra.Command{
		Use:   "serve <model> <glossary>",
		Short: "Serve a SchemaTree model via an HTTP Server",
		Long: "Load the <model> (schematree binary) and the <glossary> (glossary binary) and the recommendation" +
			" endpoint using an HTTP Server.\nAvailable endpoints are stated in the server README.",
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			modelBinary := &args[0]
			glossaryBinary := &args[1]

			// Load the schematree from the binary file.
			model, err := schematree.LoadSchemaTree(*modelBinary)
			if err != nil {
				log.Panicln(err)
			}
			schematree.PrintMemUsage()

			// Load the glossary from the binary file.
			glos, err := glossary.ReadFromFile(*glossaryBinary)
			if err != nil {
				log.Panicln(err)
			}

			// read config file if given as parameter, test if everything needed is there, create a workflow
			// if no config file is given, the standard recommender is set as workflow.
			var workflow *strategy.Workflow
			if workflowFile != "" {
				config, err := configuration.ReadConfigFile(&workflowFile)
				if err != nil {
					log.Panicln(err)
				}
				err = config.Test()
				if err != nil {
					log.Panicln(err)
				}
				workflow, err = configuration.ConfigToWorkflow(config, model)
				if err != nil {
					log.Panicln(err)
				}
				log.Printf("Run Config Workflow %v", workflowFile)
			} else {
				workflow = strategy.MakePresetWorkflow("direct", model)
				fmt.Printf("Run Standard Recommender ")
			}

			// Initiate the HTTP server. Make it stop on <Enter> press.
			router := server.SetupEndpoints(model, glos, workflow, 500)
			fmt.Printf("Now listening on 0.0.0.0:%v\n", serveOnPort)
			http.ListenAndServe(fmt.Sprintf("0.0.0.0:%v", serveOnPort), router)

			// Note: Code before started server as sub-routine and waited for return.
			//go http.ListenAndServe(fmt.Sprintf("0.0.0.0:%v", serveOnPort), router)
			//waitForReturn()
		},
	}
	// cmdBuildTree.Flags().StringVarP(&treeBinary, "tree", "t", "", "read stored schematree from `file`")
	// cmdBuildTree.MarkFlagRequired("load")
	cmdServe.Flags().IntVarP(&serveOnPort, "port", "p", 8080, "`port` of http server")
	cmdServe.Flags().StringVarP(&workflowFile, "workflow", "w", "", "`path` to config file that defines the workflow")

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

	// subcommand split-dataset
	cmdSplitDataset := &cobra.Command{
		Use:   "split-dataset",
		Short: "Split a dataset using various methods",
		Long: "Select the method with which to split a N-Triple dataset file and" +
			" generate multiple smaller datasets in the same directory and with" +
			" suffixed names. Suffixes depend on chosen splitter method.",
		Args: cobra.NoArgs,
	}

	// subsubcommand split-dataset by-type
	cmdSplitDatasetByType := &cobra.Command{
		Use:   "by-type <dataset>",
		Short: "Split a dataset according to the type of wikidata entry",
		Long: "Split a N-Triple <dataset> file into three files according to the type of wikidata" +
			" entry: item, prop and misc.\nThe split files are generated in the same directory" +
			" as the <dataset>, stripped of their compression extension and given the following" +
			" names: <base>-item.nt.gz, <base>-prop.nt.gz, <base>-misc.nt.gz\n" +
			"This method assumes that all entries for a given subject are defined in contiguous lines.",
		Args: cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			inputDataset := &args[0]

			// Make the split
			sStats, err := preparation.SplitByType(*inputDataset)
			if err != nil {
				log.Panicln(err)
			}

			// Prepare and output the stats for it
			totalCount := float64(sStats.ItemCount + sStats.PropCount + sStats.MiscCount)
			fmt.Println("Split dataset by type:")
			fmt.Printf("  item: %d (%f)\n", sStats.ItemCount, float64(sStats.ItemCount)/totalCount)
			fmt.Printf("  prop: %d (%f)\n", sStats.PropCount, float64(sStats.PropCount)/totalCount)
			fmt.Printf("  misc: %d (%f)\n", sStats.MiscCount, float64(sStats.MiscCount)/totalCount)

		},
	}

	// subsubcommand split-dataset by-type
	cmdSplitDatasetByPrefix := &cobra.Command{
		Use:   "by-prefix <dataset>",
		Short: "Split a dataset according to the prefix of the subject",
		Long: "Split a N-Triple <dataset> file into three files according to the preset of the subject" +
			" into: item, prop and misc.\nThe split files are generated in the same directory" +
			" as the <dataset>, stripped of their compression extension and given the following" +
			" names: <base>-item.nt.gz, <base>-prop.nt.gz, <dataset>-misc.nt.gz",
		Args: cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			inputDataset := &args[0]

			// Make the split
			sStats, err := preparation.SplitByPrefix(*inputDataset)
			if err != nil {
				log.Panicln(err)
			}

			// Prepare and output the stats for it
			totalCount := float64(sStats.ItemCount + sStats.PropCount + sStats.MiscCount)
			fmt.Println("Split dataset by prefix:")
			fmt.Printf("  item: %d (%f)\n", sStats.ItemCount, float64(sStats.ItemCount)/totalCount)
			fmt.Printf("  prop: %d (%f)\n", sStats.PropCount, float64(sStats.PropCount)/totalCount)
			fmt.Printf("  misc: %d (%f)\n", sStats.MiscCount, float64(sStats.MiscCount)/totalCount)

		},
	}

	// subsubcommand split-dataset 1-in-n
	// TODO: Explain naming convention used for split datasets
	cmdSplitDatasetBySampling := &cobra.Command{
		Use:   "1-in-n <dataset>",
		Short: "Split a dataset using systematic sampling",
		Long: "Split a N-Triple <dataset> file into two files where every Nth subject goes into" +
			" one file and the rest into the second file.\nThe split files are generated in the same directory" +
			" as the <dataset>, stripped of their compression extension and given the following" +
			" names: <base>-1in<n>-test.nt.gz, <base>-1in<n>-train.nt.gz\n" +
			"This method assumes that all entries for a given subject are defined in contiguous lines.",
		Args: cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			inputDataset := &args[0]
			preparation.SplitBySampling(*inputDataset, int64(everyNthSubject))
		},
	}
	cmdSplitDatasetBySampling.Flags().UintVarP(&everyNthSubject, "nth", "n", 1000, "split every N-th subject")

	// subcommand filter-dataset
	cmdFilterDataset := &cobra.Command{
		Use:   "filter-dataset",
		Short: "Filter a dataset using various methods",
		Long:  "Filter the dataset for the purpose of building other models.",
		Args:  cobra.NoArgs,
	}

	// subsubcommand filter-dataset for-schematree
	cmdFilterDatasetForSchematree := &cobra.Command{
		Use:   "for-schematree <dataset>",
		Short: "Prepare the dataset for inclusion in the SchemaTree",
		Long: "Remove entries that should not be considered by the SchemaTree builder.\nThe new file is" +
			" generated in the same directory as the <dataset>, stripped of their compression extension" +
			" and given the following name: <base>-filtered.nt.gz",
		Args: cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			inputDataset := &args[0]

			// Execute the filter
			sStats, err := preparation.FilterForSchematree(*inputDataset)
			if err != nil {
				log.Panicln(err)
			}

			// Prepare and output the stats for it
			totalCount := float64(sStats.KeptCount + sStats.LostCount)
			fmt.Println("Filter dataset for schematree:")
			fmt.Printf("  kept: %d (%f)\n", sStats.KeptCount, float64(sStats.KeptCount)/totalCount)
			fmt.Printf("  lost: %d (%f)\n", sStats.LostCount, float64(sStats.LostCount)/totalCount)

		},
	}

	// subsubcommand filter-dataset for-glossary
	cmdFilterDatasetForGlossary := &cobra.Command{
		Use:   "for-glossary <dataset>",
		Short: "Prepare the dataset for inclusion in the Glossary",
		Long: "Remove entries that should not be considered by the Glossary builder.\nThe new file is" +
			" generated in the same directory as the <dataset>, stripped of their compression extension" +
			" and given the following name: <base>-filtered.nt.gz",
		Args: cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			inputDataset := &args[0]

			// Execute the filter
			sStats, err := preparation.FilterForGlossary(*inputDataset)
			if err != nil {
				log.Panicln(err)
			}

			// Prepare and output the stats for it
			totalCount := float64(sStats.KeptCount + sStats.LostCount)
			fmt.Println("Filter dataset for glossary:")
			fmt.Printf("  kept: %d (%f)\n", sStats.KeptCount, float64(sStats.KeptCount)/totalCount)
			fmt.Printf("  lost: %d (%f)\n", sStats.LostCount, float64(sStats.LostCount)/totalCount)

		},
	}

	// subsubcommand filter-dataset for-evaluation
	cmdFilterDatasetForEvaluation := &cobra.Command{
		Use:   "for-evaluation <dataset>",
		Short: "Prepare the dataset for usage in the evaluation",
		Long: "Remove entries that would not affect the evaluation results but which would make" +
			" the evaluation slower. Usually the case with multiple labels.\nCurrently this does" +
			" exactly the same as the 'for-schematree' filter.",
		Args: cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			inputDataset := &args[0]

			// Execute the filter
			sStats, err := preparation.FilterForEvaluation(*inputDataset)
			if err != nil {
				log.Panicln(err)
			}

			// Prepare and output the stats for it
			totalCount := float64(sStats.KeptCount + sStats.LostCount)
			fmt.Println("Filter dataset for evaluation:")
			fmt.Printf("  kept: %d (%f)\n", sStats.KeptCount, float64(sStats.KeptCount)/totalCount)
			fmt.Printf("  lost: %d (%f)\n", sStats.LostCount, float64(sStats.LostCount)/totalCount)

		},
	}

	// putting the command hierarchy together
	cmdRoot.AddCommand(cmdSplitDataset)
	cmdSplitDataset.AddCommand(cmdSplitDatasetByType)
	cmdSplitDataset.AddCommand(cmdSplitDatasetByPrefix)
	cmdSplitDataset.AddCommand(cmdSplitDatasetBySampling)
	cmdRoot.AddCommand(cmdFilterDataset)
	cmdFilterDataset.AddCommand(cmdFilterDatasetForSchematree)
	cmdFilterDataset.AddCommand(cmdFilterDatasetForGlossary)
	cmdFilterDataset.AddCommand(cmdFilterDatasetForEvaluation)
	cmdRoot.AddCommand(cmdBuildTree)
	cmdRoot.AddCommand(cmdBuildTreeTyped)
	cmdRoot.AddCommand(cmdBuildGlossary)
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
