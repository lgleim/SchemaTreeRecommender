package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"recommender/assessment"
	"recommender/schematree"
	"recommender/strategy"
)

func setupRecommenderHandler(tree *schematree.SchemaTree, workflow *strategy.Workflow) func(http.ResponseWriter, *http.Request) {

	// Fetch the map of all properties in the SchemaTree
	pMap := tree.PropMap

	return func(res http.ResponseWriter, req *http.Request) {

		// Decode the JSON input and build a list of input strings
		var properties []string
		err := json.NewDecoder(req.Body).Decode(&properties)
		if err != nil {
			res.Write([]byte("Malformed Request. Expected an array of property IRIs"))
			return
		}
		fmt.Println(properties)

		// Match the input strings to build a list of input properties.
		list := []*schematree.IItem{}
		for _, pString := range properties {
			p, ok := pMap[pString]
			if ok {
				list = append(list, p)
			}
		}
		// fmt.Println(tree.Support(list), tree.Root.Support)

		// Make an assessment of the input properties.
		assessment := assessment.NewInstance(list, tree, true)

		// Make a recommendation based on the assessed input and chosen strategy.
		t1 := time.Now()
		rec := workflow.Recommend(assessment)
		fmt.Println(time.Since(t1))

		// Put a hard limit on the recommendations returned.
		if len(rec) > 500 {
			rec = rec[:500]
		}

		// Write the recommendations as a JSON array.
		res.Header().Set("Content-Type", "application/json")
		json.NewEncoder(res).Encode(rec)
	}
}

// SetupEndpoints configures a router with all necessary endpoints and their corresponding handlers.
func SetupEndpoints(tree *schematree.SchemaTree, workflow *strategy.Workflow) http.Handler {
	router := http.NewServeMux()
	router.HandleFunc("/recommender", setupRecommenderHandler(tree, workflow))
	// router.HandleFunc("/wikiRecommender", wikiRecommender)
	return router
}

// ServeCustomizedSchematree : Serve a modified version of the SchemaTree that includes some
// hard-coded customizations with backoff strategies. This is a temporary method to allow the
// integration of backoff strategies while the complete change to server/main is in development.

// TODO: Right now, this method is shut-down because I need some more info about wiki-recommender.

// func ServeCustomizedSchematree(schema *schematree.SchemaTree, port int, strategyName string) {
// 	pMap := schema.PropMap

// 	// Build a strategy. In this case this will be hard-coded with some backoff strategies.
// 	var strat = strategy.MakePresetStrategy(strategyName, schema)

// 	// Setup the recommender using the strategy that was just built.
// 	recommender := func(w http.ResponseWriter, r *http.Request) {
// 		var properties []string
// 		err := json.NewDecoder(r.Body).Decode(&properties)
// 		if err != nil {
// 			w.Write([]byte("Malformed Request. Expected an array of property IRIs"))
// 			return
// 		}
// 		fmt.Println(properties)

// 		list := []*schematree.IItem{}
// 		for _, pString := range properties {
// 			p, ok := pMap[pString]
// 			if ok {
// 				list = append(list, p)
// 			}
// 		}
// 		// fmt.Println(schema.Support(list), schema.Root.Support)

// 		// Make an assessment of the input properties.
// 		assessment := assessment.NewInstance(list, schema, true)

// 		t1 := time.Now()
// 		rec := strat.Recommend(assessment)
// 		fmt.Println(time.Since(t1))

// 		if len(rec) > 500 {
// 			rec = rec[:500]
// 		}

// 		w.Header().Set("Content-Type", "application/json")
// 		json.NewEncoder(w).Encode(rec)
// 	}

// 	// TODO: Like the current version of schematree/main, we will mimick the wiki recomender. In
// 	// the future, each endpoint should have their own method where they are constructed, all
// 	// orchestrated by the main method of server.
// 	wikiRecommender := func(w http.ResponseWriter, r *http.Request) {
// 		var properties []string
// 		err := json.NewDecoder(r.Body).Decode(&properties)
// 		if err != nil {
// 			w.Write([]byte("Malformed Request. Expected an array of property IRIs"))
// 			return
// 		}
// 		// fmt.Println(properties)

// 		list := []*schematree.IItem{}
// 		for _, pString := range properties {
// 			p, ok := pMap["http://www.wikidata.org/prop/direct/"+pString]
// 			if ok {
// 				list = append(list, p)
// 			}
// 		}
// 		// fmt.Println(schema.Support(list), schema.Root.Support)

// 		t1 := time.Now()
// 		rec := schema.RecommendProperty(list)
// 		fmt.Println(time.Since(t1))

// 		res := []string{}
// 		for _, r := range rec {
// 			if strings.HasPrefix(*r.Property.Str, "http://www.wikidata.org/prop/direct/") {
// 				res = append(res, strings.TrimPrefix(*r.Property.Str, "http://www.wikidata.org/prop/direct/"))
// 			}
// 		}

// 		w.Header().Set("Content-Type", "application/json")
// 		json.NewEncoder(w).Encode(res)
// 	}

// 	http.HandleFunc("/recommender", recommender)
// 	http.HandleFunc("/wikiRecommender", wikiRecommender)
// 	go http.ListenAndServe(fmt.Sprintf("0.0.0.0:%v", port), nil)
// 	fmt.Printf("Now listening on port %v\n", port)
// }
