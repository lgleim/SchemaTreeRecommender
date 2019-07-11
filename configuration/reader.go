package configuration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"recommender/schematree"
	"recommender/strategy"

	"github.com/pkg/errors"
)

//Layer defines configuration of one layer (condition, backoff pair) in the workflow
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

//Configuration defines one workflow configuration
type Configuration struct {
	Testset string  // testset to apply (only relevant for batch evaluation. Inrelevant for standard usage)
	Layers  []Layer // layers to apply
}

//ReadConfigFile reads json config file <name> to Configuration struct
func ReadConfigFile(name *string) (conf *Configuration, err error) {
	var c Configuration
	file, err := ioutil.ReadFile(*name)
	if err != nil {
		err = errors.Errorf("Read File failed")
		return
	}
	err = json.Unmarshal(file, &c)
	conf = &c
	return
}

//ConfigToWorkflow converts a configuration to a workflow
func ConfigToWorkflow(config *Configuration, tree *schematree.SchemaTree) (wf *strategy.Workflow, err error) {
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
				back = strategy.MakeDeleteLowFrequencyProcedure(tree, l.ParallelExecutions, strategy.StepsizeLinear, strategy.MakeMoreThanInternalCondition(l.Threshold))
			case "stepsizeProportional":
				back = strategy.MakeDeleteLowFrequencyProcedure(tree, l.ParallelExecutions, strategy.StepsizeProportional, strategy.MakeMoreThanInternalCondition(l.Threshold))
			default:
				err = errors.Errorf("Merger not found")
				return
			}
		case "standard":
			back = strategy.MakeAssessmentAwareDirectProcedure()
		case "splitProperty":
			var merger strategy.MergerFunc
			var splitter strategy.SplitterFunc
			switch l.Merger {
			case "max":
				merger = strategy.MaxMerger

			case "avg":
				merger = strategy.AvgMerger
			default:
				err = errors.Errorf("Merger not found")
				return
			}

			switch l.Splitter {
			case "everySecondItem":
				splitter = strategy.EverySecondItemSplitter

			case "twoSupportRanges":
				splitter = strategy.TwoSupportRangesSplitter
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

// Test if the configuration is well formatted and all attributes for the chosen strategy are set.
// Check for correct attribution happens in configToWorkflow()
func (conf *Configuration) Test() (err error) {
	if len(conf.Layers) == 0 {
		err = errors.Errorf("Configuration File Failure: No Layers Specified")
		return
	}
	for i, lay := range conf.Layers {
		if lay.Backoff == "" {
			err = errors.Errorf("Configuration File Failure: Layer %v Backoff Strategy is empty", i)
			return
		}
		if lay.Backoff == "splitProperty" && (lay.Merger == "" || lay.Splitter == "") {
			err = errors.Errorf("Configuration File Failure: Layer %v needs splitter and merger", i)
		}
		if lay.Backoff == "deleteLowFrequency" && (lay.Stepsize == "" || lay.ParallelExecutions == 0) {
			err = errors.Errorf("Configuration File Failure: Layer %v needs Stepsize Function and #parallel executions", i)
		}
	}
	return nil
}
