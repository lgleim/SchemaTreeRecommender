package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"recommender/schematree"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestEval(t *testing.T) {
	go func() {
		for {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			if m.Alloc/1024/1024 > 1000 {
				t.Errorf("memory exeeded: Alloc = %v", m.Alloc/1024/1024)
			}
			time.Sleep(1 * time.Second)
		}
	}()
	runtime.GOMAXPROCS(runtime.NumCPU())

	trainedModel := "../testdata/10M.nt.gz.schemaTree.bin"
	testFile := "../testdata/10M.nt.gz"

	logr := log.New(os.Stderr, "", 0)

	stats := make(map[uint16][]uint32)

	tree, err := schematree.LoadSchemaTree(trainedModel)
	if err != nil {
		log.Fatalln(err)
	}

	var wg sync.WaitGroup
	results := make(chan evalResult, 1000) // collect eval results via channel

	// evaluate the rank the recommender assigns the left out property
	evaluate := func(properties schematree.IList, leftOut *schematree.IItem) {
		var recs []schematree.RankedPropertyCandidate
		if len(properties) != 0 {
			start := time.Now()

			recs = tree.RecommendProperty(properties)

			if time.Since(start).Nanoseconds() > 500000000 {
				t.Errorf("recomendation time too long: %v", time.Since(start).Nanoseconds()/1000000000)
			}

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

	subjectCount := schematree.SubjectSummaryReader(testFile, tree.PropMap, tree.TypeMap, handler, 0)
	logr.Printf("\nEvaluation with total of %v subject sets!\n", subjectCount)
	close(results)
	wg.Wait()

	var lenght uint32
	for _, rank_list := range stats {
		lenght += uint32(len(rank_list))
	}
	total := make([]uint32, 0, lenght)

	for _, rank_list := range stats {
		total = append(total, rank_list...)
	}
	stats[0] = total

	// compute statistics
	output := fmt.Sprintf("%8v, %8v, %8v, %12v, %8v, %8v, %8v, %10v, %10v\n", "set", "median", "mean", "stddev", "top1", "top5", "top10", "sampleSize", "#subjects")
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

		if top10 <= 0.85 {
			t.Errorf("top10 recomendation result insufficient: %v", top10)
		}

		output += fmt.Sprintf("%8v, %8v, %8.4f, %12.4f, %8.4f, %8.4f, %8.4f, %10v, %10v\n", setLen, median+1, mean+1, math.Sqrt(variance), top1*100, top5*100, top10*100, len(v), len(v)/(setLen+1))
	}
	logr.Printf("%s", output)
}
