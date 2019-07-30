package main

import (
	"fmt"
	"math"
	"os"
	"recommender/assessment"
	"recommender/schematree"
	"recommender/strategy"
	"sort"
	"sync"
	"time"
)

type evalResult struct {
	setSize    uint16 // number of properties used to generate recommendations (both type and non-type)
	numTypes   uint16 // number of type properties in the property set
	numLeftOut uint16 // number of properties that have been left out an needed to be recommended back
	rank       uint32 // rank calculated for recommendation, equal to lec(recommendations)+1 if not fully recommendated back
	numTP      uint32 // confusion matrix - number of left out properties that have been recommended
	numFP      uint32 // confusion matrix - number of recommendations that have not been left out
	numTN      uint32 // confusion matrix - number of properties that have neither been recommended or left out
	numFN      uint32 // confusion matrix - number of properties that are left out but have not been recommended
	duration   int64  // duration (in nanoseconds) of how long the recommendation took
	group      uint16 // extra value that can store values like custom-made groups
}

type evalSummary struct {
	setSize             int
	median              float64
	mean                float64
	stddev              float64
	top1                float64
	top5                float64
	top10               float64
	sampleSize          int
	subjectCount        float64
	worst5average       float64
	duration            float64
	hitRate             float64
	precision           float64
	precisionAt10       float64
	recommendationCount float64
}

