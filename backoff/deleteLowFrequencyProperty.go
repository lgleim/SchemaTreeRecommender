package backoff

import (
	"errors"
	"math"
	ST "recommender/schematree"
)

type StepsizeFunc func(int, int, int) int

type InternalCondition func(*ST.PropertyRecommendations) bool

type BackoffDeleteLowFrequencyItems struct {
	tree               *ST.SchemaTree
	parallelExecutions int
	stepsize           func(int, int, int) int // Stepsize function
	condition          InternalCondition
}

// step size function literals for defining how many items should be removed.
// linear removal: f(x) = x
var StepsizeLinear = func(size, iterator, parallelExecutions int) int {
	if iterator < size {
		return iterator
	}
	return size - 1
}

func MakeMoreThanInternalCondition(threshold int) InternalCondition {
	return func(recs *ST.PropertyRecommendations) bool {
		if len(*recs) > threshold {
			return true
		}
		return false
	}
}
func MakeMoreThanProbabilityInternalCondition(threshold float32) InternalCondition {
	return func(recs *ST.PropertyRecommendations) bool {
		if recs.Top10AvgProbibility() > threshold {
			return true
		}
		return false
	}
}

// proportional removal up to 30% of all items
var StepsizeProportional = func(size, iterator, parallelExecutions int) int {
	return int(math.Round(0.4 * float64(iterator) / float64(parallelExecutions) * float64(size)))
}

// NewBackoffDeleteLowFrequencyItems : constructor method
func NewBackoffDeleteLowFrequencyItems(pTree *ST.SchemaTree, pParallelExecutions int, pStepsize StepsizeFunc, pCondition InternalCondition) *BackoffDeleteLowFrequencyItems {
	return &BackoffDeleteLowFrequencyItems{tree: pTree, parallelExecutions: pParallelExecutions, stepsize: pStepsize, condition: pCondition}
}

func (strat *BackoffDeleteLowFrequencyItems) init(pTree *ST.SchemaTree, pParallelExecutions int, pStepsize func(int, int, int) int) {
	strat.tree = pTree
	strat.parallelExecutions = pParallelExecutions
	strat.stepsize = pStepsize
}

//Recommend a propertyRecommendations list with the delete low Frequency Property Backoff strategy
func (strat *BackoffDeleteLowFrequencyItems) Recommend(propertyList ST.IList) (ranked ST.PropertyRecommendations) {
	sublists, removelists := strat.split(propertyList)
	ranked = strat.recommendInParrallel(sublists, removelists)
	return
}

func (strat *BackoffDeleteLowFrequencyItems) split(propertyList ST.IList) (sublists, removelists []ST.IList) {
	//  sort the list according to support
	propertyList.Sort()

	// Create sublists and removelists to track the created sublists and what was removed
	sublists = make([]ST.IList, strat.parallelExecutions, strat.parallelExecutions)
	removelists = make([]ST.IList, strat.parallelExecutions, strat.parallelExecutions)

	//create the subsets according to the sebsize function. When the stepsize exeeded the limit of the list no sublist for that stepsize will be constructed.
	for i := 0; i < strat.parallelExecutions; i++ {
		stepsize := strat.stepsize(len(propertyList), i+1, strat.parallelExecutions)
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
func (strat *BackoffDeleteLowFrequencyItems) manipulate(propertyList ST.IList, i int) (reducedPropertyList, removedPropertyList ST.IList, err error) {
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
	recommendations ST.PropertyRecommendations
	subprocess      int
}

// executed the recommender on the sublists in parallel and returns that property recommendation on the largest subset which satisfies the used Condition (Enabler)
// Integrating the enabler is still TODO
func (strat *BackoffDeleteLowFrequencyItems) recommendInParrallel(sublists, removelists []ST.IList) ST.PropertyRecommendations {
	rankedList := make([]ST.PropertyRecommendations, len(sublists))

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
			// Take that recommendation where less items were removed and that satisfies the condition for a good recommendation.
			// If non applies than return
			//fmt.Println("Test:", j)
			if (j == len(sublists)-1) || (strat.condition(&rankedList[j])) {
				return rankedList[j]
			}
			j++
		}
	}
}

func (strat *BackoffDeleteLowFrequencyItems) execRecommender(items ST.IList, removelist ST.IList, subprocess int, c chan chanObject) {
	// Compute Recommendation for the subset
	recommendation := strat.tree.RecommendProperty(items)
	// Delete those items which were recommended but were actually deleted before.
	// OPT: Optimize Runtime here (O(n^2) to O(n*log(n) by first sorting and then efficient compare))
	for _, r := range removelist {
		for i, item := range recommendation {
			if *item.Property.Str == *r.Str { // https://yourbasic.org/golang/delete-element-slice/
				copy(recommendation[i:], recommendation[i+1:])                       // Shift recommendation[i+1:] left one index.
				recommendation[len(recommendation)-1] = ST.RankedPropertyCandidate{} // Erase last element (write zero value).
				recommendation = recommendation[:len(recommendation)-1]
				break
			}
		}
	}
	res := chanObject{recommendation, subprocess}
	c <- res
}
