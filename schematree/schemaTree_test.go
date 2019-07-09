package schematree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadSchemaTree(t *testing.T) {

	t.Run("TypedSchemaTree", func(t *testing.T) {
		tree, _ := LoadSchemaTree("../testdata/10M.nt.gz.schemaTree.typed.bin")
		assert.EqualValues(t, 1497, len(tree.PropMap))
		assert.EqualValues(t, 1, tree.MinSup)
		assert.True(t, tree.Typed)

	})
	t.Run("UnTypedSchemaTree", func(t *testing.T) {
		tree, _ := LoadSchemaTree("../testdata/10M.nt.gz.schemaTree.bin")
		assert.EqualValues(t, 1242, len(tree.PropMap))
		assert.EqualValues(t, 1, tree.MinSup)
		assert.False(t, tree.Typed)
	})

}
