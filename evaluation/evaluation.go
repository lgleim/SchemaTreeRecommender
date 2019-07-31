package main

import (
	"recommender/assessment"
	"recommender/schematree"
	"recommender/strategy"
	"sync"
	"time"
)

type evalResult struct {
	setSize    uint16 // number of properties used to generate recommendations (both type and non-type)
	numTypes   uint16 // number of type properties in the property set
	numLeftOut uint16 // number of properties that have been left out an needed to be recommended back
	rank       uint32 // rank calculated for recommendation, equal to lec(recommendations)+1 if not fully recommendated back
	numTP      uint32 // confusion matrix - number of left out properties that have been recommended
	numTPAtL   uint32 // number of left out properties that have been recommended until position L, where L is numLeftOut
	numFP      uint32 // confusion matrix - number of recommendations that have not been left out
	numTN      uint32 // confusion matrix - number of properties that have neither been recommended or left out
	numFN      uint32 // confusion matrix - number of properties that are left out but have not been recommended
	duration   int64  // duration (in nanoseconds) of how long the recommendation took
	group      uint16 // extra value that can store values like custom-made groups
	note       string // @TODO: Temporarily added to aid in evaluation debugging
}

// evaluatePair will generate an evalResult for a pair of ( reducedProps , leftoutProps ).
// This function will take a list of reduced properties, run the recommender workflow with
// those reduced properties, generate evaluation result entries by using the recently adquired
// recommendations and the leftout properties.
// The aim is to evaluate how well the leftout properties appear in the recommendations that are
// generated using the reduced set of properties (from where the properties have been left out).
// Note that 'nil' can be returned.
func evaluatePair(
	tree *schematree.SchemaTree,
	workflow *strategy.Workflow,
	reducedProps schematree.IList,
	leftoutProps schematree.IList,
) *evalResult {

	// Evaluator will not generate stats if no properties exist to make a recommendation.
	if len(reducedProps) == 0 {
		return nil
	}

	// Run the recommender with the input properties.
	start := time.Now()
	asm := assessment.NewInstance(reducedProps, tree, true)
	recs := workflow.Recommend(asm)
	duration := time.Since(start).Nanoseconds()

	// Calculate the statistics for the evalResult

	// Count the number of properties in the reduced set that are types.
	var numTypeProps uint16
	for _, rp := range reducedProps {
		if rp.IsType() {
			numTypeProps++
		}
	}

	// Iterate through the list of left out properties to detect matching recommendations.
	var maxMatchIndex = 0 // indexes always start at zero
	var numTP, numFP, numFN, numTN, numTPAtL uint32
	for _, lop := range leftoutProps {

		// First go through all recommendations and see if a matching property was found.
		var matchFound bool
		var matchIndex int
		for i, rec := range recs {
			if rec.Property == lop { // @todo: check if same pointers
				matchFound = true
				matchIndex = i
				break
			}
		}

		// If the current left-out property has a matching recommendation.
		// Calculating the maxMatchIndex helps in the future to calculate the rank.
		if matchFound {
			numTP++                             // in practice this is also the number of matches
			if matchIndex < len(leftoutProps) { // keep track
				numTPAtL++
			}
			if matchIndex > maxMatchIndex { // keep track of max for later
				maxMatchIndex = matchIndex
			}
		}

		// If the current left-out property does not have a matching recommendation.
		if !matchFound {
			numFN++
		}
	}
	numFP = uint32(len(recs)) - numTP
	numTN = uint32(len(tree.PropMap)) - numTP - numFN - numFP

	// Calculate the rank: the number of non-left out properties that were given before
	// all left-out properties are recommended, plus 1.
	// When all recommendation have been found, we can derive by taking the maximal index
	// of all matches and using the number of matches to find out how many non-matching
	// recommendations exists until that maximal match index.
	// If not recommendations were found, we add a penalizing number.
	var rank uint32
	if numTP == uint32(len(leftoutProps)) {
		rank = uint32(maxMatchIndex + 1 - len(leftoutProps))
	} else {
		// The rank could also be set to = uint32(len(recs) + 1)
		// That would make it dependent on number of recommendations. Problem is, when the
		// recommender returns a small number of recommendations, then the rank is small
		// as well.
		rank = 10000 // uint32(len(recs) + 1)
	}

	// Prepare the full evalResult by deriving some values.
	result := evalResult{
		setSize:    uint16(len(reducedProps)),
		numTypes:   numTypeProps,
		numLeftOut: uint16(len(leftoutProps)),
		rank:       rank,
		numTP:      numTP,
		numTPAtL:   numTPAtL,
		numFN:      numFN,
		numFP:      numFP,
		numTN:      numTN,
		duration:   duration,
	}
	return &result
}

