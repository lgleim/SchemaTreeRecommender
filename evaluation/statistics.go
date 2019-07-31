package main

import (
	"fmt"
	"math"
	"os"
	"sort"
)

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
