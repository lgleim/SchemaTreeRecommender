package schematree

import "testing"

var treePath = "../10M.nt.gz.schemaTree.bin"

func TestRecommender(t *testing.T) {
	schema, err := LoadSchemaTree(treePath)

	if err != nil {
		t.Errorf("Schematree could not be loaded")
	}
	pMap := schema.PropMap
	b := BackoffSplitPropertySet{}
	b.init(schema, TwoSupportRangesSplitter, DummyMerger)

	prop1, _ := pMap["http://www.wikidata.org/prop/direct/P31"]
	prop2, _ := pMap["http://www.wikidata.org/prop/direct/P21"]
	prop3, _ := pMap["http://www.wikidata.org/prop/direct/P27"]
	props := IList{prop1, prop2, prop3}

	b.Recommend(props)

}

func TestAvgMerger(t *testing.T) {

	schema, err := LoadSchemaTree(treePath)

	if err != nil {
		t.Errorf("Schematree could not be loaded")
	}
	pMap := schema.PropMap
	b := BackoffSplitPropertySet{}
	b.init(schema, TwoSupportRangesSplitter, DummyMerger)

	prop1, _ := pMap["http://www.wikidata.org/prop/direct/P31"]
	prop2, _ := pMap["http://www.wikidata.org/prop/direct/P21"]
	prop3, _ := pMap["http://www.wikidata.org/prop/direct/P27"]

	rec1 := PropertyRecommendations{RankedPropertyCandidate{Property: prop1, Probability: 0.2}, RankedPropertyCandidate{Property: prop2, Probability: 0.5}}
	rec2 := PropertyRecommendations{RankedPropertyCandidate{Property: prop1, Probability: 0.8}, RankedPropertyCandidate{Property: prop3, Probability: 0.4}}
	rec3 := PropertyRecommendations{RankedPropertyCandidate{Property: prop2, Probability: 0.2}}
	rec4 := PropertyRecommendations{RankedPropertyCandidate{Property: prop2, Probability: 0.3}}

	recommendations := []PropertyRecommendations{rec1, rec2, rec3, rec4}

	res := AvgMerger(recommendations)

	for _, r := range res {
		// Test values
		if *(r.Property.Str) == *(prop1.Str) && r.Probability != float64(0.25) {
			t.Errorf("Property 1 should have probability 0.25 but has %f", r.Probability)
		} else if *r.Property.Str == *prop2.Str && r.Probability != float64(0.25) {
			t.Errorf("Property 2 should have probability 0.25 but has %f", r.Probability)
		} else if *r.Property.Str == *prop3.Str && r.Probability != float64(0.1) {
			t.Errorf("Property 3 should have probability 0.1 but has %f", r.Probability)
		}
	}
	return
}

func TestMaxMerger(t *testing.T) {

	schema, err := LoadSchemaTree(treePath)

	if err != nil {
		t.Errorf("Schematree could not be loaded")
	}
	pMap := schema.PropMap
	b := BackoffSplitPropertySet{}
	b.init(schema, TwoSupportRangesSplitter, DummyMerger)

	prop1, _ := pMap["http://www.wikidata.org/prop/direct/P31"]
	prop2, _ := pMap["http://www.wikidata.org/prop/direct/P21"]
	prop3, _ := pMap["http://www.wikidata.org/prop/direct/P27"]

	rec1 := PropertyRecommendations{RankedPropertyCandidate{Property: prop1, Probability: 0.2}, RankedPropertyCandidate{Property: prop2, Probability: 0.5}}
	rec2 := PropertyRecommendations{RankedPropertyCandidate{Property: prop1, Probability: 0.8}, RankedPropertyCandidate{Property: prop3, Probability: 0.4}}
	rec3 := PropertyRecommendations{RankedPropertyCandidate{Property: prop2, Probability: 0.2}}

	recommendations := []PropertyRecommendations{rec1, rec2, rec3}

	res := MaxMerger(recommendations)

	for _, r := range res {
		// Test values
		if *(r.Property.Str) == *(prop1.Str) && r.Probability != 0.8 {
			t.Errorf("Property 1 should have probability 0.8 but has %f", r.Probability)
		} else if *r.Property.Str == *prop2.Str && r.Probability != 0.5 {
			t.Errorf("Property 2 should have probability 0.5 but has %f", r.Probability)
		} else if *r.Property.Str == *prop3.Str && r.Probability != 0.4 {
			t.Errorf("Property 3 should have probability 0.4 but has %f", r.Probability)
		}
	}
	return
}
