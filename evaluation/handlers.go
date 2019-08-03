package main

import (
	"recommender/schematree"
	"sort"
	"sync"
)

// Handlers are methods that receive a set of properties (in the form of a subject summary -
// all properties for a single subject), split those properties into
// pairs (one or multiple) of { reduced properties, left out properties } and then call
// an callback method for each of those pairs in order to generate an evalResult for each.
// Handlers can differ in the way they perform that split of properties.
type handlerFunc func(*schematree.SubjectSummary, func(schematree.IList, schematree.IList) *evalResult) []*evalResult

// HandlerTakeOneButType will call the evaluator multiple times. Each time it leaves one
// property out and perform a recommendation with all remaining properties. It makes sure
// to never leave out a type property: for untyped tree this will have no effects, and for
// typed trees we actually never want remove the typing information (we consider it
// essential for any system using types).
func HandlerTakeOneButType(
	s *schematree.SubjectSummary,
	evaluator func(schematree.IList, schematree.IList) *evalResult,
) []*evalResult {

	results := make([]*evalResult, 0, len(s.Properties))

	// Terminate early if reduced set will be empty.
	if len(s.Properties)-1 <= 0 {
		return results
	}

	// Fill the reduced set with all properties except the last. Last goes to leftout set.
	reducedSet := make(schematree.IList, len(s.Properties)-1) // assumes len() > 0
	leftoutSet := make(schematree.IList, 1)
	cnt := 0 // keep track of how many keys we iterated through, as we want len-1
	for key := range s.Properties {
		if cnt != len(s.Properties)-1 {
			reducedSet[cnt] = key
			// fmt.Println("[RED] " + *reducedSet[cnt].Str)
		} else {
			leftoutSet[0] = key
			// fmt.Println("[LEF] " + *leftoutSet[0].Str)
		}
		cnt++
	}

	// Iterate through all leave-out-one combinations with the help of tactful swapping.
	for idx := 0; idx < len(s.Properties); idx++ {
		if !leftoutSet[0].IsType() { // Only evaluate if leftout property is not a type property.
			newResult := evaluator(reducedSet, leftoutSet)
			newResult.note = s.Str + " " + *leftoutSet[0].Str
			results = append(results, newResult)
		}
		if idx < len(reducedSet) { // run len(s.Property) times and swap len(s.reducedSet) times
			temp := leftoutSet[0]
			leftoutSet[0] = reducedSet[idx]
			reducedSet[idx] = temp
		}
	}

	return results
}

// HandlerTakeAllButBest will select the reduced set by ordering all properties by their
// "best" criteria { isType() < !isType() < SortOrder } and then pick the first NumBest.
//
// NumBest is defined using the following method:
//   - NumBest = number of type predicates in the subject summary
//   - we required that non-typed trees have at least 1 leftout property:
//     NumBest = min( NumBest, non-type properties in subject summary - 1 )
//   - this might result in a reduced set that is equal-or-less-than zero, those cases
//     will not fire any evaluator
//
// This handler is almost identical to `handlerTakeButType` for typed tree, and is modified to
// also work on subject summaries where the tree is untyped.
//
// Only a single evaluation is done.
func HandlerTakeAllButBest(
	s *schematree.SubjectSummary,
	evaluator func(schematree.IList, schematree.IList) *evalResult,
) []*evalResult {

	results := make([]*evalResult, 0, 1)

	// Terminate early if reduced set will be empty.
	if s.NumTypePredicates <= 0 {
		return results
	}

	// Copy the properties and sort it according to special criteria.
	completeSet := make(schematree.IList, len(s.Properties))
	numNonTypeProps := 0
	cnt := 0
	for key := range s.Properties {
		completeSet[cnt] = key
		if !key.IsType() {
			numNonTypeProps++
		}
		cnt++
	}
	sort.Slice(
		completeSet,
		func(a, b int) bool {
			if completeSet[a].IsType() != completeSet[b].IsType() { // xor
				return completeSet[a].IsType() // only one is true, the first one
			}
			return completeSet[a].SortOrder < completeSet[b].SortOrder
		},
	)

	// Calculate NumBest using the described method. (could have used some min function)
	numBest := s.NumTypePredicates
	if numNonTypeProps-1 < numBest {
		numBest = numNonTypeProps - 1
	}

	// If either the reduced or the leftout set will have no properties, then end early
	if numBest <= 0 || numBest >= len(completeSet) {
		return results
	}

	// Form the reduced and leftout sets using the complete sorted set.
	reducedSet := completeSet[:numBest]
	leftoutSet := completeSet[numBest:]
	newResult := evaluator(reducedSet, leftoutSet)

	// @debug: Write all the reduced set property names
	propNames := ""
	for _, item := range reducedSet {
		propNames = propNames + *item.Str + " "
	}
	newResult.note = s.Str + " ( " + propNames + ")"

	results = append(results, newResult)
	return results
}

