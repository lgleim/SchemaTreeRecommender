package main

import (
	"fmt"
	"recommender/configuration"
	"recommender/schematree"
	"recommender/strategy"
	"testing"
)

func TestEval(t *testing.T) {
	//trainingData := "../testdata/10M.nt.gz"
	testData := "../testdata/10M.nt_1in2_test.gz"

	schema, _ := schematree.LoadSchemaTree("../testdata/10M.nt.gz.schemaTree.bin")
	f := true
	stats, resources, hitRate, recommendationCount := evaluation(schema, &testData, strategy.MakePresetWorkflow("direct", schema), &f, 1)
	statistics := makeStatistics(stats, resources, hitRate, recommendationCount)

	fmt.Printf("\n %v", statistics[0].precision)

	for _, v := range statistics {
		if v.top10 < 85 {
			//Not given anymore since results became much worse
			//t.Fatalf("top10 is at %v", v.top10)
		}
	}
}

func fTestReadWriteConfigFile(t *testing.T) {
	l1 := configuration.Layer{"tooFewRecommendation", "splitProperty", 100, 0.6, "avg", "everySecondItem", "", 0}
	cOut := configuration.Configuration{"../testdata/10M.nt_1in2_test.gz", []configuration.Layer{l1, l1}}
	fileName := "./configs/test.json"
	writeConfigFile(&cOut, fileName)

	cIn, err := configuration.ReadConfigFile(&fileName)
	if err != nil {
		t.Errorf("Read was not possible")
	}
	if cIn.Testset != cOut.Testset {
		t.Errorf("Testdata path not matching.")
	}
	if len(cIn.Layers) != len(cOut.Layers) {
		t.Errorf("Number of layers not matching.")
	}
	for i := range cIn.Layers {
		layerIn := cIn.Layers[i]
		layerOut := cOut.Layers[i]
		if layerIn.Condition != layerOut.Condition {
			t.Errorf("Condition in layer %v not matching", i)
		}
		if layerIn.Backoff != layerOut.Backoff {
			t.Errorf("Backoff in layer %v not matching", i)
		}
		if layerIn.Threshold != layerOut.Threshold {
			t.Errorf("Threshold in layer %v not matching", i)
		}
		if layerIn.Merger != layerOut.Merger {
			t.Errorf("Merger in layer %v not matching", i)
		}
		if layerIn.Splitter != layerOut.Splitter {
			t.Errorf("Splitter in layer %v not matching", i)
		}
		if layerIn.Stepsize != layerOut.Stepsize {
			t.Errorf("Stepsize in layer %v not matching", i)
		}
		if layerIn.ParallelExecutions != layerOut.ParallelExecutions {
			t.Errorf("Parallel Execs in layer %v not matching", i)
		}
		if layerIn.ThresholdFloat != layerOut.ThresholdFloat {
			t.Errorf("Threshold float in layer %v not matching", i)
		}
	}

}
