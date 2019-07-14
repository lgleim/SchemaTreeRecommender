package assessment

import (
	"recommender/schematree"
)

// Instance - An assessment on properties
//
// COMMENT: The assessment module might be too tighly coupled with the SchemaTree. In the
//          early days the SchemaTree delegate the job of creating the input arguments to
//          the caller of the method. The caller would create the IList and then pass it
//          to the SchemaTree. This behaviour should be avoid because it makes too much
//          internal information visible to the outside. The correct behaviour is to accept
//          an array of strings (or byte-arrays) and then construct the IList oneself.
//          To fix this issue in the future, please consider making IList and IItem private
//          and then use the schematree.Recommend(props []string, types []string).
//          Likewise, assessments should be working with arrays of strings and not IItems.
//          The benefit in the current method is a faster evaluation since the IList
//          construction does not need to be done multiple times.
type Instance struct {
	Props                 schematree.IList
	tree                  *schematree.SchemaTree
	useOptimisticCache    bool // using cache will make an optimistic assumption that `props` are not altered
	cachedRecommendations schematree.PropertyRecommendations
}

// NewInstance : constructor method
func NewInstance(argProps schematree.IList, argTree *schematree.SchemaTree, argUseCache bool) *Instance {
	return &Instance{
		Props:                 argProps,
		tree:                  argTree,
		useOptimisticCache:    argUseCache,
		cachedRecommendations: nil,
	}
}

// NewInstanceFromInput : constructor method to receive strings and convert them into the current
// assessment format that uses IList.
func NewInstanceFromInput(argProps []string, argTypes []string, argTree *schematree.SchemaTree, argUseCache bool) *Instance {
	propList := argTree.BuildPropertyList(argProps, argTypes)

	return &Instance{
		Props:                 propList,
		tree:                  argTree,
		useOptimisticCache:    argUseCache,
		cachedRecommendations: nil,
	}
}

// CalcPropertyLength : Calculate the amount of properties.
func (inst *Instance) CalcPropertyLength() int {
	return len(inst.Props)
}

// CalcRecommendations : Will execute the core schematree recommender on the properties and return
// the list of recommendations. Cache-enabled operation.
func (inst *Instance) CalcRecommendations() schematree.PropertyRecommendations {
	if inst.useOptimisticCache == true {
		if inst.cachedRecommendations == nil {
			inst.cachedRecommendations = inst.tree.RecommendProperty(inst.Props)
		}
		return inst.cachedRecommendations
	}
	return inst.tree.RecommendProperty(inst.Props)
}
