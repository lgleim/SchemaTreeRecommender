package main

import (
	"fmt"
	"recommender/configuration"
	"recommender/schematree"
)

// Run all config files defined in ./configs and create a results csv table in ./
// with schematree in ../testdata/10M.nt.gz.schemaTree.bin
func batchConfigBenchmark(treePath string, configs int, typed bool, handler string) (eval []evalSummary, err error) {

	schema, err := schematree.Load(treePath)
	if err != nil {
		return nil, err
	}
	var filename string
	eval = make([]evalSummary, 0, configs)
	for i := 0; i < configs; i++ {
		filename = fmt.Sprintf("./configs/config_%v.json", i)
		res, err := runConfig(&filename, schema, typed, handler, int16(i))
		if err != nil {
			return nil, err
		}
		eval = append(eval, res)
	}
	return
}

// runs config files and groups by config file
func runConfig(name *string, tree *schematree.SchemaTree, typed bool, handler string, run int16) (statistic evalSummary, err error) {
	config, err := configuration.ReadConfigFile(name)
	if err != nil {
		return
	}
	wf, err := configuration.ConfigToWorkflow(config, tree)
	if err != nil {
		return
	}
	results := evaluateDataset(tree, wf, typed, config.Testset, handler)
	statistic = makeStatistics(results, "numNonTypes")[0]
	statistic.groupBy = run
	return
}
