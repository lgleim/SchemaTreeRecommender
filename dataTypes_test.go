package main

import (
	"fmt"
	"strconv"
	"testing"
)

func i2iItem(i int) *iItem {
	s := strconv.Itoa(i)
	return &iItem{&s, uint64(i), uint32(i), nil}
}

func orderedList(length int) (ls iList) {
	ls = make([]*iItem, length, length)
	for i := 0; i < length; i++ {
		ls[i] = i2iItem(i)
	}
	return
}
func TestPropertyDeduplication(t *testing.T) {
	ps := orderedList(10)

	ls := iList{ps[0], ps[1], ps[3], ps[2], ps[0], ps[2], ps[3], ps[0]}
	ls.sortAndDeduplicate()
	if len(ls) != 4 || ls[0] != ps[0] || ls[1] != ps[1] || ls[2] != ps[2] || ls[3] != ps[3] {
		t.Error("sortAndDeduplicate not working! Result is " + fmt.Sprint(ls))
	}

	// is := make(map[*iItem]bool)
	// ss := make(map[*string]bool)

	// for _, x := range properties {
	// 	is[x] = true
	// 	ss[x.Str] = true
	// }
	// if len(is) != len(properties) {
	// 	panic("duplicated items in properties")
	// }
	// if len(ss) != len(properties) {
	// 	panic("duplicated strings in properties")
	// }
}
