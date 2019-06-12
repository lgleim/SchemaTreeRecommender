package schematree

import (
	"errors"
	"math"
)

type backoffDeleteLowFrequencyItems struct {
	tree               *SchemaTree
	parallelExecutions int
	stepsize           func(int, int, int) int // Stepsize function
}

// step size function literals for defining how many items should be removed.
// linear removal: f(x) = x
var stepsizeLinear = func(size, iterator, parallelExecutions int) int {
	return iterator
}

// proportional removal up to 80% of all items
var stepsizeProportional = func(size, iterator, parallelExecutions int) int {
	return int(math.Round(0.8 * float64(iterator) / float64(parallelExecutions) * float64(size)))
}

func (strat *backoffDeleteLowFrequencyItems) init(pTree *SchemaTree, pParallelExecutions int, pStepsize func(int, int, int) int) {
	strat.tree = pTree
	strat.parallelExecutions = pParallelExecutions
	strat.stepsize = pStepsize
}

//Recommend a propertyRecommendations list with the delete low Frequency Property Backoff strategy
func (strat *backoffDeleteLowFrequencyItems) recommend(propertyList IList) (ranked PropertyRecommendations) {
	sublists, removelists := strat.split(propertyList)
	ranked = strat.recommendInParrallel(sublists, removelists)
	return
}

func (strat *backoffDeleteLowFrequencyItems) split(propertyList IList) (sublists, removelists []IList) {
	//  sort the list according to support
	propertyList.Sort()

	// Create sublists and removelists to track the created sublists and what was removed
	sublists = make([]IList, strat.parallelExecutions, strat.parallelExecutions)
	removelists = make([]IList, strat.parallelExecutions, strat.parallelExecutions)

	//create the subsets according to the sebsize function. When the stepsize exeeded the limit of the list no sublist for that stepsize will be constructed.
	for i := 0; i < strat.parallelExecutions; i++ {
		stepsize := strat.stepsize(i, len(propertyList), strat.parallelExecutions)
		s, r, err := strat.manipulate(propertyList, stepsize)
		if err == nil {
			sublists[i] = s
			removelists[i] = r
		} else {
			sublists = sublists[:len(sublists)-1]
			removelists = sublists[:len(removelists)-1]
		}
	}
	return
}

//Delete the last i ites in the property list, by slicing. No values of the underlying array are touched. If len(propertyList) is smaller than i then an error will be returned
func (strat *backoffDeleteLowFrequencyItems) manipulate(propertyList IList, i int) (reducedPropertyList, removedPropertyList IList, err error) {
	if len(propertyList) < i {
		reducedPropertyList = nil
		err = errors.New("Invalid manipulation of the property list since property list is too short")
	} else {
		reducedPropertyList = propertyList[:len(propertyList)-i]
		removedPropertyList = propertyList[len(propertyList)-i:]
		err = nil
	}
	return
}

// Object which is used to communicate between this goroutine here and the started subroutines.
type chanObject struct {
	recommendations PropertyRecommendations
	subprocess      int
}

// executed the recommender on the sublists in parallel and returns that property recommendation on the largest subset which satisfies the used Condition (Enabler)
// Integrating the enabler is still TODO
func (strat *backoffDeleteLowFrequencyItems) recommendInParrallel(sublists, removelists []IList) PropertyRecommendations {
	rankedList := make([]PropertyRecommendations, len(sublists))

	c := make(chan chanObject, len(sublists))
	// Start recommenders
	for i, items := range sublists {
		go strat.execRecommender(items, removelists[i], i, c)
	}

	// Wait for response. For #sublist many.
	var j int // = 0 // traverse from lowest number of deletion to highest number of deletion item sets
	for {
		rec := <-c
		rankedList[rec.subprocess] = rec.recommendations
		//fmt.Println("Arrive", rec.subprocess)
		// work if the last observed elemnt is returned from go routine.
		for rankedList[j] != nil {
			// TODO integrate condition for good recommendation here (replace false here)
			// Take that recommendation where less items were removed and that satisfies the condition for a good recommendation.
			// If non applies than return
			//fmt.Println("Test:", j)
			if j == len(sublists)-1 || false {
				return rankedList[j]
			}
			j++
		}
	}
}

func (strat *backoffDeleteLowFrequencyItems) execRecommender(items IList, removelist IList, subprocess int, c chan chanObject) {
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
