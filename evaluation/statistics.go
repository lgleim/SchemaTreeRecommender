package main

import (
	"fmt"
	"math"
	"os"
	"sort"
)

type evalSummary struct {
	groupBy      int16
	rankAvg      float64
	rankIfHitAvg float64
	duration     float64
	recall       float64
	precision    float64
	precisionAtL float64
	topL         float64
	top1         float64
	top5         float64
	top10        float64
	median       float64
	variance     float64
	subjects     int64
}

// makeStatics receive a list of evaluation results and makes a summary of them.
func makeStatistics(results []evalResult, groupBy string) (statistics []evalSummary) {
	resultsByGroup := make(map[int][]evalResult) // stores grouped results
	const allResults int = -1                    // catch all group
	statistics = make([]evalSummary, 0, len(results)+1)
	groupIds := []int{allResults} // keep track of existing groups
	groupExists := make(map[int]bool)

	for _, res := range results {
		var groupId int

		if groupBy == "numTypes" {
			groupId = int(res.numTypes)
		} else if groupBy == "numNonTypes" {
			groupId = int(res.setSize + res.numLeftOut - res.numTypes)
		} else if groupBy == "setSize" {
			groupId = int(res.setSize)
		} else if groupBy == "numLeftOut" {
			groupId = int(res.numLeftOut)
		} else {
			panic("No suitable groupBy has been selected.")
		}

		if !groupExists[groupId] {
			groupExists[groupId] = true
			groupIds = append(groupIds, groupId)
		}

		resultsByGroup[allResults] = append(resultsByGroup[allResults], res)
		resultsByGroup[groupId] = append(resultsByGroup[groupId], res)
	}

	// compute statistics
	sort.Ints(groupIds)

	for _, index := range groupIds {
		groupedResults := resultsByGroup[index]
		resCount := len(groupedResults)

		var Duration int64
		var HitCount, NumTP, NumFP, NumFN, InTop1, InTop5, InTop10, InTopL uint32
		var Rank, RankIfHit uint64
		var Precision, PrecisionAtL, Recall float64 // RecallAtL==PrecisionAtL

		sort.Slice( // sort results by rank in order to be able to compute the median
			groupedResults,
			func(i, j int) bool { return groupedResults[i].rank < groupedResults[j].rank },
		)

		for _, result := range groupedResults {
			if result.rank < 1 {
				fmt.Printf("rank below 1")
			}
			if result.rank == 1 {
				InTop1++
			}
			if result.rank <= 5 {
				InTop5++
			}
			if result.rank <= 10 {
				InTop10++
			}
			if result.rank <= uint32(result.numLeftOut) {
				InTopL++
			}

			Duration += result.duration
			NumTP += result.numTP
			NumFN += result.numFN
			NumFP += result.numFP
			Recall += float64(result.numTP) / float64(result.numLeftOut)
			Precision += float64(result.numTP) / float64(result.numTP+result.numFP)
			PrecisionAtL += float64(result.numTPAtL) / float64(result.numLeftOut)
			Rank += uint64(result.rank)
			if result.rank < 500 {
				RankIfHit += uint64(result.rank)
				HitCount++
			}
		}

		var mean, median, variance float64

		if resCount == 1 {
			mean = float64(groupedResults[0].rank)
			median = mean
			variance = 0
		} else {
			if resCount%2 != 0 {
				median = float64(groupedResults[resCount/2].rank)
			} else {
				median = (float64(groupedResults[resCount/2-1].rank) + float64(groupedResults[resCount/2].rank)) / 2.0
			}
			mean = float64(Rank) / float64(resCount)

			for _, result := range groupedResults {
				err := float64(result.rank) - mean
				variance += err * err / float64(resCount)
			}
		}

		newStat := evalSummary{
			groupBy:      int16(index),
			rankAvg:      mean,
			rankIfHitAvg: 500, // set to max by default
			duration:     float64(Duration) / float64(resCount),
			recall:       Recall / float64(resCount),
			precision:    Precision / float64(resCount),
			precisionAtL: PrecisionAtL / float64(resCount), // == recallAtL
			topL:         float64(InTopL) / float64(resCount),
			top1:         float64(InTop1) / float64(resCount),
			top5:         float64(InTop5) / float64(resCount),
			top10:        float64(InTop10) / float64(resCount),
			median:       median,
			variance:     variance,
			subjects:     int64(resCount),
		}
		if HitCount != 0 {
			newStat.rankIfHitAvg = float64(RankIfHit) / float64(HitCount)
		}

		statistics = append(statistics, newStat)
	}
	return
}

func writeStatisticsToFile(filename string, groupBy string, statistics []evalSummary) {
	f, _ := os.Create(filename + ".csv")
	f.WriteString(fmt.Sprintf(
		"%12s,%12s,%12s,%12s,%12s,%12s,%12s,%12s,%12s,%12s,%12s,%12s,%12s,%12s\n",
		groupBy, "subjects", "duration",
		"mean", "meanOfHits", "median", "stddev",
		"top1", "top5", "top10", "topL",
		"recall", "precision", "precisionAtL",
	))

	for _, stat := range statistics {
		f.WriteString(fmt.Sprintf(
			"%12d,%12d,%12.4f,%12.4f,%12.4f,%12.4f,%12.4f,%12.4f,%12.4f,%12.4f,%12.4f,%12.4f,%12.4f,%12.4f\n",
			stat.groupBy, stat.subjects, stat.duration/1000000,
			stat.rankAvg, stat.rankIfHitAvg, stat.median, math.Sqrt(stat.variance),
			stat.top1*100, stat.top5*100, stat.top10*100, stat.topL*100,
			stat.recall*100, stat.precision*100, stat.precisionAtL*100,
		))
	}
	f.Close()
	return
}
