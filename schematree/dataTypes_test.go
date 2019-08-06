package schematree

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

//// Test Data ////

func i2iItem(i int) *IItem {
	s := strconv.Itoa(i)
	return &IItem{&s, uint64(i), uint32(i), nil}
}

func orderedList(length int) (ls IList) {
	ls = make([]*IItem, length, length)
	for i := 0; i < length; i++ {
		ls[i] = i2iItem(i)
	}
	return
}

func recommendations(length int) (ls PropertyRecommendations) {
	ps := orderedList(10)
	ls = make([]RankedPropertyCandidate, length, length)
	for i := 0; i < length; i++ {
		ls[i] = RankedPropertyCandidate{ps[i], (1 / float64(i+1))}
	}
	return
}

//// IItem ////

func TestIItem(t *testing.T) {
	s := "item"
	t.Run("increment", func(t *testing.T) {
		item := &IItem{&s, 1, 1, nil}
		assert.Equal(t, uint64(1), item.TotalCount)
		item.increment()
		assert.Equal(t, uint64(2), item.TotalCount)
	})
	t.Run("string", func(t *testing.T) {
		item := i2iItem(2)
		assert.Equal(t, "2x\t2 (2)", item.String())
	})
}

//// IList ////

func TestPropertyDeduplication(t *testing.T) {
	ps := orderedList(10)

	ls := IList{ps[0], ps[1], ps[3], ps[2], ps[0], ps[2], ps[3], ps[0]}
	ls.sortAndDeduplicate()
	if len(ls) != 4 || ls[0] != ps[0] || ls[1] != ps[1] || ls[2] != ps[2] || ls[3] != ps[3] {
		t.Error("sortAndDeduplicate not working! Result is " + fmt.Sprint(ls))
	}
}

func TestSort(t *testing.T) {
	items := orderedList(3)
	t.Run("sort", func(t *testing.T) {
		l := IList{items[1], items[0], items[2]}
		l.Sort()
		assert.Equal(t, len(l), 3)
		assert.Equal(t, items[0], l[0])
		assert.Equal(t, items[1], l[1])
		assert.Equal(t, items[2], l[2])
	})
}

func TestToSet(t *testing.T) {
	ps := orderedList(10)
	t.Run("toSet", func(t *testing.T) {
		pSet := ps.toSet()
		assert.Equal(t, 10, len(pSet))
		for _, item := range pSet {
			assert.Equal(t, true, item)
		}
	})
}

func TestIListString(t *testing.T) {
	ps := orderedList(10)
	t.Run("String", func(t *testing.T) {
		str := ps.String()
		assert.Equal(t, "[ 0 1 2 3 4 5 6 7 8 9 ]", str)
	})
}

// Property Recommendations
func TestPropertyRecommendations(t *testing.T) {
	ps := recommendations(5)
	t.Run("String", func(t *testing.T) {
		str := ps.String()
		assert.Equal(t, "0: 1\n1: 0.5\n2: 0.3333333333333333\n3: 0.25\n4: 0.2\n", str)
	})
}
