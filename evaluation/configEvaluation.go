package main

import (
	"fmt"
	"os"
	"recommender/configuration"
	"recommender/schematree"
)

// Run all config files defined in ./configs and create a results csv table in ./
// with schematree in ../testdata/10M.nt.gz.schemaTree.bin
func batchConfigBenchmark(treePath string, configs int, typed bool, handler string) (err error) {

	schema, err := schematree.LoadSchemaTree(treePath)
	if err != nil {
		return err
	}
	var filename string
	eval := make([]evalSummary, 0, configs)
	for i := 0; i < configs; i++ {
		filename = fmt.Sprintf("./configs/config_%v.json", i)
		res, err := runConfig(&filename, schema, typed, handler)
		if err != nil {
			return err
		}
		eval = append(eval, res)
	}
	writeCSV(&eval, "batch_test_results.csv")
	return nil
}

func runConfig(name *string, tree *schematree.SchemaTree, typed bool, handler string) (result evalSummary, err error) {
	config, err := configuration.ReadConfigFile(name)
	if err != nil {
		return
	}
	wf, err := configuration.ConfigToWorkflow(config, tree)
	if err != nil {
		return
	}
	result = evaluation(tree, &config.Testset, wf, &typed, handler)[0]
	return
}

func writeCSV(evaluation *[]evalSummary, filename string) {
	output := fmt.Sprintf("%8v, %8v, %8v, %8v, %12v, %8v, %8v, %8v, %8v, %10v, %10v, %8v, %8v, %8v, %8v, %8v\n", "Config No.", "set", "median", "mean", "variance", "top1", "top5", "top10", "worst5avg", "sampleSize", "#subjects", "duration", "hitRate", "Precision", "Precision At 10", "Recommendation Count")
	e := *evaluation
	for i, eval := range e {
		output += fmt.Sprintf("%8v, %8v, %8v, %8v, %12v, %8v, %8v, %8v, %8v, %10v, %10v, %8v,%8v, %8v,%8v, %8v \n", i, eval.setSize, eval.median, eval.mean, eval.variance, eval.top1, eval.top5, eval.top10, eval.worst5average, eval.sampleSize, eval.subjectCount, eval.duration, eval.hitRate, eval.precision, eval.precisionAt10, eval.recommendationCount)
	}
	f, _ := os.Create(filename)
	f.WriteString(output)
	f.Close()
}