// HandlerTakeMoreButCommon will iteratively leave out more and more properties but the most common.
//
// It will order all properties by their "type-aware most common" criteria:
//   { isType() < !isType() < SortOrder }
// It starts with a reduced set that contains all type properties and one non-type property. Then it
// will add a non-type properties to the reduced set, one-by-one until no more properties are left out.
//
// Around NumNonType - 1 evaluations will be done.
func HandlerTakeMoreButCommon(
	s *schematree.SubjectSummary,
	evaluator func(schematree.IList, schematree.IList) *evalResult,
) []*evalResult {

	// Copy the properties and sort it according to special criteria.
	completeSet := make(schematree.IList, len(s.Properties))
	numTypes := 0
	cnt := 0
	for key := range s.Properties {
		completeSet[cnt] = key
		if key.IsType() {
			numTypes++
		}
		cnt++
	}
	sort.Slice(
		completeSet,
		func(a, b int) bool { // return true if a comes before b (a < b)
			if completeSet[a].IsType() != completeSet[b].IsType() { // xor
				return completeSet[a].IsType() // only one is true, the first one
			}
			return completeSet[a].SortOrder < completeSet[b].SortOrder
		},
	)

	// Initialize results with expected capacity
	if len(s.Properties) <= numTypes {
		return make([]*evalResult, 0)
	}
	results := make([]*evalResult, 0, len(s.Properties)-numTypes-1)

	// Start with the smallest reduced set and continue until biggest is achieved.
	var reducedSet schematree.IList
	var leftoutSet schematree.IList
	for sp := numTypes + 1; sp < len(s.Properties); sp++ {
		reducedSet = completeSet[:sp] // reduced is everything before split-point
		leftoutSet = completeSet[sp:] // leftout is everything after and including split-point
		newResult := evaluator(reducedSet, leftoutSet)

		// @debug: Write all the reduced set property names
		propNames := ""
		for _, item := range reducedSet {
			propNames = propNames + *item.Str + " "
		}
		newResult.note = s.Str + " ( " + propNames + ")"

		results = append(results, newResult)
	}

	return results
}

// handleTakeButType is a handler method that, upon receiving a subject summary,
// will call the evaluator with all type properties as reduced properties and all
// others as left out properties.
func handlerTakeButType(
	s *schematree.SubjectSummary,
	evaluator func(schematree.IList, schematree.IList) *evalResult,
) []*evalResult {
	results := make([]*evalResult, 0, 1)

	properties := make(schematree.IList, 0, len(s.Properties))
	for p := range s.Properties {
		properties = append(properties, p)
	}
	properties.Sort()

	countTypes := 0
	for _, property := range properties {
		if property.IsType() {
			countTypes += 1
		}
	}

	var reducedEntitySet schematree.IList
	var leftOut schematree.IList
	if countTypes == 0 {
		return results // returns empty slice
		//if no types, use the third most frequent properties
		//reducedEntitySet = properties[:3]
		//leftOut = properties[3:]
	} else {
		reducedEntitySet = make(schematree.IList, 0, countTypes)
		leftOut = make(schematree.IList, 0, len(properties)-countTypes)
		for _, property := range properties {
			if property.IsType() {
				reducedEntitySet = append(reducedEntitySet, property)
			} else {
				leftOut = append(leftOut, property)
			}
		}
	}
	//here, the entries are not sorted by set size, bzt by this roundID, s.t. all results from one entity are grouped
	results = append(results, evaluator(reducedEntitySet, leftOut))
	return results
}

// handlerTakeAllButType is a handler method that, upon receiving a subject summary,
// will call the evaluator with all type properties as reduced properties and all
// others as left out properties.
// Note: This version has been adapted from `handerlTakeButType` when there were some
// operations with roundID. It should deliver the same results.
func handlerTakeAllButType(
	summary *schematree.SubjectSummary,
	evaluator func(schematree.IList, schematree.IList) *evalResult,
) []*evalResult {
	results := make([]*evalResult, 0, 1)

	// Count the number of types and non-types. This is an optimization to speed up
	// the subset generation.
	countTypes := 0
	for property := range summary.Properties {
		if property.IsType() {
			countTypes++
		}
	}

	// End early if this subject has no types, as recommendation won't be generated without properties.
	if countTypes == 0 {
		return results
	}

	// Create and fill both subsets
	reducedProps := make(schematree.IList, 0, countTypes)
	leftoutProps := make(schematree.IList, 0, len(summary.Properties)-countTypes)
	for property := range summary.Properties {
		if property.IsType() {
			reducedProps = append(reducedProps, property)
		} else {
			leftoutProps = append(leftoutProps, property)
		}
	}

	// Only one result is generated for this handler. If no types properties exist, then
	// the evaluator will return nil.
	res := evaluator(reducedProps, leftoutProps)
	if res != nil {
		res.note = summary.Str // @TODO: Temporarily added to aid in evaluation debugging
		results = append(results, res)
	}
	return results // return an array of one or zero results
}

