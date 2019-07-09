package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"recommender/assessment"
	"recommender/schematree"
	"recommender/strategy"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sort"
	"strconv"
	"sync"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile := flag.String("memprofile", "", "write memory profile to `file`")
	traceFile := flag.String("trace", "", "write execution trace to `file`")
	trainedModel := flag.String("model", "", "read stored schematree from `file`")
	testFile := flag.String("testSet", "", "the file to parse")
	batchTest := flag.Bool("batchTest", false, "Switch between batch test and normal test")
	createConfigs := flag.Bool("createConfigs", false, "Create a bunch of config")
	createConfigsCreater := flag.String("creater", "", "Json which defines the creater config file in ./configs")
	numberConfigs := flag.Int("numberConfigs", 1, "CNumber of config files in ./configs")

	logr := log.New(os.Stderr, "", 0)

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

	stats := make(map[uint16][]uint32)

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
		err := batchConfigBenchmark(*trainedModel, *numberConfigs)
		if err != nil {
			log.Fatalln("Batch Config Failed", err)
			return
		}
	} else {

		if *testFile == "" {
			log.Fatalln("A test set must be provided!")
		}

		f, err := os.Open(*testFile + ".eval")
		if err == nil {
			logr.Println("Loading evaluation results from previous run!")
			decoder := gob.NewDecoder(f)

			err = decoder.Decode(&stats)
			if err != nil {
				log.Fatalln("Failed to decode stats!", err)
			}
			// // to be deleted ...
			// var summary []evalSummary
			// err = decoder.Decode(&summary)
			// fmt.Println(err, summary)
			// // ... /

		} else {
			// evaluation
			if *trainedModel == "" {
				log.Fatalln("A model must be provided!")
			}
			tree, err := schematree.LoadSchemaTree(*trainedModel)
			if err != nil {
				log.Fatalln(err)
			}
			stats, _ = evaluation(tree, testFile, strategy.MakePresetWorkflow("direct", tree))

			f, _ := os.Create(*testFile + ".eval")
			e := gob.NewEncoder(f)
			// e.Encode(summary)
			e.Encode(stats)
			f.Close()
		}
		writeStatisticsToFile(stats, *testFile)
	}
}

type evalResult struct {
	setSize   uint16
	position  uint32
	resources *evalResources
}
type evalResources struct {
	duration         int64
	memoryAllocation uint64
}

