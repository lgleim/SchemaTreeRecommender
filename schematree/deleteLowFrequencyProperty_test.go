package schematree

import (
	"fmt"
	"testing"
)

var treePath = "../testdata/10M.nt.gz.schemaTree.bin"

func TestManipulator(t *testing.T) {
	b := backoffDeleteLowFrequencyItems{}
	b.init(nil, 5, stepsizeLinear)

	ps := orderedList(10)
	ls := IList{ps[0], ps[1], ps[3], ps[2], ps[0], ps[2], ps[3], ps[0]}
	ls1, _, err1 := b.manipulate(ls, 5)
	if len(ls1) != 3 || ls1[0] != ps[0] || ls1[1] != ps[1] || ls1[2] != ps[3] || err1 != nil {
		t.Error("Slicing not working (1)! Result is " + fmt.Sprint(ls1))
	}
	ls2, _, err2 := b.manipulate(ls, 8)
	if len(ls2) != 0 || err2 != nil {
		t.Error("Slicing not working (2)! Result is " + fmt.Sprint(ls2) + " and should be []")
	}
	_, _, err3 := b.manipulate(ls, 10)
	if err3 == nil {
		t.Error("Error not detected  (3) ")
	}

}

func TestExecRecommender(t *testing.T) {
	schema, err := LoadSchemaTree(treePath)

	if err != nil {
		t.Errorf("Schematree could not be loaded")
	}
	pMap := schema.PropMap
	b := backoffDeleteLowFrequencyItems{}
	b.init(schema, 1, stepsizeLinear)
	c := make(chan chanObject, 1)

	prop1, _ := pMap["http://www.wikidata.org/prop/direct/P31"]
	prop2, _ := pMap["http://www.wikidata.org/prop/direct/P21"]
	prop3, _ := pMap["http://www.wikidata.org/prop/direct/P27"]
	props := IList{prop1, prop2, prop3}

	removed := []*IItem{}
	recommenderClassic := schema.RecommendProperty(props)
	for i := 0; i < 10; i++ {
		removed = append(removed, recommenderClassic[i].Property)
	}

	b.execRecommender(props, removed, 1, c)
	rec := <-c
	for _, r := range rec.recommendations {
		for _, r2 := range removed {
			if *r.Property.Str == *r2.Str {
				t.Errorf("Deletion of removed item didn't work")
			}
		}
	}
}
