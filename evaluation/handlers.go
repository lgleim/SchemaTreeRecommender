package main

import (
	"recommender/schematree"
	"sync"
)

// Handlers are methods that receive a set of properties (in the form of a subject summary -
// all properties for a single subject), split those properties into
// pairs (one or multiple) of { reduced properties, left out properties } and then call
// an callback method for each of those pairs in order to generate an evalResult for each.
// Handlers can differ in the way they perform that split of properties.
type handlerFunc func(*schematree.SubjectSummary, func(schematree.IList, schematree.IList) *evalResult) []*evalResult

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
		leftOut = make(schematree.IList, len(properties)-countTypes, len(properties)-countTypes)
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

	// Count the number of types and non-types. This is an optimization to speed up
	// the subset generation.
	countTypes := 0
	for property := range summary.Properties {
		if property.IsType() {
			countTypes++
		}
	}

	// Create and fill both subsets
	reducedProps := make(schematree.IList, 0, countTypes)
	leftOutProps := make(schematree.IList, 0, len(summary.Properties)-countTypes)
	for property := range summary.Properties {
		if property.IsType() {
			reducedProps = append(reducedProps, property)
		} else {
			leftOutProps = append(leftOutProps, property)
		}
	}

	// Only one result is generated for this handler. If no types properties exist, then
	// the evaluator will return nil.
	result := evaluator(reducedProps, leftOutProps)
	if result != nil {
		result.note = summary.Str    // @TODO: Temporarily added to aid in evaluation debugging
		return []*evalResult{result} // return an array of a single result
	}
	return []*evalResult{} // return an empty array of results
}

// handlerTake1N is a handler method that, upon receiving a subject summary,
// it will use leave-one-out (or jackknife resampling) to generate and evalResults
// for every property that is left out.
//
// @TODO: Take a good look at this. Somehow this handler accepted a `isTyped` argument
//        and it is unclear what reason there is. Perhaps, this is actually a hint to
//        create two handlers out of this one: TakeOneButType and TakeOne
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

// @TODO: This code is how it originally was done, by evaluating every left out
//        property in isolation.
