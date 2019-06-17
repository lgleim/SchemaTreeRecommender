package schematree

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Serve provides a REST API for the given schematree on the given port
func Serve(schema *SchemaTree, port int, runBackoffStrategy string) {
	pMap := schema.PropMap

	recommender := func(w http.ResponseWriter, r *http.Request) {
		var properties []string
		err := json.NewDecoder(r.Body).Decode(&properties)
		if err != nil {
			w.Write([]byte("Malformed Request. Expected an array of property IRIs"))
			return
		}
		fmt.Println(properties)

		list := []*IItem{}
		for _, pString := range properties {
			p, ok := pMap[pString]
			if ok {
				list = append(list, p)
			}
		}
		// fmt.Println(schema.Support(list), schema.Root.Support)

		t1 := time.Now()

		// TODO replace this by a call of the framework, as soon as the framework exists.
		// switch between recommender and backoff strategy
		var rec PropertyRecommendations
		if runBackoffStrategy == "deletelowfrequency" {
			fmt.Println("Run with Delete Low Frequency Backoff Strategy instead of Recommender")
			b := backoffDeleteLowFrequencyItems{}
			b.init(schema, 5, stepsizeLinear)
			rec = b.recommend(list)
		} else if runBackoffStrategy == "splitProperties" {
			fmt.Println("Run with Split Properties Backoff Strategy instead of Recommender")
			b := backoffSplitPropertySet{}
			b.init(schema, twoSupportRangesSplitter, avgMerger)
			rec = b.recommend(list)
		} else {
			fmt.Println("Normal Recommendation")
			rec = schema.RecommendProperty(list)
		}

		fmt.Println(time.Since(t1))

		if len(rec) > 500 {
			rec = rec[:500]
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rec)
	}

	wikiRecommender := func(w http.ResponseWriter, r *http.Request) {
		var properties []string
		err := json.NewDecoder(r.Body).Decode(&properties)
		if err != nil {
			w.Write([]byte("Malformed Request. Expected an array of property IRIs"))
			return
		}
		// fmt.Println(properties)

		list := []*IItem{}
		for _, pString := range properties {
			p, ok := pMap["http://www.wikidata.org/prop/direct/"+pString]
			if ok {
				list = append(list, p)
			}
		}
		// fmt.Println(schema.Support(list), schema.Root.Support)

		t1 := time.Now()
		rec := schema.RecommendProperty(list)
		fmt.Println(time.Since(t1))

		res := []string{}
		for _, r := range rec {
			if strings.HasPrefix(*r.Property.Str, "http://www.wikidata.org/prop/direct/") {
				res = append(res, strings.TrimPrefix(*r.Property.Str, "http://www.wikidata.org/prop/direct/"))
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}

	http.HandleFunc("/recommender", recommender)
	http.HandleFunc("/wikiRecommender", wikiRecommender)
	go http.ListenAndServe(fmt.Sprintf("0.0.0.0:%v", port), nil)
	fmt.Printf("Now listening on port %v\n", port)
}