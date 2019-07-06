package glossary

import (
	"bytes"
	"fmt"
	"recommender/io"
	recIO "recommender/io"
)

// Key of each glossary entry.
type Key struct {
	Property string
	Lang     string
}

// Content of each glossary entry. Identifiers are usually supplied in the map.
type Content struct {
	Label       string
	Description string
}

// Glossary holds an entire glossary.
type Glossary map[Key]*Content // glossary[property,language]

// BuildGlossary from a dataset of N-Triples
// todo: Should this method receive the filepath, a filehandler, or a tripleparser?
func BuildGlossary(filePath string) (*Glossary, error) {

	// Setup property types of the wikidata ontology
	var wdLabelPredicate = []byte("<http://www.w3.org/2000/01/rdf-schema#label>")
	var wdDescriptionPredicate = []byte("<http://schema.org/description>")

	// Get a N-Triple parser for the input file.
	tParser, err := recIO.NewTripleParser(filePath)
	if err != nil {
		return nil, err
	}
	defer tParser.Close()

	// Initialize the glossary that is to be returned.
	glos := make(Glossary)

	// Initialize the 3 type of triples that can be found: label, description, other
	const (
		miscType        = iota // its 'misc' if no better type is found
		labelType       = iota
		descriptionType = iota
	)

	// Go through each triple and add it to the glossary, while also creating entries
	// on-the-fly if they don't exist.
	for trip, err := tParser.NextTriple(); trip != nil && err == nil; trip, err = tParser.NextTriple() {

		// Get the predicate and make sure its either a label or description.
		thisType := miscType
		if bytes.Equal(trip.Predicate, wdLabelPredicate) {
			thisType = labelType
		} else if bytes.Equal(trip.Predicate, wdDescriptionPredicate) {
			thisType = descriptionType
		} else { // skip if type is not important
			continue
		}

		// Get the text and language of the triple object.
		text, lang := io.InterpreteLangLiteral(trip.Object)
		if len(text) == 0 || len(lang) == 0 { // Only accept entries where both text and lang exist.
			continue
		}

		// Create the entry if it doesn't exist yet.
		thisKey := &Key{string(trip.Subject), string(lang)}
		thisContent, thisContentOk := glos[*thisKey]
		if !thisContentOk {
			thisContent = &Content{}
			glos[*thisKey] = thisContent
		}

		// Add the information of this triple to the glossary.
		if thisType == labelType {
			thisContent.Label = string(text)
		} else if thisType == descriptionType {
			thisContent.Description = string(text)
		}

	}
	if err != nil {
		return nil, err
	}

	return &glos, nil
}

// Output the glossary to stdout.
func (glos *Glossary) Output() {
	fmt.Println(*glos)
	// b, err := json.MarshalIndent(*glos, "", "  ")
	// if err == nil {
	// 	fmt.Println(string(b))
	// } else {
	// 	fmt.Println(err)
	// }
	return
}