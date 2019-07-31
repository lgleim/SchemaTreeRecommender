package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"recommender/configuration"
	recIO "recommender/io"
	"recommender/schematree"
	"recommender/strategy"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile := flag.String("memprofile", "", "write memory profile to `file`")
	traceFile := flag.String("trace", "", "write execution trace to `file`")
	trainedModel := flag.String("model", "", "read stored schematree from `file`")
	configPath := flag.String("workflow", "", "Path to workflow config file for single evaluation")
	testFile := flag.String("testSet", "", "the file to parse")
	batchTest := flag.Bool("batchTest", false, "Switch between batch test and normal test")
	createConfigs := flag.Bool("createConfigs", false, "Create a bunch of config")
	createConfigsCreater := flag.String("creater", "", "Json which defines the creater config file in ./configs")
	numberConfigs := flag.Int("numberConfigs", 1, "CNumber of config files in ./configs")
	typedEntities := flag.Bool("typed", false, "Use type information or not")
	handlerType := flag.String("handler", "takeOneButType", "Choose the handler: takeOneButType, takeAllButBest")
	groupBy := flag.String("groupBy", "setSize", "Choose groupBy: setSize, numTypes, numLeftOut, numNonTypes")
	writeResults := flag.Bool("results", false, "Turn on to write an additional CSV file with all evaluation results")

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

	if *createConfigs {
		if *createConfigsCreater == "" {
			log.Fatalln("A Create Config File must be provided in ./configs!")
		}
		createConfigFiles(createConfigsCreater)
	} else if *batchTest {
		// Run all config files and benchmark those. Schematree is taken from ../testdata/10M.nt.gz.schemaTree.bin
		// test data is encoded in the config files
		// Output is csv file in ./
		if *trainedModel == "" {
			log.Fatalln("A model must be provided for Batch Test!")
			return
		}
		fmt.Printf("Evaluating the Config Files...")
		datasetStatistics, err := batchConfigBenchmark(*trainedModel, *numberConfigs, *typedEntities, *handlerType)
		if err != nil {
			log.Fatalln("Batch Config Failed", err)
			return
		}

		fmt.Printf("Writing results to CSV file...")
		writeStatisticsToFile("BatchTestResults", "Config File", datasetStatistics)
		fmt.Printf(" Complete.\n")
	} else {

		if *testFile == "" {
			log.Fatalln("A test set must be provided!")
		}

		// evaluation
		if *trainedModel == "" {
			log.Fatalln("A model must be provided!")
		}
		tree, err := schematree.LoadSchemaTree(*trainedModel)
		if err != nil {
			log.Fatalln(err)
		}

		var wf *strategy.Workflow
		if *configPath != "" {
			//load workflow config if given
			config, err := configuration.ReadConfigFile(configPath)
			if err != nil {
				log.Fatalln(err)
			}
			err = config.Test()
			if err != nil {
				log.Fatalln(err)
			}
			wf, err = configuration.ConfigToWorkflow(config, tree)
			if err != nil {
				log.Fatalln(err)
			}
		} else {
			// if no workflow config given then run standard recommender
			wf = strategy.MakePresetWorkflow("direct", tree)
		}

		fmt.Println("Evaluating the dataset...")
		datasetResults := evaluateDataset(tree, wf, *typedEntities, *testFile, *handlerType)

		// Calculate the base name of the input file to generate CSVs with similar names.
		testBase := recIO.TrimExtensions(*testFile) + "-" + *handlerType + "-" + *groupBy

		// When results flag is given, will also write a CSV for evalResult array
		if *writeResults {
			fmt.Printf("Writing results to CSV file...")
			writeResultsToFile(testBase+"-results", datasetResults)
			fmt.Printf(" Complete.\n")
		}

		fmt.Printf("Aggregating the results...")
		datasetStatistics := makeStatistics(datasetResults, *groupBy)
		fmt.Printf(" Complete.\n")

		fmt.Printf("Writing statistics to CSV file...")
		writeStatisticsToFile(testBase+"-stats", *groupBy, datasetStatistics)
		fmt.Printf(" Complete.\n")

		fmt.Printf("%v+\n", datasetStatistics[0])
	}
	//so something with statistics
	//fmt.Printf("%v+", statistics[0])
}
