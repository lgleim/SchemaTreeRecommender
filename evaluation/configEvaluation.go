package main

import (
	"fmt"
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
	//writeCSV
	return nil
}

func runConfig(name *string, tree *schematree.SchemaTree, typed bool, handler string) (statistic evalSummary, err error) {
	config, err := configuration.ReadConfigFile(name)
	if err != nil {
		return
	}
	wf, err := configuration.ConfigToWorkflow(config, tree)
	if err != nil {
		return
	}
	results := evaluateDataset(tree, wf, typed, config.Testset, handler)
	statistic = makeStatistics(results, "setSize")[0]
	return
}