func evaluation(tree *schematree.SchemaTree, testFile *string, wf *strategy.Workflow) (stats map[uint16][]uint32, resources map[uint16][]*evalResources) {
	resources = make(map[uint16][]*evalResources)
	stats = make(map[uint16][]uint32)

	// memoize empty set recommendations
	setSup := float64(tree.Root.Support) // empty set occured in all transactions
	emptyRecs := make([]schematree.RankedPropertyCandidate, len(tree.PropMap), len(tree.PropMap))
	for _, prop := range tree.PropMap {
		emptyRecs[int(prop.SortOrder)] = schematree.RankedPropertyCandidate{
			Property:    prop,
			Probability: float64(prop.TotalCount) / setSup,
		}
	}

	var wg sync.WaitGroup
	results := make(chan evalResult, 1000) // collect eval results via channel

	// evaluate the rank the recommender assigns the left out property
	evaluate := func(properties schematree.IList, leftOut *schematree.IItem) {
		var duration int64
		var recs []schematree.RankedPropertyCandidate
		var m runtime.MemStats
		var resource evalResources
		if len(properties) == 0 {
			recs = emptyRecs
		} else {
			start := time.Now()
			asm := assessment.NewInstance(properties, tree, true)
			recs = wf.Recommend(asm)
			duration = time.Since(start).Nanoseconds()
			runtime.ReadMemStats(&m)
			resource = evalResources{duration, m.Alloc / 1024 / 1024}
		}
		for i, r := range recs {
			if r.Property == leftOut { // found item to recover
				for i > 0 && recs[i-1].Probability == r.Probability {
					i--
				}
				results <- evalResult{uint16(len(properties)), uint32(i), &resource}
				break
			}
		}
	}

	handler := func(s *schematree.SubjectSummary) {
		properties := make(schematree.IList, 0, len(s.Properties))
		for p := range s.Properties {
			properties = append(properties, p)
		}
		properties.Sort()

		// take out one property from the list at a time and determine in which position it will be recommended again
		tmp := make(schematree.IList, len(properties)-1, len(properties)-1)
		copy(tmp, properties[1:])
		for i := range tmp {
			evaluate(tmp, properties[i])
			tmp[i] = properties[i]
		}
		evaluate(tmp, properties[len(properties)-1])
	}

	handlerTyped := func(s *schematree.SubjectSummary) {
		properties := make(schematree.IList, 0, len(s.Properties))
		for p := range s.Properties {
			properties = append(properties, p)
		}
		properties.Sort()

		// take out one property from the list at a time and determine in which position it will be recommended again
		tmp := make(schematree.IList, len(properties)-1, len(properties)-1)
		copy(tmp, properties[1:])
		for i := range tmp {
			if properties[i].IsProp() { // Only evaluate if the leftout is a property and not a type
				evaluate(tmp, properties[i])
			}
			tmp[i] = properties[i]
		}
		if properties[len(properties)-1].IsProp() {
			evaluate(tmp, properties[len(properties)-1])
		}
	}

	go func() {
		wg.Add(1)
		for res := range results {
			stats[res.setSize] = append(stats[res.setSize], res.position)
			resources[res.setSize] = append(resources[res.setSize], res.resources)
		}
		wg.Done()
	}()

	// TODO flag if anfrage
	// ohne types
	if false {
		schematree.SubjectSummaryReader(*testFile, tree.PropMap, handler, 0, false)
		close(results)
		wg.Wait()
	} else {
		// else mit types
		schematree.SubjectSummaryReader(*testFile, tree.PropMap, handlerTyped, 0, true)
		close(results)
		wg.Wait()
	}

	var lenght uint32
	for _, rank_list := range stats {
		lenght += uint32(len(rank_list))
	}
	total := make([]uint32, 0, lenght)
	total_res := make([]*evalResources, 0, lenght)

	for _, rank_list := range stats {
		total = append(total, rank_list...)
	}
	for _, rank_list := range resources {
		total_res = append(total_res, rank_list...)
	}
	stats[0] = total
	resources[0] = total_res

	return stats, resources
}

func makeStatistics(stats map[uint16][]uint32, resources map[uint16][]*evalResources) (statistics []evalSummary) {
	// compute statistics
	duration := make(map[uint16]float64)
	memoryAllocation := make(map[uint16]float64)
	var averageSize float64

	for k, v := range resources {
		for _, res := range v {
			duration[k] = duration[k] + float64(res.duration)
			memoryAllocation[k] = memoryAllocation[k] + float64(res.memoryAllocation)
		}
	}
	for k, v := range resources {
		duration[k] = duration[k] / float64(len(v))
		memoryAllocation[k] = memoryAllocation[k] / float64(len(v))
	}

	statistics = make([]evalSummary, len(stats))
	setLens := make([]int, 0, len(stats))
	for setLen := range stats {
		setLens = append(setLens, int(setLen))
	}

	for _, value := range setLens {
		averageSize += float64(value)
	}
	averageSize = averageSize / float64(len(setLens))

	sort.Ints(setLens)
	for i, setLen := range setLens {
		v := stats[uint16(setLen)]
		if len(v) == 0 {
			continue
		}
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })

		var sum uint64
		var mean, meanSquare, median, variance, top1, top5, top10, subjects, worst5average float64
		l := float64(len(v))

		top1 = float64(sort.Search(len(v), func(i int) bool { return v[i] >= 1 })) / float64(len(v))
		top5 = float64(sort.Search(len(v), func(i int) bool { return v[i] >= 5 })) / float64(len(v))
		top10 = float64(sort.Search(len(v), func(i int) bool { return v[i] >= 10 })) / float64(len(v))

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
		statistics[i] = evalSummary{setLen, median + 1, mean + 1, math.Sqrt(variance), top1 * 100, top5 * 100, top10 * 100, len(v), subjects, worst5average + 1, duration[uint16(setLen)], memoryAllocation[uint16(setLen)]}
	}
	return
}

