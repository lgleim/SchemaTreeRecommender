package schematree

type backoffSplitPropertySet struct {
	tree     *SchemaTree
	splitter func(IList) []IList                                     // Split the property list
	merger   func([]PropertyRecommendations) PropertyRecommendations // Merge the property list
}

//splits into two sublists. "Equal" mixture of high and low support properties in both sets.
var everySecondItemSplitter = func(properties IList) (sublists []IList) {
	properties.Sort()
	sublists = make([]IList, 2, 2)
	for i, p := range properties {
		if i%2 == 0 {
			sublists[0] = append(sublists[0], p)
		} else {
			sublists[1] = append(sublists[1], p)
		}
	}
	return
}

// splits the data set into two equally sized sublists, one containing all the high support properties, and one all the low support properties.
var twoSupportRangesSplitter = func(properties IList) (sublists []IList) {
	properties.Sort()
	sublists = make([]IList, 2, 2)
	mid := int(float64(len(properties)) / 2.0)
	sublists[0] = properties[mid:]
	sublists[1] = properties[:mid]
	return
}

// just chooses the first recommendation as final recommendation
var dummyMerger = func(recommendations []PropertyRecommendations) (merged PropertyRecommendations) {
	merged = recommendations[0]
	return
}

// init the backoff strategy. needed ist a schematree, a splitter function that splits the property list into sublists, and a merger which then merges the recommendations on the sublists
func (strat *backoffSplitPropertySet) init(pTree *SchemaTree, pSplitter func(IList) []IList, pMerger func([]PropertyRecommendations) PropertyRecommendations) {
	strat.tree = pTree
	strat.splitter = pSplitter
	strat.merger = pMerger
}

//Recommend a propertyRecommendations list with the delete low Frequency Property Backoff strategy
func (strat *backoffSplitPropertySet) recommend(propertyList IList) (ranked PropertyRecommendations) {
	sublists := strat.splitter(propertyList)
	recommendations := strat.recommendInPrallel(sublists)
	ranked = strat.merger(recommendations)
	return
}

// run several instances of the recommender in parallel on the sublists. Result are several recommendations
func (strat *backoffSplitPropertySet) recommendInPrallel(sublists []IList) (recommendations []PropertyRecommendations) {

	recommendations = make([]PropertyRecommendations, len(sublists), len(sublists))

	// merge all other recommendations as 'removed recommendations'
	// Maybe not the most efficient way...
	mergeRemoved := func(sublists []IList, current int) (merged IList) {
		merged = make(IList, 0, 0)
		for i, list := range sublists {
			// leave out currently views sublist
			if i != current {
				//fmt.Print(list.String())
				merged = append(merged, list...)
			}
		}
		return
	}

	c := make(chan chanObject, len(sublists))
	//start routines
	for i, list := range sublists {
		removed := mergeRemoved(sublists, i)
		strat.execRecommender(list, removed, i, c)
	}

	// wait for result
	var res chanObject
	for range sublists {
		res = <-c
		recommendations[res.subprocess] = res.recommendations
	}
	return
}

// TODO WHEN RESTRUCTURE: file deleteLowFrequencyProperty got the exactly same function! SHARE!
func (strat *backoffSplitPropertySet) execRecommender(items IList, removelist IList, subprocess int, c chan chanObject) {
	// Compute Recommendation for the subset
	recommendation := strat.tree.RecommendProperty(items)
	// Delete those items which were recommended but were actually deleted before.
	// OPT: Optimize Runtime here (O(n^2) to O(n*log(n) by first sorting and then efficient compare))
	for _, r := range removelist {
		for i, item := range recommendation {
			if *item.Property.Str == *r.Str { // https://yourbasic.org/golang/delete-element-slice/
				copy(recommendation[i:], recommendation[i+1:])                    // Shift recommendation[i+1:] left one index.
				recommendation[len(recommendation)-1] = RankedPropertyCandidate{} // Erase last element (write zero value).
				recommendation = recommendation[:len(recommendation)-1]
				break
			}
		}
	}
	res := chanObject{recommendation, subprocess}
	c <- res
}
