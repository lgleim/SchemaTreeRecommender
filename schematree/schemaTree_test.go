package schematree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var filePath = "../testdata/test.nt.gz"
var treePath = "../testdata/10M.nt.gz.schemaTree.bin"

func TestSchemaTree(t *testing.T) {
	tree := New(false, 0)
	t.Run("Root is a proper empty root node", func(t *testing.T) { emptyRootNodeTest(t, tree.Root) })

}

// typedTreeTest test a typed schematree generated by the testdata/test.nt.gz file
func typedTreeTest(t *testing.T, tree *SchemaTree) {
	assert.EqualValues(t, 184, len(tree.PropMap))
	assert.EqualValues(t, 1, tree.MinSup)
	assert.True(t, tree.Typed)
}

// typedTreeTest test a untyped schematree generated by the testdata/test.nt.gz file
func untypedTreeTest(t *testing.T, tree *SchemaTree) {
	assert.EqualValues(t, 176, len(tree.PropMap))
	assert.EqualValues(t, 1, tree.MinSup)
	assert.False(t, tree.Typed)
}

func TestCreate(t *testing.T) {

	t.Run("TypedSchemaTree", func(t *testing.T) {
		tree, _ := Create(filePath, 0, true, 1)
		typedTreeTest(t, tree)
	})

	t.Run("UntypedSchemaTree", func(t *testing.T) {
		tree, _ := Create(filePath, 0, false, 1)
		untypedTreeTest(t, tree)
	})
}

func TestTwoPass(t *testing.T) {

	t.Run("typed schematree", func(t *testing.T) {
		tree := New(true, 1)
		tree.TwoPass(filePath, 100)
		typedTreeTest(t, tree)
	})

	t.Run("untyped schematree", func(t *testing.T) {
		tree := New(false, 1)
		tree.TwoPass(filePath, 100)
		untypedTreeTest(t, tree)
	})

}

func TestLoad(t *testing.T) {

	t.Run("TypedSchemaTree", func(t *testing.T) {
		tree, _ := Load("../testdata/10M.nt.gz.schemaTree.typed.bin")
		assert.EqualValues(t, 1497, len(tree.PropMap))
		assert.EqualValues(t, 1, tree.MinSup)
		assert.True(t, tree.Typed)

	})
	t.Run("UnTypedSchemaTree", func(t *testing.T) {
		tree, _ := Load("../testdata/10M.nt.gz.schemaTree.bin")
		assert.EqualValues(t, 1242, len(tree.PropMap))
		assert.EqualValues(t, 1, tree.MinSup)
		assert.False(t, tree.Typed)
	})

}

func TestInsert(t *testing.T) {
	tree, _ := Load("../testdata/10M.nt.gz.schemaTree.typed.bin")

	properties := make(map[*IItem]uint32)
	prop1 := tree.PropMap.get("http://www.wikidata.org/prop/direct/P31")
	rootSup := tree.Root.Support
	p1Sup := tree.Root.getOrCreateChild(prop1).Support
	properties[prop1] = 1
	s := SubjectSummary{properties, "", 1, 0}

	tree.Insert(&s)
	assert.EqualValues(t, rootSup+1, tree.Root.Support)
	assert.Equal(t, "http://www.wikidata.org/prop/direct/P31", *tree.Root.getOrCreateChild(prop1).ID.Str)
	assert.EqualValues(t, p1Sup+1, tree.Root.getOrCreateChild(prop1).Support)

	tree.Insert(&s)
	assert.Less(t, rootSup+1, tree.Root.Support)
	assert.Less(t, p1Sup+1, tree.Root.getOrCreateChild(prop1).Support)
}

func testAddProperty(tree *SchemaTree, str string, totalCount uint64, sortOrder uint32) {
	tree.PropMap.get(str).TotalCount = totalCount
	tree.PropMap.get(str).SortOrder = sortOrder
}

func TestUpdateSortOrder(t *testing.T) {

	tree := New(true, 1)
	testAddProperty(tree, "p1", 3, 3)
	testAddProperty(tree, "p2", 2, 2)
	testAddProperty(tree, "p3", 2, 1)
	testAddProperty(tree, "p4", 1, 2)

	tree.updateSortOrder()
	assert.EqualValues(t, tree.PropMap.get("p1").SortOrder, 0)
	assert.EqualValues(t, tree.PropMap.get("p2").SortOrder, 1)
	assert.EqualValues(t, tree.PropMap.get("p3").SortOrder, 2)
	assert.EqualValues(t, tree.PropMap.get("p4").SortOrder, 3)

}