// performEvaluation will produce an evaluation CSV, where a test `dataset` is applied on a
// constructed SchemaTree `tree`, by using the strategy `workflow`.
// A parameter `isTyped` is required to provide for reading the dataset and it has to be synchronized
// with the build SchemaTree model.
// `evalMethod` will set which sampling procedures will be used for the test.
func evaluateDataset(
	tree *schematree.SchemaTree,
	workflow *strategy.Workflow,
	isTyped bool,
	filePath string,
	evalMethod string,
) []evalResult {

	// Initialize required variables for managing all the results with multiple threads.
	resultList := make([]evalResult, 0)
	resultWaitGroup := sync.WaitGroup{}
	resultQueue := make(chan evalResult, 1000) // collect eval results via channel

	// Start a parellel thread to process and results that are received from the handlers.
	go func() {
		resultWaitGroup.Add(1)
		//var roundID uint16
		for res := range resultQueue {
			//roundID++
			//res.group = roundID
			resultList = append(resultList, res)
		}
		resultWaitGroup.Done()
	}()

	// Depending on the evaluation method, we will use a different handler
	var handler handlerFunc
	if evalMethod == "handlerTake1N" { // take one out
		handler = handlerTake1N
	} else if evalMethod == "TakeOneButType" { // take one out except type
		handler = HandlerTakeOneButType
	} else if evalMethod == "TakeAllButNumTypeBest" { // take all best except number of types
		handler = HandlerTakeAllButNumTypeBest
	} else if evalMethod == "handlerTakeButType" { // take all but types
		handler = handlerTakeButType
	} else if evalMethod == "TakeAllButType" {
		handler = handlerTakeAllButType
	} else if evalMethod == "historicTakeButType" { // original workings of take all but types
		handler = buildHistoricHandlerTakeButType()
	} else {
		panic("No suitable handler has been selected.")
	}

	// We also construct the method that will evaluate a pair of property sets.
	evaluator := func(reduced schematree.IList, leftout schematree.IList) *evalResult {
		return evaluatePair(tree, workflow, reduced, leftout)
	}

	// Build the complete callback function for the subject summary reader.
	// Given a SubjectSummary, we use the handlers to split it into reduced and leftout set.
	// Then we evaluate that pair of property sets. At last, we deliver the result to our
	// resultQueue that will aggregate all results (from multiple sources) in a single list.
	subjectCallback := func(summary *schematree.SubjectSummary) {
		var results []*evalResult = handler(summary, evaluator)
		for _, res := range results {
			// for convenience, this will treat 'nil' results so that old handlers don't need
			// to look out for 'nil' results that can be returned by the evaluator()
			if res != nil {
				resultQueue <- *res // send structs to channel (not pointers)
			}
		}
	}

	// Start the subject summary reader and collect all results into resultList, using the
	// process that is managing the resultQueue.
	schematree.SubjectSummaryReader(filePath, tree.PropMap, subjectCallback, 0, isTyped)
	close(resultQueue)     // mark the end of results channel
	resultWaitGroup.Wait() // wait until the parallel process that manages the queue is terminated

	return resultList
}
