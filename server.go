package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func serve(schema *SchemaTree, port int) {
	dashboard := func(w http.ResponseWriter, r *http.Request) {
		var properties []string
		err := json.NewDecoder(r.Body).Decode(&properties)
		if err != nil {
			// w.Write([]byte("Malformed Request"))
			panic("Malformed Request")
		}
		fmt.Println(properties)

		rdftype := schema.propMap.get("http://www.w3.org/1999/02/22-rdf-syntax-ns#type")
		memberOf := schema.propMap.get("http://www.wikidata.org/prop/direct/P463")
		list := []*iItem{rdftype, memberOf}
		// fmt.Println(schema.Support(list), schema.Root.Support)

		t1 := time.Now()
		rec := schema.recommendProperty(list)
		fmt.Println(time.Since(t1))

		if len(rec) > 10 {
			rec = rec[:10]
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rec)
	}

	http.HandleFunc("/recommender", dashboard)
	go http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
	fmt.Printf("Now listening on port %v\n", port)
}
