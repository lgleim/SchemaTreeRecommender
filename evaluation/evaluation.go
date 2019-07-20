package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"recommender/assessment"
	"recommender/configuration"
	"recommender/schematree"
	"recommender/strategy"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sort"
	"sync"
	"time"
)

type evalResult struct {
	setSize             uint16
	position            uint32
	duration            uint64
	hit                 bool
	recommendationCount uint16
}

type evalSummary struct {
	setSize             int
	median              float64
	mean                float64
	variance            float64
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

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile := flag.String("memprofile", "", "write memory profile to `file`")
	traceFile := flag.String("trace", "", "write execution trace to `file`")
	trainedModel := flag.String("model", "", "read stored schematree from `file`")
	configPath := flag.String("workflow", "", "Path to workflow config file for single evaluation")
	testFile := flag.String("testSet", "", "the file to parse")
	batchTest := flag.Bool("batchTest", false, "Switch between batch test and normal test")
	createConfigs := flag.Bool("createConfigs", false, "Create a bunch of config")
	createConfigsCreater := flag.String("creater", "", "Json which defines the creater config file in ./configs")
	numberConfigs := flag.Int("numberConfigs", 1, "CNumber of config files in ./configs")
	typedEntities := flag.Bool("typed", false, "Use type information or not")
	handlerType := flag.String("handler", "handlerTake1N", "Choose the handler handlerTakeButType or handlerTake1N ")

	var statistics []evalSummary

	// parse commandline arguments/flags
	flag.Parse()

	// write cpu profile to file
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	// write cpu profile to file
	if *memprofile != "" {
		defer func() {
			f, err := os.Create(*memprofile)
			if err != nil {
				log.Fatal("could not create memory profile: ", err)
			}
			runtime.GC() // get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatal("could not write memory profile: ", err)
			}
			f.Close()
		}()
	}

	// write cpu profile to file
	if *traceFile != "" {
		f, err := os.Create(*traceFile)
		if err != nil {
			log.Fatal("could not create trace file: ", err)
		}
		if err := trace.Start(f); err != nil {
			log.Fatal("could not start tracing: ", err)
		}
		defer trace.Stop()
	}

	if *createConfigs {
		if *createConfigsCreater == "" {
			log.Fatalln("A Create Config File must be provided in ./configs!")
		}
		createConfigFiles(createConfigsCreater)
	} else if *batchTest {
		// Run all config files and benchmark those. Schematree is taken from ../testdata/10M.nt.gz.schemaTree.bin
		// test data is encoded in the config files
		// Output is csv file in ./
		if *trainedModel == "" {
			log.Fatalln("A model must be provided for Batch Test!")
			return
		}
		err := batchConfigBenchmark(*trainedModel, *numberConfigs, *typedEntities, *handlerType)
		if err != nil {
			log.Fatalln("Batch Config Failed", err)
			return
		}
	} else {

		if *testFile == "" {
			log.Fatalln("A test set must be provided!")
		}

		// evaluation
		if *trainedModel == "" {
			log.Fatalln("A model must be provided!")
		}
		tree, err := schematree.LoadSchemaTree(*trainedModel)
		if err != nil {
			log.Fatalln(err)
		}

		var wf *strategy.Workflow
		if *configPath != "" {
			//load workflow config if given
			config, err := configuration.ReadConfigFile(configPath)
			if err != nil {
				log.Fatalln(err)
			}
			err = config.Test()
			if err != nil {
				log.Fatalln(err)
			}
			wf, err = configuration.ConfigToWorkflow(config, tree)
			if err != nil {
				log.Fatalln(err)
			}
		} else {
			// if no workflow config given then run standard recommender
			wf = strategy.MakePresetWorkflow("direct", tree)
		}
		statistics = evaluation(tree, testFile, wf, typedEntities, *handlerType)
		writeStatisticsToFile(*testFile, statistics)
		fmt.Printf("%v+", statistics[0])
	}
	//so something with statistics
	//fmt.Printf("%v+", statistics[0])
}

