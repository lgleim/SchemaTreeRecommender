package main

import (
	"recommender/schematree"
	"testing"
)

func TestEval(t *testing.T) {
	//trainingData := "../testdata/10M.nt.gz"
	testData := "../testdata/10M.nt_1in2_test.gz"

	schema := schematree.NewSchemaTree()
	//schema.TwoPass(trainingData, 1000000)
	schema, _ = schematree.LoadSchemaTree("../testdata/10M.nt.gz.schemaTree.bin")
	stats, resources := evaluation(schema, &testData)
	statistics := makeStatistics(stats, resources)

	for _, v := range statistics {
		if v.top10 < 85 {
			t.Fatalf("top10 is at %v", v.top10)
		}
	}
}
