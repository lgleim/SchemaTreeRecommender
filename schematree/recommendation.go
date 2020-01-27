package schematree

import (
	"fmt"
	"sort"
)

// RankedPropertyCandidate is a struct to rank suggestions
type RankedPropertyCandidate struct {
	Property    *IItem
	Probability float64
}

// PropertyRecommendations is a list of RankedPropertyCandidates
type PropertyRecommendations []RankedPropertyCandidate

// String returns the string representation of property candidates
func (ps PropertyRecommendations) String() string {
	s := ""
	for _, p := range ps {
		s += fmt.Sprintf("%v: %v\n", *p.Property.Str, p.Probability)
	}
	return s
}

// Top10AvgProbibility calculates average of probability of the top ten recommendations
// if less than 10 recommendations, then missing values have probability 0
func (ps PropertyRecommendations) Top10AvgProbibility() float32 {
	var sum float64
	for i := 0; i < 10; i++ {
		if i < len(ps) {
			sum += ps[i].Probability
		}
	}
	return float32(sum) / 10.0
}

// Recommend recommends a ranked list of property candidates by given strings
// Note: This method should be used in the future where assessments have no access to IItem.
func (tree *SchemaTree) Recommend(properties []string, types []string) PropertyRecommendations {

	list := tree.BuildPropertyList(properties, types)

	// Run the SchemaTree recommender
	var candidates PropertyRecommendations
	candidates = tree.RecommendProperty(list)

	return candidates
}

// BuildPropertyList receives prop and type strings, and builds a list of IItem from it that can later
// be used to execute the recommender.
func (tree *SchemaTree) BuildPropertyList(properties []string, types []string) IList {

	list := []*IItem{}
	// Find IItems of property strings
	for _, pString := range properties {
		p, ok := tree.PropMap[pString]
		if ok {
			list = append(list, p)
		}
	}

	// Find IItems of type strings
	for _, tString := range types {
		tString := "t#" + tString
		p, ok := tree.PropMap[tString]
		if ok {
			list = append(list, p)
		}
	}

	return list
}

// RecommendProperty recommends a ranked list of property candidates by given IItems
func (tree *SchemaTree) RecommendProperty(properties IList) (ranked PropertyRecommendations) {

	if len(properties) > 0 {

		properties.Sort() // descending by support

		pSet := properties.toSet()

		candidates := make(map[*IItem]uint32)

		var makeCandidates func(startNode *SchemaNode)
		makeCandidates = func(startNode *SchemaNode) { // head hunter function ;)
			for _, child := range startNode.Children {
				if child.ID.IsProp() {
					candidates[child.ID] += child.Support
				}
				makeCandidates(child)
			}
		}

		// the least frequent property from the list is farthest from the root
		rarestProperty := properties[len(properties)-1]

		var setSupport uint64
		// walk from each "leaf" instance of that property towards the root...
		for leaf := rarestProperty.traversalPointer; leaf != nil; leaf = leaf.nextSameID { // iterate all instances for that property
			if leaf.prefixContains(properties) {
				setSupport += uint64(leaf.Support) // number of occuences of this set of properties in the current branch

				// walk up
				for cur := leaf; cur.parent != nil; cur = cur.parent {
					if !(pSet[cur.ID]) {
						if cur.ID.IsProp() {
							candidates[cur.ID] += leaf.Support
						}
					}
				}
				// walk down
				makeCandidates(leaf)
			}
		}

		// now that all candidates have been collected, rank them
		i := 0
		setSup := float64(setSupport)
		ranked = make([]RankedPropertyCandidate, len(candidates), len(candidates))
		for candidate, support := range candidates {
			ranked[i] = RankedPropertyCandidate{candidate, float64(support) / setSup}
			i++
		}

		// sort descending by support
		sort.Slice(ranked, func(i, j int) bool { return ranked[i].Probability > ranked[j].Probability })
	} else {
		// TODO: Race condition on propMap: fatal error: concurrent map iteration and map write
		// fmt.Println(tree.Root.Support)
		setSup := float64(tree.Root.Support) // empty set occured in all transactions
		ranked = make([]RankedPropertyCandidate, len(tree.PropMap), len(tree.PropMap))
		for _, prop := range tree.PropMap {
			ranked[int(prop.SortOrder)] = RankedPropertyCandidate{prop, float64(prop.TotalCount) / setSup}
		}
	}

	return
}

