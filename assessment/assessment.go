package assessment

import (
	"recommender/schematree"
)

// Instance - An assessment on properties
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
