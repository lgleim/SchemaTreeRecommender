package main

import (
	"fmt"
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
	subjects     int16
}

// makeStatics receive a list of evaluation results and makes a summary of them.
func makeStatistics(results []evalResult, groupBy string) (statistics []evalSummary) {
	average := int(-1)
	groupedMap := make(map[int][]evalResult)
	indexList := make([]int, 0, len(results))
	indexList = append(indexList, average)

	for _, res := range results {
		var group int

		if groupBy == "numTypes" {
			group = int(res.numTypes)
		} else if groupBy == "setSize" {
			group = int(res.setSize)
		} else if groupBy == "numLeftOut" {
			group = int(res.numLeftOut)
		}
		indexList = append(indexList, group)
		groupedMap[average] = append(groupedMap[average], res)
		groupedMap[group] = append(groupedMap[group], res)
	}

	// compute statistics
	sort.Ints(indexList)

	for i, index := range indexList {
		groupedResults := groupedMap[index]
		length := len(groupedResults)

		totalDuration := int64(0)
		totalRank := uint32(0)
		totalRankIfHit := uint32(0)
		allHitCount := uint32(0)
		totalNumTP := uint32(0)
		totalNumFP := uint32(0)
		totalNumFN := uint32(0)
		totalInTop1 := uint32(0)
		totalInTop5 := uint32(0)
		totalInTop10 := uint32(0)
		totalInTopL := uint32(0)
		totalNumTPAtL := uint32(0)

		for _, result := range groupedResults {
			if result.rank == 1 {
				totalInTop1++
			}
			if result.rank <= 5 {
				totalInTop5++
			}
			if result.rank <= 10 {
				totalInTop10++
			}
			if result.rank <= uint32(result.numLeftOut) {
				totalInTopL++
			}
			if result.numFN == 0 {
				totalRankIfHit += result.rank
				allHitCount++
			}

			totalDuration += result.duration
			totalNumTP += result.numTP
			totalNumFN += result.numFN
			totalNumFP += result.numFP
			totalRank += result.rank
			totalNumTPAtL += result.numTNAtL
		}

		var mean, meanSquare, median, variance float64

		if length == 1 {
			mean = float64(groupedResults[0].rank)
			median = mean
			variance = 0
		} else {
			if length%2 != 0 {
				median = float64(groupedResults[length/2].rank)
			} else {
				median = (float64(groupedResults[length/2-1].rank) + float64(groupedResults[length/2].rank)) / 2.0
			}

			for _, result := range groupedResults {
				meanSquare += float64(result.rank) * float64(result.rank) / float64(length)
			}
			mean = float64(totalRank) / float64(length)
			variance = meanSquare - (mean * mean)
		}

		statistics[i].duration = float64(totalDuration) / float64(length)
		statistics[i].recall = float64(totalNumTP) / float64(totalNumTP+totalNumFN)
		statistics[i].precision = float64(totalNumTP) / float64(totalNumTP+totalNumFP)
		statistics[i].precisionAtL = float64(totalNumTPAtL) / float64(totalNumTP+totalNumFN)
		statistics[i].top1 = float64(totalInTop1) / float64(length)
		statistics[i].top5 = float64(totalInTop5) / float64(length)
		statistics[i].top10 = float64(totalInTop10) / float64(length)
		statistics[i].topL = float64(totalInTopL) / float64(length)
		statistics[i].rankAvg = mean
		statistics[i].rankIfHitAvg = float64(totalRankIfHit) / float64(allHitCount)
		statistics[i].groupBy = int16(index)
		statistics[i].variance = variance
		statistics[i].median = median
		statistics[i].subjects = int16(length)
	}
	return
}

func writeStatisticsToFile(filename string, groupBy string, statistics []evalSummary) {
	f, _ := os.Create(filename + ".csv")
	f.WriteString(fmt.Sprintf(
		"%15s,%15s,%15s,%15s,%15s,%15s,%15s,%15s,%15s,%15s,%15s,%15s,%15s,%15s\n",
		groupBy, "subjects", "duration",
		"mean", "meanOfHits", "median", "variance",
		"top1", "top5", "top10", "topL",
		"recall", "precision", "precisionAtL",
	))

	for _, stat := range statistics {
		f.WriteString(fmt.Sprintf(
			"%15d,%15d,%15.4f,%15.4f,%15.4f,%15.4f,%15.4f,%15.4f,%15.4f,%15.4f,%15.4f,%15.4f,%15.4f,%15.4f,\n",
			stat.groupBy, stat.subjects, stat.duration/1000000,
			stat.rankAvg, stat.rankIfHitAvg, stat.median, stat.variance,
			stat.top1*100, stat.top5*100, stat.top10*100, stat.topL*100,
			stat.recall*100, stat.precision*100, stat.precisionAtL*100,
		))
	}
	f.Close()
	return
}
