package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"schematree"
	"sort"
	"strconv"
	"sync"

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

	logr := log.New(os.Stderr, "", 0)

	// parse commandline arguments/flags
	flag.Parse()

	if *testFile == "" {
		log.Fatalln("A test set must be provided!")
	}

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

		var wg sync.WaitGroup
		results := make(chan evalResult, 1000) // collect eval results via channel

		// evaluate the rank the recommender assigns the left out property
		evaluate := func(properties schematree.IList, leftOut *schematree.IItem) {
			recs := tree.RecommendProperty(properties)
			for i, r := range recs {
				if r.Property == leftOut { // found item to recover
					for i > 0 && recs[i-1].Probability == r.Probability {
						i--
					}
					results <- evalResult{uint16(len(properties)), uint32(i)}
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

		go func() {
			wg.Add(1)
			for res := range results {
				stats[res.setSize] = append(stats[res.setSize], res.position)
			}
			wg.Done()
		}()

		subjectCount := schematree.SubjectSummaryReader(*testFile, tree.PropMap, tree.TypeMap, handler, 0)
		logr.Printf("\nEvaluation with total of %v subject sets!\n", subjectCount)
		close(results)
		wg.Wait()

		f, _ := os.Create(*testFile + ".eval")
		e := gob.NewEncoder(f)
		// e.Encode(summary)
		e.Encode(stats)
		f.Close()
	}

	makeStatistics(stats, *testFile)
}

type evalResult struct {
	setSize  uint16
	position uint32
}

type evalSummary struct {
	setSize  uint16
	median   float64
	mean     float64
	variance float64
}

func makeStatistics(stats map[uint16][]uint32, fileName string) (output string) {
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
		line, _ := plotter.NewLine(toPoints(v))
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
	for i, y := range v {
		pts[i].X = float64(i) / l
		pts[i].Y = float64(y) / maxY
	}
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