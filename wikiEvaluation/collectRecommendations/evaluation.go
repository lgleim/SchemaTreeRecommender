package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

var netClient = &http.Client{
	Timeout: time.Second * 20,
}

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
		Properties := []string{}
		json.Unmarshal([]byte(scanner.Text()), &Properties)

		// Properties is a string array with 4-50 PXXX wikidata property ids
		counts[0]++
		counts[len(Properties)]++
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
	var resLock sync.Mutex

	// actual evaluation function interatively collecting recommendations
	evaluate := func(
		Properties []string, // original full set of Properties
		getRecs func(Properties []string) []string, // recommendation function to use
	) (
		aggRecs [][]string, // slice of all property recommendations received from recommender service for logging purpose
	) {
		// defer func() {
		// 	if r := recover(); r != nil {
		// 		fmt.Println("Recovered in f", r)
		// 	}
		// }()

		aggRecs = make([][]string, 0, len(Properties)-3)
		curPs := append([]string(nil), Properties[:3]...) // start with 3 Properties

		// set of Properties to recover
		pRem := make(map[string]bool, len(Properties))
		for _, p := range Properties[3:] {
			pRem[p] = true
		}

		updated := true // a new property was found in the last interation
		for iteration := 0; updated && len(curPs) < len(Properties); iteration++ {

			updated = false

			recs := getRecs(curPs)          // get recommendations
			aggRecs = append(aggRecs, recs) // add to log
			if len(recs) > 10 {             // cap to at max 10
				recs = recs[:10]
			}

			// find first correct recommendation if any
			for _, r := range recs {
				if pRem[r] {
					// ensure this is indeed a new recommendation. should never be true
					for _, x := range curPs {
						if r == x {
							continue
						}
					}

					// add to recovered Properties
					curPs = append(curPs, r)
					updated = true
					break
				}
			}
		}
		return
	}

	// setup parallel workers
	var wg sync.WaitGroup
	pSets := make(chan []string)
	worker := func() {
		for Properties := range pSets {
			// avoid duplicate requests
			sorted := append([]string(nil), Properties...)
			sort.Strings(sorted)
			key := strings.Join(sorted, "|")

			resLock.Lock()
			if _, ok := results[key]; ok {
				resLock.Unlock()
				continue
			}

			// Properties is a string array with 4-50 PXXX wikidata property ids
			result := Result{
				Properties:      Properties,
				TreeRecommended: make([][]string, 0, len(Properties)-3),
				// WikiRecommended: make([][]string, 0, len(Properties)-3),
			}
			results[key] = result
			resLock.Unlock()

			result.TreeRecommended = evaluate(Properties, getTreeRecs)
			// result.WikiRecommended = evaluate(Properties, getWikiRecs)

			results[key] = result
		}
		wg.Done()
	}

	// set up worker
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go worker()
	}

	//
	// Process actual pSets file
	//
	cnt := 0
	for scanner.Scan() {
		cnt++
		// if cnt > 10 {
		// 	break
		// }

		fmt.Printf("Concurrently processing set %v/%v\n", cnt, counts[0])

		Properties := []string{}
		json.Unmarshal([]byte(scanner.Text()), &Properties)

		pSets <- Properties

	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	close(pSets)
	wg.Wait()

	// fmt.Println(results)

	out, err := os.Create(*pSetFile + ".recommendations")
	if err != nil {
		log.Fatal("could not create CPU profile: ", err)
	}
	defer out.Close()

	json.NewEncoder(out).Encode(results)

}

// Result is a wikievaluation result
type Result struct {
	Properties      []string   // the set of Properties the evaluation is derived from
	TreeRecommended [][]string // sequence of recommendation results of recommendation iteration
	WikiRecommended [][]string // sequence of recommendation results of recommendation iteration
}

func getTreeRecs(Properties []string) []string {
	data, err := json.Marshal(Properties)
	if err != nil {
		panic(fmt.Sprintf("Failed to Marshal Properties array %v with error message %v", Properties, err))
	}

	res, err := netClient.Post("http://bruegel.informatik.rwth-aachen.de:8080/wikiRecommender", "application/json", bytes.NewBuffer(data))
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
		panic(fmt.Sprintf("received malformatted response from schematree recommender for property set %v", Properties))
	}
	// fmt.Println(Properties)
	// fmt.Println(recs)
	return recs
}

func getWikiRecs(Properties []string) []string {
	url := "https://www.wikidata.org/w/api.php?action=wbsgetsuggestions&limit=10&format=json&properties=" + strings.Join(Properties, "|")
	res, err := netClient.Get(url)
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
		panic(fmt.Sprintf("received malformatted response from wikidata recommender for property set %v. Error: %v", Properties, err))
	}

	ranked := make([]string, len(recs.Search), len(recs.Search))
	for i, r := range recs.Search {
		ranked[i] = r.ID
	}
	// fmt.Println(url, ranked)
	return ranked
}
