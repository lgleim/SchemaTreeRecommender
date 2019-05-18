package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	pSetFile := flag.String("pSet", "", "the pSet JSON file to use for the evaluation")

	// parse commandline arguments/flags
	flag.Parse()

	if *pSetFile == "" {
		log.Fatalln("A pSet must be provided!")
	}

	f, err := os.Open(*pSetFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	//
	// collect stats
	//
	var counts [51]int
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		properties := []string{}
		json.Unmarshal([]byte(scanner.Text()), &properties)

		// properties is a string array with 4-50 PXXX wikidata property ids
		counts[0]++
		counts[len(properties)]++
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Evaluating with %v property sets. Set size distribution: \n", counts[0])
	for i, c := range counts[1:] {
		fmt.Printf("%v\t%v\n", i+1, c)
	}
	f.Seek(0, 0) // rewinding file

	//
	// evaluation
	//
	fmt.Println("\nStarting evaluation:")
	scanner = bufio.NewScanner(f)
	results := make(map[string]Result)

	// Mean Reciprocal Rank (MRR) computation logic
	var treeMRRsum, wikiMRRsum [10]float64
	var treeMRRcnt, wikiMRRcnt [10]uint64
	treeRR := make(chan [20]float64)
	go func() {
		for x := range treeRR {
			for i := 0; i < 10; i++ {
				treeMRRsum[i] += x[i]
				treeMRRcnt[i] += uint64(x[10+i])
			}
		}
	}()
	wikiRR := make(chan [20]float64)
	go func() {
		for x := range wikiRR {
			for i := 0; i < 10; i++ {
				wikiMRRsum[i] += x[i]
				wikiMRRcnt[i] += uint64(x[10+i])
			}
		}
	}()

	// actual evaluation function interatively collecting recommendations
	evaluate := func(
		properties []string, // original full set of properties
		pRem map[string]bool, // set of removed properties
		getRecs func(properties []string) []string, // recommendation function to use
		RR chan [20]float64, // where to collect reciprocal rank
	) (
		aggRecs [][]string, // slice of all property recommendations received from recommender service for logging purpose
		aggPrecRec [][10]PrecRec, // slice of respective precision & recall for each of these recommendation calls for k=1..10 in each [10]PrecRec
		recovered [10]int, // the number of properties that could be recovered in total for k=1..10
	) {
		// defer func() {
		// 	if r := recover(); r != nil {
		// 		fmt.Println("Recovered in f", r)
		// 	}
		// }()

		aggRecs = make([][]string, 0, len(properties)-3)
		aggPrecRec = make([][10]PrecRec, 0, len(properties)-3)
		curPs := append([]string(nil), properties[:3]...) // start with 3 properties
		minK := 0                                         // ks still in

		updated := true // a new property was found in the last interation
		for iteration := 0; updated && len(curPs) < len(properties); iteration++ {

			updated = false

			recs := getRecs(curPs)          // get recommendations
			aggRecs = append(aggRecs, recs) // add to log
			if len(recs) > 10 {             // cap to at max 10
				recs = recs[:10]
			}

			// precision & recall
			var precRec [10]PrecRec
			for i := 0; i < 10; i++ {
				precRec[i] = precisionAndRecall(recs[:min(i+1, len(recs))], pRem, i+1)
			}
			aggPrecRec = append(aggPrecRec, precRec)

			// MRR & reconstruction
			// find first correct recommendation if any
			for k, r := range recs {
				if pRem[r] {
					// ensure this is indeed a new recommendation. should never be true
					for _, x := range curPs {
						if r == x {
							continue
						}
					}

					// add to recovered properties
					curPs = append(curPs, r)

					var rr [20]float64
					if k > minK {
						// register unsuccessfull runs to get results similar to the ones in the reference paper
						for i := minK; i < k; i++ {
							rr[10+i] = 1
						}

						minK = k // update minK
					}

					for i := minK; i < 10; i++ {
						recovered[i]++ // count for reconstruction measure
						rr[i] = float64(1) / float64(k+1)
						rr[10+i] = 1
					}

					RR <- rr

					updated = true
					break
				}
			}
		}
		return
	}

	// var wikiMRRsum [10]float64
	// var wikiMRRcnt [10]uint64

	cnt := 0
	for scanner.Scan() {
		cnt++
		if cnt > 1 {
			break
		}

		fmt.Printf("Processing set %v/%v\n", cnt, counts[0])

		properties := []string{}
		json.Unmarshal([]byte(scanner.Text()), &properties)

		sorted := append([]string(nil), properties...)
		sort.Strings(sorted)

		key := strings.Join(sorted, "|")

		if _, ok := results[key]; ok {
			continue
		}

		// properties is a string array with 4-50 PXXX wikidata property ids
		result := Result{
			properties:      properties,
			treeRecommended: make([][]string, 0, len(properties)-3),
			treePrecRec:     make([][10]PrecRec, 0, len(properties)-3),
			wikiRecommended: make([][]string, 0, len(properties)-3),
		}
		results[key] = result

		// set of properties to recover
		pRem := make(map[string]bool, len(properties))
		for _, p := range properties[3:] {
			pRem[p] = true
		}

		result.treeRecommended, result.treePrecRec, result.treeRecovered = evaluate(properties, pRem, getTreeRecs, treeRR)
		result.wikiRecommended, result.wikiPrecRec, result.wikiRecovered = evaluate(properties, pRem, getWikiRecs, wikiRR)
		results[key] = result
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	// print MRR
	fmt.Println("MRR:\nk\tTree\tWiki")
	for i, x := range treeMRRsum {
		fmt.Printf("%v\t%v\t%v\n", i+1, x/float64(treeMRRcnt[9]), wikiMRRsum[i]/float64(wikiMRRcnt[i]))
	}

	fmt.Println("Mean precision/recall\nk\tprecision\trecall")
	var treePreS, treeRecS, wikiPreS, wikiRecS [10]float64
	var treeCtr, wikiCtr uint64
	for _, x := range results {
		for _, prs := range x.treePrecRec {
			treeCtr++
			for i, pr := range prs {
				treePreS[i] += pr.precision
				treeRecS[i] += pr.recall
			}
		}
		for _, prs := range x.wikiPrecRec {
			wikiCtr++
			for i, pr := range prs {
				treePreS[i] += pr.precision
				treeRecS[i] += pr.recall
			}
		}
	}
	fmt.Println("TreeRecommender")
	for i, _ := range treePreS {
		fmt.Printf("%v\t%v\t%v\n", i+1, treePreS[i]/float64(treeCtr), treeRecS[i]/float64(treeCtr))
	}
	fmt.Println("WikiRecommender")
	for i, _ := range wikiPreS {
		fmt.Printf("%v\t%v\t%v\n", i+1, wikiPreS[i]/float64(wikiCtr), wikiRecS[i]/float64(wikiCtr))
	}

}

type PrecRec struct {
	precision float64
	recall    float64
}

// Result is a wikievaluation result
type Result struct {
	properties []string // the set of properties the evaluation is derived from

	treeRecommended [][]string    // sequence of recommendation results of recommendation iteration
	treeRecovered   [10]int       // the number of properties recovered for k=1..10
	treePrecRec     [][10]PrecRec // the recovery precision & recall for k=1..10

	wikiRecommended [][]string    // sequence of recommendation results of recommendation iteration
	wikiRecovered   [10]int       // the number of properties recovered for k=1..10
	wikiPrecRec     [][10]PrecRec // the recovery precision & recall for k=1..10
}

func min(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func precisionAndRecall(Prec []string, Prem map[string]bool, k int) PrecRec {
	i := 0
	for _, r := range Prec {
		if Prem[r] {
			i++
		}
	}
	precision := float64(i) / float64(len(Prec))
	recall := float64(i) / math.Min(float64(len(Prec)), float64(k))
	return PrecRec{precision, recall}
}

func getTreeRecs(properties []string) []string {
	data, err := json.Marshal(properties)
	if err != nil {
		panic(fmt.Sprintf("Failed to Marshal properties array %v with error message %v", properties, err))
	}

	res, err := http.Post("http://bruegel.informatik.rwth-aachen.de:8080/wikiRecommender", "application/json", bytes.NewBuffer(data))
	if err != nil {
		panic(err)
	}
	if res.StatusCode != 200 {
		b, _ := ioutil.ReadAll(res.Body)
		panic(string(b))
	}

	var recs []string
	err = json.NewDecoder(res.Body).Decode(&recs)
	if err != nil {
		panic(fmt.Sprintf("received malformatted response from schematree recommender for property set %v", properties))
	}
	// fmt.Println(properties)
	// fmt.Println(recs)
	return recs
}

func getWikiRecs(properties []string) []string {
	url := "https://www.wikidata.org/w/api.php?action=wbsgetsuggestions&limit=10&format=json&properties=" + strings.Join(properties, "|")
	res, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	if res.StatusCode != 200 {
		b, _ := ioutil.ReadAll(res.Body)
		panic(fmt.Sprint(url, string(b)))
	}
	var recs struct {
		Search []struct {
			ID string `json:"id"`
		} `json:"search"`
	}
	err = json.NewDecoder(res.Body).Decode(&recs)
	if err != nil {
		panic(fmt.Sprintf("received malformatted response from wikidata recommender for property set %v. Error: %v", properties, err))
	}

	ranked := make([]string, len(recs.Search), len(recs.Search))
	for i, r := range recs.Search {
		ranked[i] = r.ID
	}
	// fmt.Println(url, ranked)
	return ranked
}