type evalSummary struct {
	setSize          int
	median           float64
	mean             float64
	variance         float64
	top1             float64
	top5             float64
	top10            float64
	sampleSize       int
	subjectCount     float64
	worst5average    float64
	duration         float64
	memoryAllocation float64
}

func writeStatisticsToFile(stats map[uint16][]uint32, fileName string) (output string) {
	// compute statistics
	output = fmt.Sprintf("%8v, %8v, %8v, %12v, %8v, %8v, %8v, %10v, %10v\n", "set", "median", "mean", "stddev", "top1", "top5", "top10", "sampleSize", "#subjects")
	setLens := make([]int, 0, len(stats))
	for setLen := range stats {
		setLens = append(setLens, int(setLen))
	}
	sort.Ints(setLens)
	for _, setLen := range setLens {
		v := stats[uint16(setLen)]
		if len(v) == 0 {
			continue
		}
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })

		var sum uint64
		var mean, meanSquare, median, variance, top1, top5, top10 float64
		l := float64(len(v))

		top1 = float64(sort.Search(len(v), func(i int) bool { return v[i] >= 1 })) / float64(len(v))
		top5 = float64(sort.Search(len(v), func(i int) bool { return v[i] >= 5 })) / float64(len(v))
		top10 = float64(sort.Search(len(v), func(i int) bool { return v[i] >= 10 })) / float64(len(v))

		if len(v) == 1 {
			mean = float64(v[0])
			median = mean
			variance = 0
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
		}

		output += fmt.Sprintf("%8v, %8v, %8.4f, %12.4f, %8.4f, %8.4f, %8.4f, %10v, %10v\n", setLen, median+1, mean+1, math.Sqrt(variance), top1*100, top5*100, top10*100, len(v), len(v)/(setLen+1))
	}
	f, _ := os.Create(fileName + ".csv")
	f.WriteString(output)
	f.Close()

	// Plot experiment
	p, _ := plot.New()
	p.Title.Text = "Distribution of recommendation position of correct element for given set sizes"
	p.X.Label.Text = "Percentile of recommendations"
	p.Y.Label.Text = "normalized recommendation rank"
	// Draw a grid behind the data
	p.Add(plotter.NewGrid())

	p.X.Tick.Marker = ticks{}
	p.Y.Tick.Marker = p.X.Tick.Marker

	// Draw legend on the top left
	p.Legend.Left = true
	p.Legend.Top = true
	p.Legend.XOffs = 0.7 * vg.Inch
	p.Legend.YOffs = -0.2 * vg.Inch

	l := len(stats)
	for i, setLen := range setLens {
		v := stats[uint16(setLen)]
		// plotutil.AddLines(p, strconv.Itoa(setLen))
		line, err := plotter.NewLine(toPoints(v))
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		line.LineStyle.Color = color.RGBA{0, uint8(i * 255 / l), 0, 255}
		p.Add(line)
		p.Legend.Add(strconv.Itoa(setLen), line)
	}
	// Save the plot to a PNG file.
	if err := p.Save(24*vg.Inch, 12*vg.Inch, fileName+".svg"); err != nil {
		panic(err)
	}
	return
}

// toPoints returns corresponding points on 0-100 range in XY dimensions
func toPoints(v []uint32) (pts plotter.XYs) {
	if len(v) < 2 {
		log.Fatalln("v has to few samples", v)
	}
	pts = make(plotter.XYs, len(v), len(v))
	l := float64(len(v)-1) / 100
	maxY := float64(v[len(v)-1]) / 100
	if maxY == 0 {
		maxY = 1
	}
	for i, y := range v {
		pts[i].X = float64(i) / l
		pts[i].Y = float64(y) / maxY
	}

	// downsample to a maximum of 100 points
	pts = LTTB(pts, 100)
	return
}

type ticks struct{}