// RecommendPropertiesAndTypes recommends a ranked list of property and type candidates by given IItems
func (tree *SchemaTree) RecommendPropertiesAndTypes(properties IList) (ranked PropertyRecommendations) {

	if len(properties) > 0 {

		properties.Sort() // descending by support

		pSet := properties.toSet()

		candidates := make(map[*IItem]uint32)

		var makeCandidates func(startNode *SchemaNode)
		makeCandidates = func(startNode *SchemaNode) { // head hunter function ;)
			for _, child := range startNode.Children {
				candidates[child.ID] += child.Support
				makeCandidates(child)
			}
		}

		// the least frequent property from the list is farthest from the root
		rarestProperty := properties[len(properties)-1]

		var setSupport uint64
		// walk from each "leaf" instance of that property towards the root...
		for leaf := rarestProperty.traversalPointer; leaf != nil; leaf = leaf.nextSameID { // iterate all instances for that property
			if leaf.prefixContains(properties) {
				setSupport += uint64(leaf.Support) // number of occuences of this set of properties in the current branch

				// walk up
				for cur := leaf; cur.parent != nil; cur = cur.parent {
					if !(pSet[cur.ID]) {
						candidates[cur.ID] += leaf.Support
					}
				}
				// walk down
				makeCandidates(leaf)
			}
		}

		// now that all candidates have been collected, rank them
		i := 0
		setSup := float64(setSupport)
		ranked = make([]RankedPropertyCandidate, len(candidates), len(candidates))
		for candidate, support := range candidates {
			ranked[i] = RankedPropertyCandidate{candidate, float64(support) / setSup}
			i++
		}

		// sort descending by support
		sort.Slice(ranked, func(i, j int) bool { return ranked[i].Probability > ranked[j].Probability })
	} else {
		// TODO: Race condition on propMap: fatal error: concurrent map iteration and map write
		// fmt.Println(tree.Root.Support)
		setSup := float64(tree.Root.Support) // empty set occured in all transactions
		ranked = make([]RankedPropertyCandidate, len(tree.PropMap), len(tree.PropMap))
		for _, prop := range tree.PropMap {
			ranked[int(prop.SortOrder)] = RankedPropertyCandidate{prop, float64(prop.TotalCount) / setSup}
		}
	}

	return
}

// func (tree *schemaTree) recommendType(properties iList) typeRecommendations {
// 	var setSupport uint32
// 	//tree.root.support // empty set occured in all transactions

// 	properties.sort() // descending by support

// 	pSet := properties.toSet()

// 	candidates := make(map[*iItem]uint32)

// 	var makeCandidates func(startNode *SchemaNode)
// 	makeCandidates = func(startNode *SchemaNode) { // head hunter function ;)
// 		for _, child := range startNode.children {
// 			candidates[child.ID] += child.support
// 			makeCandidates(child)
// 		}
// 	}

// 	// walk from each leaf towards root...l
// 	for leaf := properties[len(properties)-1].traversalPointer; leaf != nil; leaf = leaf.nextSameID {
// 		if leaf.prefixContains(&properties) {
// 			setSupport += leaf.support // number of occuences of this set of properties in the current branch
// 			for cur := leaf; cur.parent != nil; cur = cur.parent {
// 				if !(pSet[cur.ID]) {
// 					candidates[cur.ID] += leaf.support
// 				}
// 			}
// 			makeCandidates(leaf)
// 		}
// 	}

// 	// TODO: If there are no candidates, consider doing (n-1)-gram smoothing over property subsets

// 	// now that all candidates have been collected, rank them
// 	ranked := make([]rankedCandidate, 0, len(candidates))
// 	for candidate, support := range candidates {
// 		ranked = append(ranked, rankedCandidate{candidate, float64(support) / float64(setSupport)})
// 	}

// 	// sort descending by support
// 	sort.Slice(ranked, func(i, j int) bool { return ranked[i].probability > ranked[j].probability })

// 	return ranked
// }
