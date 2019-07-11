package strategy

import (
	"fmt"
	ST "recommender/schematree"
	"strconv"
	"testing"
)

//var treePath = "../testdata/10M.nt.gz.schemaTree.bin"

func i2iItem(i int) *ST.IItem {
	s := strconv.Itoa(i)
	return &ST.IItem{Str: &s, TotalCount: uint64(i), SortOrder: uint32(i)}
}

func orderedList(length int) (ls ST.IList) {
	ls = make([]*ST.IItem, length, length)
	for i := 0; i < length; i++ {
		ls[i] = i2iItem(i)
	}
	return
}

func TestManipulator(t *testing.T) {
	b := BackoffDeleteLowFrequencyItems{}
	b.init(nil, 5, StepsizeLinear)

	ps := orderedList(10)
	ls := ST.IList{ps[0], ps[1], ps[3], ps[2], ps[0], ps[2], ps[3], ps[0]}
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
	schema, err := ST.LoadSchemaTree(treePath)

	if err != nil {
		t.Errorf("Schematree could not be loaded")
	}
	pMap := schema.PropMap
	b := BackoffDeleteLowFrequencyItems{}
	b.init(schema, 1, StepsizeLinear)
	c := make(chan chanObject, 1)

	prop1, _ := pMap["http://www.wikidata.org/prop/direct/P31"]
	prop2, _ := pMap["http://www.wikidata.org/prop/direct/P21"]
	prop3, _ := pMap["http://www.wikidata.org/prop/direct/P27"]
	props := ST.IList{prop1, prop2, prop3}

	removed := []*ST.IItem{}
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