// Ticks returns Ticks in the specified range.
func (ticks) Ticks(min, max float64) []plot.Tick {
	if max <= min {
		panic("illegal range")
	}

	const suggestedTicks = 11

	labels, step, q, mag := talbotLinHanrahan(min, max, suggestedTicks, withinData, nil, nil, nil)
	majorDelta := step * math.Pow10(mag)
	if q == 0 {
		// Simple fall back was chosen, so
		// majorDelta is the label distance.
		majorDelta = labels[1] - labels[0]
	}

	// Choose a reasonable, but ad
	// hoc formatting for labels.
	fc := byte('f')
	var off int
	if mag < -1 || 6 < mag {
		off = 1
		fc = 'g'
	}
	if math.Trunc(q) != q {
		off += 2
	}
	prec := minInt(6, maxInt(off, -mag))
	var ticks []plot.Tick
	for _, v := range labels {
		ticks = append(ticks, plot.Tick{Value: v, Label: strconv.FormatFloat(v, fc, prec, 64)})
	}

	var minorDelta float64
	// See talbotLinHanrahan for the values used here.
	switch step {
	case 1, 2.5:
		minorDelta = majorDelta / 5
	case 2, 3, 4, 5:
		minorDelta = majorDelta / step
	default:
		if majorDelta/2 < dlamchP {
			return ticks
		}
		minorDelta = majorDelta / 2
	}

	// Find the first minor tick not greater
	// than the lowest data value.
	var i float64
	for labels[0]+(i-1)*minorDelta > min {
		i--
	}
	// Add ticks at minorDelta intervals when
	// they are not within minorDelta/2 of a
	// labelled tick.
	for {
		val := labels[0] + i*minorDelta
		if val > max {
			break
		}
		found := false
		for _, t := range ticks {
			if math.Abs(t.Value-val) < minorDelta/2 {
				found = true
			}
		}
		if !found {
			ticks = append(ticks, plot.Tick{Value: val})
		}
		i++
	}

	return ticks
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// LTTB down-samples the data to contain only threshold number of points that
// have the same visual shape as the original data
// adapted from https://github.com/dgryski/go-lttb/blob/master/lttb.go
func LTTB(data plotter.XYs, threshold int) plotter.XYs {

	if threshold >= len(data) || threshold == 0 {
		return data // Nothing to do
	}

	sampled := make(plotter.XYs, 0, threshold)

	// Bucket size. Leave room for start and end data points
	every := float64(len(data)-2) / float64(threshold-2)

	sampled = append(sampled, data[0]) // Always add the first point

	bucketStart := 1
	bucketCenter := int(math.Floor(every)) + 1

	var a int

	for i := 0; i < threshold-2; i++ {

		bucketEnd := int(math.Floor(float64(i+2)*every)) + 1

		// Calculate point average for next bucket (containing c)
		avgRangeStart := bucketCenter
		avgRangeEnd := bucketEnd

		if avgRangeEnd >= len(data) {
			avgRangeEnd = len(data)
		}

		avgRangeLength := float64(avgRangeEnd - avgRangeStart)

		var avgX, avgY float64
		for ; avgRangeStart < avgRangeEnd; avgRangeStart++ {
			avgX += data[avgRangeStart].X
			avgY += data[avgRangeStart].Y
		}
		avgX /= avgRangeLength
		avgY /= avgRangeLength

		// Get the range for this bucket
		rangeOffs := bucketStart
		rangeTo := bucketCenter

		// Point a
		pointAX := data[a].X
		pointAY := data[a].Y

		maxArea := -1.0

		var nextA int
		for ; rangeOffs < rangeTo; rangeOffs++ {
			// Calculate triangle area over three buckets
			area := (pointAX-avgX)*(data[rangeOffs].Y-pointAY) - (pointAX-data[rangeOffs].X)*(avgY-pointAY)
			// We only care about the relative area here.
			// Calling math.Abs() is slower than squaring
			area *= area
			if area > maxArea {
				maxArea = area
				nextA = rangeOffs // Next a is this b
			}
		}

		sampled = append(sampled, data[nextA]) // Pick this point from the bucket
		a = nextA                              // This a is the next a (chosen b)

		bucketStart = bucketCenter
		bucketCenter = bucketEnd
	}

	sampled = append(sampled, data[len(data)-1]) // Always add last

	return sampled
}