func evaluation(tree *schematree.SchemaTree, testFile *string, wf *strategy.Workflow, typed *bool, evalType string) []evalSummary {
	durations := make(map[uint16][]uint64)
	stats := make(map[uint16][]uint32)
	hitRates := make(map[uint16][]bool)
	recommendationCounts := make(map[uint16][]uint16)

	var wg sync.WaitGroup
	roundID := uint16(1)
	results := make(chan evalResult, 1000) // collect eval results via channel

	setSup := float64(tree.Root.Support) // empty set occured in all transactions
	emptyRecs := make([]schematree.RankedPropertyCandidate, len(tree.PropMap), len(tree.PropMap))
	for _, prop := range tree.PropMap {
		emptyRecs[int(prop.SortOrder)] = schematree.RankedPropertyCandidate{
			Property:    prop,
			Probability: float64(prop.TotalCount) / setSup,
		}
	}

	// evaluate the rank the recommender assigns the left out property
	evaluate := func(properties schematree.IList, leftOutList schematree.IList, groupBy uint16) {
		var duration uint64
		var recs []schematree.RankedPropertyCandidate

		if len(properties) == 0 {
			recs = emptyRecs
		} else {
			start := time.Now()
			asm := assessment.NewInstance(properties, tree, true)
			recs = wf.Recommend(asm)
			duration = uint64(time.Since(start).Nanoseconds() / 1000000)
		}

		for _, leftOut := range leftOutList {

			included := false
			for i, r := range recs {
				if r.Property == leftOut { // found item to recover
					//for i > 0 && recs[i-1].Probability == r.Probability {
					//	i--
					//}
					results <- evalResult{groupBy, uint32(i) + 1, duration, true, uint16(len(recs))}
					included = true
					break
				}
			}
			//punish if not in recommendation rec included
			if !included {
				results <- evalResult{groupBy, 10000, duration, false, uint16(len(recs))}
			}
		}
	}

	handlerTakeButType := func(s *schematree.SubjectSummary) {

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
			return
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
		roundID++
		evaluate(reducedEntitySet, leftOut, roundID)
	}

	handlerTake1N := func(s *schematree.SubjectSummary) {
		properties := make(schematree.IList, 0, len(s.Properties))
		for p := range s.Properties {
			properties = append(properties, p)
		}
		properties.Sort()

		// take out one property from the list at a time and determine in which position it will be recommended again
		reducedEntitySet := make(schematree.IList, len(properties)-1, len(properties)-1)
		leftOut := make(schematree.IList, 1, 1)
		copy(reducedEntitySet, properties[1:])
		for i := range reducedEntitySet {
			if !*typed || properties[i].IsProp() { // Only evaluate if the leftout is a property and not a type
				leftOut[0] = properties[i]
				evaluate(reducedEntitySet, leftOut, uint16(len(reducedEntitySet)))
			}
			reducedEntitySet[i] = properties[i]
		}
		if !*typed || properties[len(properties)-1].IsProp() {
			leftOut[0] = properties[len(properties)-1]
			evaluate(reducedEntitySet, leftOut, uint16(len(reducedEntitySet)))
		}
	}

	go func() {
		wg.Add(1)
		for res := range results {
			stats[0] = append(stats[0], res.position)
			stats[res.setSize] = append(stats[res.setSize], res.position)
			durations[0] = append(durations[0], res.duration)
			durations[res.setSize] = append(durations[res.setSize], res.duration)
			hitRates[0] = append(hitRates[0], res.hit)
			hitRates[res.setSize] = append(hitRates[res.setSize], res.hit)
			recommendationCounts[0] = append(recommendationCounts[0], res.recommendationCount)
			recommendationCounts[res.setSize] = append(recommendationCounts[res.setSize], res.recommendationCount)
		}
		wg.Done()
	}()

	if evalType == "handlerTake1N" {
		//take 1 N
		schematree.SubjectSummaryReader(*testFile, tree.PropMap, handlerTake1N, 0, *typed)
	} else if evalType == "handlerTakeButType" {
		//take all but types
		schematree.SubjectSummaryReader(*testFile, tree.PropMap, handlerTakeButType, 0, *typed)
	}

	close(results)
	wg.Wait()

	return makeStatistics(stats, durations, hitRates, recommendationCounts)
}

func makeStatistics(stats map[uint16][]uint32, durations map[uint16][]uint64, hitRates map[uint16][]bool, recommendationCounts map[uint16][]uint16) (statistics []evalSummary) {
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
	output := fmt.Sprintf("%8v, %8v, %8v, %12v, %8v, %8v, %8v, %10v, %10v,%8v, %8v, %8v, %8v,%8v\n", "set", "median", "mean", "stddev", "top1", "top5", "top10", "sampleSize", "#subjects", "Duration", "HitRate", "Precision", "Precision at 10", "Recommendation Count")

	for _, stat := range statistics {
		output += fmt.Sprintf("%8v, %8v, %8.4f, %12.4f, %8.4f, %8.4f, %8.4f, %10v, %10v, %8.4f, %8.4f, %8.4f, %8.4f, %8.4f\n", stat.setSize, stat.median, stat.mean, math.Sqrt(stat.variance), stat.top1*100, stat.top5*100, stat.top10*100, stat.sampleSize, stat.subjectCount, stat.duration, stat.hitRate*100.0, stat.precision, stat.precisionAt10, stat.recommendationCount)
	}
	f, _ := os.Create(filename + ".csv")
	f.WriteString(output)
	f.Close()
	return
}