// handlerTake1N is a handler method that, upon receiving a subject summary,
// it will use leave-one-out (or jackknife resampling) to generate and evalResults
// for every property that is left out.
//
// @todo: This is probably deprecated by `HandlerTakeOneButType`
func handlerTake1N(
	s *schematree.SubjectSummary,
	evaluator func(schematree.IList, schematree.IList) *evalResult,
) (results []*evalResult) {
	results = make([]*evalResult, 0, len(s.Properties))

	properties := make(schematree.IList, 0, len(s.Properties))
	for p := range s.Properties {
		properties = append(properties, p)
	}
	properties.Sort()

	if len(properties) == 0 {
		return
	}

	// take out one property from the list at a time and determine in which position it will be recommended again
	reducedEntitySet := make(schematree.IList, len(properties)-1, len(properties)-1)
	leftOut := make(schematree.IList, 1, 1)
	copy(reducedEntitySet, properties[1:])
	for i := range reducedEntitySet {
		// if !isTyped || properties[i].IsProp() { // Only evaluate if the leftout is a property and not a type
		if properties[i].IsProp() { // Only evaluate if the leftout is a property and not a type
			leftOut[0] = properties[i]
			results = append(results, evaluator(reducedEntitySet, leftOut))
		}
		reducedEntitySet[i] = properties[i]
	}
	// if !isTyped || properties[len(properties)-1].IsProp() {
	if properties[len(properties)-1].IsProp() {
		leftOut[0] = properties[len(properties)-1]
		results = append(results, evaluator(reducedEntitySet, leftOut))
	}
	return
}

// This handler is for compatibility. It uses the original handlerTakeButType, but multiple
// evalResults are generated for each subject: one evalResult for every left out
// property in isolation.
// Running this handler should give the same results as previous versions with handlerTakeButType.
func buildHistoricHandlerTakeButType() handlerFunc {

	roundID := uint16(1)
	var mutex = &sync.Mutex{}

	return func(
		s *schematree.SubjectSummary,
		evaluator func(schematree.IList, schematree.IList) *evalResult,
	) (results []*evalResult) {
		results = make([]*evalResult, 0)

		properties := make(schematree.IList, 0, len(s.Properties))
		for p := range s.Properties {
			properties = append(properties, p)
		}
		properties.Sort()

		countTypes := 0
		for _, property := range properties {
			if property.IsType() {
				countTypes += 1
			}
		}

		var reducedEntitySet schematree.IList
		var leftOutSet schematree.IList
		if countTypes == 0 {
			return // returns empty slice
			//if no types, use the third most frequent properties
			//reducedEntitySet = properties[:3]
			//leftOutSet = properties[3:]
		} else {
			reducedEntitySet = make(schematree.IList, 0, countTypes)
			leftOutSet = make(schematree.IList, 0, len(properties)-countTypes)
			for _, property := range properties {
				if property.IsType() {
					reducedEntitySet = append(reducedEntitySet, property)
				} else {
					leftOutSet = append(leftOutSet, property)
				}
			}
		}

		//here, the entries are not sorted by set size, bzt by this roundID, s.t. all results from one entity are grouped
		mutex.Lock()
		roundID++
		currentRoundID := roundID
		mutex.Unlock()

		boxedLeftOut := make(schematree.IList, 1, 1)
		for _, leftOut := range leftOutSet {
			boxedLeftOut[0] = leftOut
			newResult := evaluator(reducedEntitySet, boxedLeftOut)
			newResult.group = currentRoundID
			if leftOut.Str != nil {
				newResult.note = *leftOut.Str
			} else {
				newResult.note = "NIL"
			}
			if newResult.numTP < uint32(newResult.numLeftOut) {
				newResult.rank = 10000 // penalization was set to 10000
			}
			results = append(results, newResult)
		}

		//here, the entries are not sorted by set size, bzt by this roundID, s.t. all results from one entity are grouped
		return
	}
}
