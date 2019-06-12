package schematree

import "testing"

var treePath = "../10M.nt.gz.schemaTree.bin"

func TestRecommender(t *testing.T) {
	schema, err := LoadSchemaTree(treePath)

	if err != nil {
		t.Errorf("Schematree could not be loaded")
	}
	pMap := schema.PropMap
	b := backoffSplitPropertySet{}
	b.init(schema, twoSupportRangesSplitter, dummyMerger)

	prop1, _ := pMap["http://www.wikidata.org/prop/direct/P31"]
	prop2, _ := pMap["http://www.wikidata.org/prop/direct/P21"]
	prop3, _ := pMap["http://www.wikidata.org/prop/direct/P27"]
	props := IList{prop1, prop2, prop3}

	b.recommend(props)

}
