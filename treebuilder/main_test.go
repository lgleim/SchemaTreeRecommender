package main

import (
	"fmt"
	"recommender/schematree"
	"runtime"
	"testing"
)

func TestRecommendation(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	fileName := "../testdata/10M.nt.gz"
	firstNsubjects := 1000
	properties := [3]string{"http://www.wikidata.org/prop/direct/P31", "http://www.wikidata.org/prop/direct/P21", "http://www.wikidata.org/prop/direct/P27"}

	var schema *schematree.SchemaTree

	schema = schematree.NewSchemaTree(false, 1)
	schema.TwoPass(fileName, uint64(firstNsubjects))
	//schema.Save(fileName + ".schemaTree.bin")
	pMap := schema.PropMap

	list := []*schematree.IItem{}
	for _, pString := range properties {
		p, ok := pMap[pString]
		if ok {
			list = append(list, p)
		}
	}

	rec := schema.RecommendProperty(list)

	if len(rec) > 500 {
		rec = rec[:500]
	}
	fmt.Println(rec)
}
