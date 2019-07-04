package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"recommender/backoff"
	"recommender/schematree"
	"recommender/strategy"

	"github.com/pkg/errors"
)

// Run all config files defined in ./configs and create a results csv table in ./
// with schematree in ../testdata/10M.nt.gz.schemaTree.bin
func batchConfigBenchmark(treePath string, configs int) (err error) {
	schema := schematree.NewSchemaTree()
	schema, err = schematree.LoadSchemaTree(treePath)
	if err != nil {
		return err
	}
	var filename string
	eval := make([]evalSummary, 0, configs)
	for i := 0; i < configs; i++ {
		filename = fmt.Sprintf("config_%v", i)
		res, err := runConfig(&filename, schema)
		if err != nil {
			return err
		}
		eval = append(eval, res)
	}
	writeCSV(&eval, "batch_test_results")
	return nil
}

func runConfig(name *string, tree *schematree.SchemaTree) (result evalSummary, err error) {
	config, err := readConfigFile(name)
	if err != nil {
		return
	}
	wf, err := configToWorkflow(config, tree)
	if err != nil {
		return
	}
	stats, resources := evaluation(tree, &config.Testset, wf)
	result = makeStatistics(stats, resources)[0]
	return
}

// configuration of one layer (condition, backoff pair) in the workflow
type Layer struct {
	Condition          string  // executed condition aboveThreshold, tooManyRecommendations,tooFewRecommendations
	Backoff            string  // executed backoff splitProperty, deleteLowFrequency
	Threshold          int     // neeeded for conditions
	ThresholdFloat     float32 // needed for condition TooUnlikelyRecommendationsCondition
	Merger             string  // needed for splitintosubsets backoff; max, avg
	Splitter           string  // needed for splitintosubsets backoff everySecondItem, twoSupportRanges
	Stepsize           string  // needed for deletelowfrequentitmes backoff stepsizeLinear, stepsizeProportional
	ParallelExecutions int     // needed for deletelowfrequentitmes backoff
}

// one workflow configuration
type Configuration struct {
	Testset string  // tesset to apply
	Layers  []Layer // layers to apply
}

// read config file ./configs/<name>.json to Configuration struct
func readConfigFile(name *string) (conf *Configuration, err error) {
	var c Configuration
	file, err := ioutil.ReadFile("./configs/" + *name + ".json")
	if err != nil {
		return
	}
	err = json.Unmarshal(file, &c)
	conf = &c
	return
}

func configToWorkflow(config *Configuration, tree *schematree.SchemaTree) (wf *strategy.Workflow, err error) {
	workflow := strategy.Workflow{}
	for i, l := range config.Layers {
		var cond strategy.Condition
		var back strategy.Procedure
		//switch the conditions
		switch l.Condition {
		case "aboveThreshold":
			cond = strategy.MakeAboveThresholdCondition(l.Threshold)
		case "tooUnlikelyRecommendationsCondition":
			cond = strategy.MakeTooUnlikelyRecommendationsCondition(l.ThresholdFloat)
		case "tooFewRecommendations":
			cond = strategy.MakeTooFewRecommendationsCondition(l.Threshold)
		case "always":
			cond = strategy.MakeAlwaysCondition()
		default:
			cond = strategy.MakeAlwaysCondition()
			err = errors.Errorf("Condition not found: " + l.Condition)
		}

		//switch the backoffs
		switch l.Backoff {
		case "deleteLowFrequency":
			switch l.Stepsize {
			case "stepsizeLinear":
				back = strategy.MakeDeleteLowFrequencyProcedure(tree, l.ParallelExecutions, backoff.StepsizeLinear, backoff.MakeMoreThanInternalCondition(l.Threshold))
			case "stepsizeProportional":
				back = strategy.MakeDeleteLowFrequencyProcedure(tree, l.ParallelExecutions, backoff.StepsizeProportional, backoff.MakeMoreThanInternalCondition(l.Threshold))
			default:
				err = errors.Errorf("Merger not found")
				return
			}
		case "standard":
			back = strategy.MakeDirectProcedure(tree)
		case "splitProperty":
			var merger backoff.MergerFunc
			var splitter backoff.SplitterFunc
			switch l.Merger {
			case "max":
				merger = backoff.MaxMerger

			case "avg":
				merger = backoff.AvgMerger
			default:
				err = errors.Errorf("Merger not found")
				return
			}

			switch l.Splitter {
			case "everySecondItem":
				splitter = backoff.EverySecondItemSplitter

			case "twoSupportRanges":
				splitter = backoff.TwoSupportRangesSplitter
			default:
				err = errors.Errorf("Splitter not found")
				return
			}
			back = strategy.MakeSplitPropertyProcedure(tree, splitter, merger)
		case "tooFewRecommendations":
			cond = strategy.MakeTooFewRecommendationsCondition(l.Threshold)
		default:
			cond = strategy.MakeAlwaysCondition()
			err = errors.Errorf("Backoff not found: " + l.Backoff)
		}
		//create the wf layer
		workflow.Push(cond, back, fmt.Sprintf("layer %v", i))
	}
	wf = &workflow
	return
}

func writeCSV(evaluation *[]evalSummary, filename string) {
	output := fmt.Sprintf("%8v, %8v, %8v, %8v, %12v, %8v, %8v, %8v,%8v, %10v, %10v, %8v,%8v\n", "Config No.", "set", "median", "mean", "variance", "top1", "top5", "top10", "worst5avg", "sampleSize", "#subjects", "duration", "memoryAllocation")
	e := *evaluation
	for i, eval := range e {
		output += fmt.Sprintf("%8v, %8v, %8v, %8v, %12v, %8v, %8v, %8v, %8v, %10v, %10v, %8v,%8v\n", i, eval.setSize, eval.median, eval.mean, eval.variance, eval.top1, eval.top5, eval.top10, eval.worst5average, eval.sampleSize, eval.subjectCount, eval.duration, eval.memoryAllocation)
	}
	f, _ := os.Create(filename + ".csv")
	f.WriteString(output)
	f.Close()
}
