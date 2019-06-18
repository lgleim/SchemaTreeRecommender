package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"recommender/schematree"
	"recommender/strategy"
)

// TODO: In the future, the server should act as a main entrypoint for the application. Its main
// method should set the flags and port. Then it should build the strategy (or allow the user to
// set it using the flags or an configuration file), and then setup the endpoints from where it
// will serve that constructed strategy.
// The method `ServeCustomizedSchematree` that exists right now is a temporary pluggable method
// to follow the current modus-operandi.

// ServeCustomizedSchematree : Serve a modified version of the SchemaTree that includes some
// hard-coded customizations with backoff strategies. This is a temporary method to allow the
// integration of backoff strategies while the complete change to server/main is in development.
func ServeCustomizedSchematree(schema *schematree.SchemaTree, port int, strategyName string) {
	pMap := schema.PropMap

	// Build a strategy. In this case this will be hard-coded with some backoff strategies.
	var strat = strategy.MakePresetStrategy(strategyName, schema)

	// Setup the recommender using the strategy that was just built.
	recommender := func(w http.ResponseWriter, r *http.Request) {
		var properties []string
		err := json.NewDecoder(r.Body).Decode(&properties)
		if err != nil {
			w.Write([]byte("Malformed Request. Expected an array of property IRIs"))
			return
		}
		fmt.Println(properties)

		list := []*schematree.IItem{}
		for _, pString := range properties {
			p, ok := pMap[pString]
			if ok {
				list = append(list, p)
			}
		}
		// fmt.Println(schema.Support(list), schema.Root.Support)

		t1 := time.Now()
		rec := strat.Recommend(list)
		fmt.Println(time.Since(t1))

		if len(rec) > 500 {
			rec = rec[:500]
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rec)
	}

	// TODO: Like the current version of schematree/main, we will mimick the wiki recomender. In
	// the future, each endpoint should have their own method where they are constructed, all
	// orchestrated by the main method of server.
	wikiRecommender := func(w http.ResponseWriter, r *http.Request) {
		var properties []string
		err := json.NewDecoder(r.Body).Decode(&properties)
		if err != nil {
			w.Write([]byte("Malformed Request. Expected an array of property IRIs"))
			return
		}
		// fmt.Println(properties)

		list := []*schematree.IItem{}
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