// evaluatePair will generate an evalResult for a pair of ( reducedProps , leftOutProps ).
// This function will take a list of reduced properties, run the recommender workflow with
// those reduced properties, generate evaluation result entries by using the recently adquired
// recommendations and the leftout properties.
// The aim is to evaluate how well the leftout properties appear in the recommendations that are
// generated using the reduced set of properties (from where the properties have been left out).
func evaluatePair(
	tree *schematree.SchemaTree,
	workflow *strategy.Workflow,
	reducedProps schematree.IList,
	leftOutProps schematree.IList,
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
	var numTP, numFP, numFN, numTN uint32
	for _, lop := range leftOutProps {

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
			numTP++ // in practice this is also the number of matches
			if matchIndex > maxMatchIndex {
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
	if numTP == uint32(len(leftOutProps)) {
		rank = uint32(maxMatchIndex + 1 - len(leftOutProps))
	} else {
		rank = uint32(len(recs) + 1) // could be 10000 too
	}

	// Prepare the full evalResult by deriving some values.
	result := evalResult{
		setSize:    uint16(len(reducedProps)),
		numTypes:   numTypeProps,
		numLeftOut: uint16(len(leftOutProps)),
		rank:       rank,
		numTP:      numTP,
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
	} else if evalMethod == "handlerTakeButType" { // take all but types
		handler = handlerTakeButType
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
			resultQueue <- *res // send structs to channel (not pointers)
		}
	}

	// Start the subject summary reader and collect all results into resultList, using the
	// process that is managing the resultQueue.
	schematree.SubjectSummaryReader(filePath, tree.PropMap, subjectCallback, 0, isTyped)
	close(resultQueue)     // mark the end of results channel
	resultWaitGroup.Wait() // wait until the parallel process that manages the queue is terminated

	return resultList
}

// makeStatics receive a list of evaluation results and makes a summary of them.
func makeStatistics(results []evalResult) (statistics []evalSummary) {

	// Legacy step to guarantee that this function works the same.
	// Before, these variables were given by arguments but now the makeStatistic only receives the
	// array of evalResults and the variables have to be calculated here.
	//
	// In order to implement new features this method should be replaced with other code.
	durations := make(map[uint16][]int64)
	stats := make(map[uint16][]uint32)
	hitRates := make(map[uint16][]bool)
	recommendationCounts := make(map[uint16][]uint32)
	for _, res := range results {

		// Basic grouping mechanism just for compatibility. Will use roundID is provided, else setSize.
		var group uint16
		if res.group != 0 {
			group = res.group
		} else {
			group = res.setSize
		}

		stats[0] = append(stats[0], res.rank)
		stats[group] = append(stats[group], res.rank)
		durations[0] = append(durations[0], res.duration)
		durations[group] = append(durations[group], res.duration)
		hitRates[0] = append(hitRates[0], uint32(res.numLeftOut) == res.numTP)
		hitRates[group] = append(hitRates[group], uint32(res.numLeftOut) == res.numTP)
		recommendationCounts[0] = append(recommendationCounts[0], res.numTP+res.numFP)
		recommendationCounts[group] = append(recommendationCounts[group], res.numTP+res.numFP)
	}

	// compute statistics
	duration := make(map[uint16]float64)
	recommendationCount := make(map[uint16]float64)

	for k, v := range durations {
		for _, res := range v {
			duration[k] = duration[k] + float64(res)
		}
		duration[k] = duration[k] / float64(len(v))
	}

	for k, v := range recommendationCounts {
		if k > 0 {
			for _, res := range v {
				recommendationCount[k] = recommendationCount[k] + float64(res)
			}
			recommendationCount[k] = recommendationCount[k] / float64(len(v))
			recommendationCount[0] = recommendationCount[0] + recommendationCount[k]
		}
	}

	statistics = make([]evalSummary, len(stats))
	setLens := make([]int, 0, len(stats))
	for setLen := range stats {
		setLens = append(setLens, int(setLen))
	}

	var averageSize float64
	for _, value := range setLens {
		averageSize += float64(value)
	}
	averageSize = averageSize / float64(len(setLens))

	sort.Ints(setLens)
	for i, setLen := range setLens {

		v := stats[uint16(setLen)]
		h := hitRates[uint16(setLen)]
		r := recommendationCount[uint16(setLen)]
		d := duration[uint16(setLen)]

		if len(v) == 0 {
			continue
		}
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })

		var sum uint64
		var mean, meanSquare, median, variance, top1, top5, top10, precisionAt10, subjects, worst5average, hitRate, precision float64

		l := float64(len(v))
		top1 = float64(sort.Search(len(v), func(i int) bool { return v[i] > 1 })) / l
		top5 = float64(sort.Search(len(v), func(i int) bool { return v[i] > 5 })) / l
		top10 = float64(sort.Search(len(v), func(i int) bool { return v[i] > 10 })) / l

		hitCount := 0
		for _, hit := range h {
			if hit {
				hitCount++
			}
		}

		hitRate = float64(hitCount) / float64(len(h))

		if r > 0 {
			precision = float64(hitCount) / r
		} else {
			precision = 1
		}

		if setLen == 0 {
			precisionAt10 = float64(sort.Search(len(v), func(i int) bool { return v[i] > 10 })) / 10 / float64(len(setLens))
		} else {
			precisionAt10 = float64(sort.Search(len(v), func(i int) bool { return v[i] > 10 })) / math.Min(10, float64(len(v)))
		}

		if len(v) == 1 {
			mean = float64(v[0])
			median = mean
			variance = 0
			worst5average = mean

		} else {
			if len(v)%2 != 0 {
				median = float64(v[len(v)/2])
			} else {
				median = (float64(v[len(v)/2-1]) + float64(v[len(v)/2])) / 2.0
			}

			for _, x := range v {
				sum += uint64(x)
				meanSquare += float64(x) * float64(x) / l
			}
			mean = float64(sum) / l
			variance = meanSquare - (mean * mean)

			worst5 := v[len(v)-int(len(v)/100):]
			if len(worst5) == 0 {
				worst5 = append(worst5, 0)
			}
			sum = 0
			for _, value := range worst5 {
				sum += uint64(value)
			}
			worst5average = float64(sum) / float64(len(worst5))
		}

		if setLen == 0 {
			subjects = float64(len(v)) / averageSize
		} else {
			subjects = float64(len(v)) / float64(setLen)
		}

		statistics[i] = evalSummary{setLen, median, mean, math.Sqrt(variance), top1, top5, top10, len(v), subjects, worst5average, d, hitRate, precision, precisionAt10, r}
	}
	return
}

func writeStatisticsToFile(filename string, statistics []evalSummary) { // compute statistics
	f, _ := os.Create(filename + ".csv")
	f.WriteString(fmt.Sprintf(
		"%8s, %8s, %10s, %9s, %8s, %8s, %8s, %10s, %11s, %8s, %8s, %8s, %13s, %19s\n",
		"set", "median", "mean", "stddev",
		"top1", "top5", "top10", "sampleSize",
		"#subjects", "Duration", "HitRate", "Precision",
		"PrecisionAt10", "RecommendationCount",
	))

	for _, stat := range statistics {
		f.WriteString(fmt.Sprintf(
			"%8v, %8.1f, %10.4f, %9.4f, %8.4f, %8.4f, %8.4f, %10v, %11.4f, %8.4f, %8.4f, %8.4f, %13.4f, %19.4f\n",
			stat.setSize, stat.median, stat.mean, stat.stddev,
			stat.top1*100, stat.top5*100, stat.top10*100, stat.sampleSize,
			stat.subjectCount, float64(stat.duration)/1000000, stat.hitRate*100.0, stat.precision,
			stat.precisionAt10, stat.recommendationCount,
		))
	}
	f.Close()
	return
}
