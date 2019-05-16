package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"gonum.org/v1/gonum/stat"
)

// Result is a wikievaluation result
type RecommendationResult struct {
	Properties      []string   // the set of properties the evaluation is derived from
	TreeRecommended [][]string // sequence of recommendation results of recommendation iteration
	WikiRecommended [][]string // sequence of recommendation results of recommendation iteration
}

type evalAgg struct {
	precision      [10][]float64
	recall         [10][]float64
	reconstruction [10][]float64
	rank           []int
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	results := make(map[string]RecommendationResult)

	files := checkExt(".recommendations")

	// load all results
	for i, fName := range files {
		fmt.Printf("\rLoading file %v/%v... ", i+1, len(files))
		f, err := os.Open(fName)
		if err != nil {
			log.Fatal(err)
		}

		var tmp map[string]RecommendationResult

		json.NewDecoder(f).Decode(&tmp)
		f.Close()

		for k, v := range tmp {
			results[k] = v
		}
	}

	// evaluation
	fmt.Printf("Computing statistics for %v distinct property sets.\n", len(results))

	// NOTE: In the following all [10] arrays are for storing respective metrics for k=1..10
	treeVal := evalAgg{}
	wikiVal := evalAgg{}

	for _, trial := range results {
		computeStats(trial.Properties, trial.TreeRecommended, &treeVal)
		computeStats(trial.Properties, trial.WikiRecommended, &wikiVal)
	}

	fmt.Println("\nTreeRecommender")
	printStats(treeVal)

	fmt.Println("\nWikiRecommender")
	printStats(wikiVal)

	// // print MRR
	// fmt.Println("MRR:\nk\tTree\tWiki")
	// for i, x := range treeMRRsum {
	// 	fmt.Printf("%v\t%v\t%v\n", i+1, x/float64(treeMRRcnt[9]), wikiMRRsum[i]/float64(wikiMRRcnt[i]))
	// }

	// fmt.Println("Mean precision/recall\nk\tprecision\trecall")
	// var treePreS, treeRecS, wikiPreS, wikiRecS [10]float64
	// var treeCtr, wikiCtr uint64
	// for _, x := range results {
	// 	for _, prs := range x.treePrecRec {
	// 		treeCtr++
	// 		for i, pr := range prs {
	// 			treePreS[i] += pr.precision
	// 			treeRecS[i] += pr.recall
	// 		}
	// 	}
	// 	for _, prs := range x.wikiPrecRec {
	// 		wikiCtr++
	// 		for i, pr := range prs {
	// 			treePreS[i] += pr.precision
	// 			treeRecS[i] += pr.recall
	// 		}
	// 	}
	// }
	// fmt.Println("TreeRecommender")
	// for i, _ := range treePreS {
	// 	fmt.Printf("%v\t%v\t%v\n", i+1, treePreS[i]/float64(treeCtr), treeRecS[i]/float64(treeCtr))
	// }
	// fmt.Println("WikiRecommender")
	// for i, _ := range wikiPreS {
	// 	fmt.Printf("%v\t%v\t%v\n", i+1, wikiPreS[i]/float64(wikiCtr), wikiRecS[i]/float64(wikiCtr))
	// }

}

func printStats(stats evalAgg) {
	// convert to double
	tmp := make([]float64, len(stats.rank), len(stats.rank))
	for i, rs := range stats.rank {
		tmp[i] = float64(rs)
	}
	fmt.Printf("Mean Rank: %v\n", stat.Mean(tmp, nil)+1)

	fmt.Printf("Median Rank: %v\n", median(tmp)+1)

	// compute reciprocal
	for i := range tmp {
		tmp[i] = 1 / (tmp[i] + 1)
	}
	fmt.Printf("Mean Reciprocal Rank (MRR): %v\n", stat.Mean(tmp, nil))

	// avg. reconstruction
	fmt.Println("k\tMean Reconstruction:")
	for k, rs := range stats.reconstruction {
		fmt.Printf("%v\t%v\n", k+1, stat.Mean(rs, nil))
	}

	// median reconstruction
	fmt.Println("k\tMedian Reconstruction:")
	for k, rs := range stats.reconstruction {
		fmt.Printf("%v\t%v\n", k+1, median(rs))
	}
}

func median(xs []float64) float64 {
	sort.Float64s(xs)
	if len(xs)%2 == 1 {
		return xs[len(xs)/2]
	}
	return (xs[len(xs)/2] + xs[len(xs)/2+1]) / 2
}

func computeStats(properties []string, recs [][]string, stats *evalAgg) {
	// set of properties that had to be recovered
	pRem := make(map[string]bool, len(properties))
	for _, p := range properties[3:] {
		pRem[p] = true
	}

	curPs := append([]string(nil), properties[:3]...) // start with 3 properties
	minK := 0                                         // top-k recommenders still in

	var recovered [10]int

	for _, Precs := range recs {
		// cap to at max 10 recommendations
		if len(Precs) > 10 {
			Precs = Precs[:10]
		}

		// find first correct recommendation if any
		for k, r := range Precs {
			if pRem[r] {
				// ensure this was indeed a new recommendation. should never be true
				for _, x := range curPs {
					if r == x {
						fmt.Println("WARNING, recommended re-recommended property:" + r)
						continue // this does not do what it is supposed to do... SHIT
					}
				}

				// add to recovered properties
				curPs = append(curPs, r)
				stats.rank = append(stats.rank, k)

				if k > minK {
					minK = k // update minK
				}

				for i := minK; i < 10; i++ {
					recovered[i]++ // count for reconstruction measure
				}
				break
			}
		}
		// // precision & recall
		// for i := 0; i < 10; i++ {
		// 	precRec[i] = precisionAndRecall(recs[:min(i+1, len(recs))], pRem, i+1)
		// }
	}
	for i, x := range recovered {
		stats.reconstruction[i] = append(stats.reconstruction[i], float64(x)/float64(len(pRem)))
	}
}

func checkExt(ext string) []string {
	pathS, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	var files []string
	filepath.Walk(pathS, func(path string, f os.FileInfo, _ error) error {
		if !f.IsDir() {
			if filepath.Ext(path) == ext {
				files = append(files, f.Name())
			}
		}
		return nil
	})
	return files
}

// type PrecRec struct {
// 	precision float64
// 	recall    float64
// }

// func min(x, y int) int {
// 	if x > y {
// 		return x
// 	}
// 	return y
// }

// func precisionAndRecall(Prec []string, Prem map[string]bool, stillMissing float64) PrecRec {
// 	correct := 0
// 	for _, r := range Prec {
// 		if Prem[r] {
// 			correct++
// 		}
// 	}
// 	k:= float64(len(Prec))
// 	precision := float64(correct) / math.Min(k, stillMissing)
// 	recall := float64(correct) / math.Min(stillMissing, k)
// 	return PrecRec{precision, recall}
// }
