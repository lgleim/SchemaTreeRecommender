package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"recommender/assessment"
	"recommender/glossary"
	"recommender/schematree"
	"recommender/strategy"
)

// RecommenderRequest is the data representation of the request input in json.
type RecommenderRequest struct {
	Lang       string   `json:"lang"`
	Types      []string `json:"types"`
	Properties []string `json:"properties"`
}

// RecommenderResponse is the data representation of the json.
type RecommenderResponse struct {
	Recommendations []RecommendationOutputEntry `json:"recommendations"`
}

// RecommendationOutputEntry is each entry that is return from the server.
type RecommendationOutputEntry struct {
	PropertyStr *string `json:"property"`
	Label       *string `json:"label"`
	Description *string `json:"description"`
	Probability float64 `json:"probability"`
}

// setupRecommender will setup a handler to recommend properties based on the list of properties and types. It
// also receives a language with which additional information is added.
// It will return an array of recommendations, with their respective probabilities, labels and descriptions.
func setupMappedRecommender(
	model *schematree.SchemaTree,
	glos *glossary.Glossary,
	workflow *strategy.Workflow,
	hardLimit int, // Hard limit of recommendations to output
) func(http.ResponseWriter, *http.Request) {

	// // Build the JSON-Schema
	// // TODO: Maybe it needs the '$schema' parameter, but not sure if draft-v7 uses that anymore.
	// var endpointSchemaJSON = []byte(`
	// 	{
	// 		"title": "SchemaTree Recommendation Request"
	// 		"type": "object",
	// 		"properties": {
	// 			"lang": {
	// 				"type": "string"
	// 			},
	// 			"types": {
	// 				"type" : "array",
	// 				"items" : {
	// 					"type": "string"
	// 				}
	// 			},
	// 			"properties": {
	// 				"type" : "array",
	// 				"items" : {
	// 					"type": "string"
	// 				}
	// 			}
	// 		},
	// 		"required": ["lang","types","properties"]
	// 	}
	// `)
	// endpointSchema := &jsonschema.RootSchema{}
	// if err := json.Unmarshal(endpointSchemaJSON, endpointSchema); err != nil {
	// 	panic("Unable to interpret the JSON-Schema for MappedRecommender endpoint: " + err.Error())
	// }
	// // Check that the request is conform to the JSON-Schema.
	// var body = []byte(`get this somehow from the request`)
	// if errors, _ := rs.ValidateBytes(valid); len(errors) > 0 {
	// 	panic(errors)
	// }

	return func(res http.ResponseWriter, req *http.Request) {

		// Decode the JSON input and build a list of input strings
		var input = RecommenderRequest{}
		err := json.NewDecoder(req.Body).Decode(&input)
		if err != nil {
			res.Write([]byte("Malformed Request.")) // TODO: Json-Schema helps
			return
		}
		fmt.Println(input) // debug: output the request

		// TODO: Probably some more input sanitization is required.

		// Make an assessment of the input properties.
		assessment := assessment.NewInstanceFromInput(input.Properties, input.Types, model, true)

		// Make a recommendation based on the assessed input and chosen strategy.
		t1 := time.Now()
		origRecs := workflow.Recommend(assessment)
		fmt.Println(time.Since(t1))

		// Put a hard limit on the recommendations returned.
		if len(origRecs) > hardLimit {
			origRecs = origRecs[:hardLimit]
		}

		// For each recommendation, add a mapping from the glossary.
		labRecs := glossary.TranslateRecommendations(glos, input.Lang, origRecs)

		// Prepare the recommendation list. The structure of the output is flatter than the labeled recommendations.
		outputRecs := make([]RecommendationOutputEntry, len(labRecs), len(labRecs))
		for i, rec := range labRecs {
			outputRecs[i].PropertyStr = rec.Property.Str
			outputRecs[i].Label = &rec.Content.Label
			outputRecs[i].Description = &rec.Content.Description
			outputRecs[i].Probability = rec.Probability
		}

		// Pack everything into the response
		recResp := RecommenderResponse{Recommendations: outputRecs}

		// Write the recommendations as a JSON array.
		res.Header().Set("Content-Type", "application/json")
		json.NewEncoder(res).Encode(recResp)
	}

}

// setupRecommender will setup a handler to recommend properties based on the list of properties and types.
// It will return an array of recommendations with their respective probabilities.
// No gloassary information is added to the response.
func setupLeanRecommender(tree *schematree.SchemaTree, workflow *strategy.Workflow) func(http.ResponseWriter, *http.Request) {

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

// setupSupportComputation will setup a handler that returns the percentage of all training sets that contained the given property combination.
func setupSupportComputation(tree *schematree.SchemaTree) func(http.ResponseWriter, *http.Request) {

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

		t1 := time.Now()

		// Match the input strings to build a list of input properties.
		list := []*schematree.IItem{}
		for _, pString := range properties {
			p, ok := pMap[pString]
			if ok {
				list = append(list, p)
			}
		}
		support := tree.Support(list)
		total := tree.Root.Support

		fmt.Println(support, "out of", total, "in", time.Since(t1))

		fraction := float64(support) / float64(total)
		json.NewEncoder(res).Encode(fraction)
	}
}

// hacked together for gregors thesis
// recommends both missing properties and missing types
func setupPropTypeRec(
	model *schematree.SchemaTree,
) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {

		// Decode the JSON input and build a list of input strings
		var input = RecommenderRequest{}
		err := json.NewDecoder(req.Body).Decode(&input)
		if err != nil {
			res.Write([]byte("Malformed Request.")) // TODO: Json-Schema helps
			return
		}
		fmt.Println(input) // debug: output the request

		// TODO: Probably some more input sanitization is required.

		// Make a recommendation based on the assessed input and chosen strategy.
		properties := model.BuildPropertyList(input.Properties, input.Types)
		t1 := time.Now()
		labRecs := model.RecommendPropertiesAndTypes(properties)
		fmt.Println(time.Since(t1))

		// Prepare the recommendation list. The structure of the output is flatter than the labeled recommendations.
		outputRecs := make([]RecommendationOutputEntry, len(labRecs), len(labRecs))
		for i, rec := range labRecs {
			// if rec.Property.IsType() {
			outputRecs[i].PropertyStr = rec.Property.Str
			outputRecs[i].Probability = rec.Probability
		}

		// Pack everything into the response
		recResp := RecommenderResponse{Recommendations: outputRecs}

		// Write the recommendations as a JSON array.
		res.Header().Set("Content-Type", "application/json")
		json.NewEncoder(res).Encode(recResp)
	}

}

// SetupEndpoints configures a router with all necessary endpoints and their corresponding handlers.
func SetupEndpoints(model *schematree.SchemaTree, glossary *glossary.Glossary, workflow *strategy.Workflow, hardLimit int) http.Handler {
	router := http.NewServeMux()
	router.HandleFunc("/lean-recommender", setupLeanRecommender(model, workflow))
	router.HandleFunc("/recommender", setupMappedRecommender(model, glossary, workflow, hardLimit))
	router.HandleFunc("/support", setupSupportComputation(model))
	router.HandleFunc("/propType", setupPropTypeRec(model))
	// router.HandleFunc("/wikiRecommender", wikiRecommender)
	return router
}
