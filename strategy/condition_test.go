package strategy

import (
	"recommender/schematree"
	"testing"
)

//var treePath = "../testdata/10M.nt.gz.schemaTree.bin"

func TestConditions(t *testing.T) {
	schema, err := schematree.LoadSchemaTree(treePath)
	if err != nil {
		t.Errorf("Schematree could not be loaded")
	}
	pMap := schema.PropMap
	// create properties
	item1, _ := pMap["http://www.wikidata.org/prop/direct/P31"] // large number (1224) recommendations after executing on the schema tree ../testdata/10M.nt.gz.schemaTree.bin
	item2, _ := pMap["http://www.wikidata.org/prop/direct/P21"] // small number (487)

	// check all strategies
	countTooLessProperties := makeTooLessRecommendationsCondition(500, schema)
	if countTooLessProperties(schematree.IList{item1}) || !countTooLessProperties(schematree.IList{item2}) {
		t.Errorf("'TooLessRecommendationsCondition' failed.")
	}

	countTooManyProperties := makeTooManyRecommendationsCondition(500, schema)
	if !countTooManyProperties(schematree.IList{item1}) || countTooManyProperties(schematree.IList{item2}) {
		t.Errorf("'TooManyRecommendationsCondition' failed.")
	}

	aboveThreshholdCondition := makeAboveThresholdCondition(1)
	if aboveThreshholdCondition(schematree.IList{item1}) || !aboveThreshholdCondition(schematree.IList{item2, item1}) {
		t.Errorf("'aboveThreshholdCondition' failed.")
	}

}
