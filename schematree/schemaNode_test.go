package schematree

import "testing"

// func TestTimeConsuming(t *testing.T) {
//     if testing.Short() {
//         t.Skip("skipping test in short mode.")
//     }
//     ...
// }

func emptyRootNodeTest(root schemaNode, t *testing.T) {
	if root.ID == nil {
		t.Error("schemaNode ID is nil")
	}

	if *root.ID.Str != "root" {
		t.Error("iri of root node is not \"root\"")
	}

	if root.parent != nil {
		t.Error("parent of root not nil")
	}

	if len(root.Children) != 0 {
		t.Error("root node should be created with empty child array")
	}
}

func TestSchemaTree(t *testing.T) {
	tree := NewSchemaTree()

	t.Run("Root is a proper empty root node", func(t *testing.T) { emptyRootNodeTest(tree.Root, t) })

}
