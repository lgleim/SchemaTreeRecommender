package glossary

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
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
	var wdLabelPredicate = []byte("<http://schema.org/name>")
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

		// IRIREFs get stripped of their enclosing '< >' when they are stored.
		// TODO: See if the entire system (schematree as well) works with or without enclosing tags.
		iri := io.InterpreteIriRef(trip.Subject)

		// Create the entry if it doesn't exist yet.
		thisKey := &Key{string(iri), string(lang)}
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

// OutputStats of the glossary to stdout.
func (glos *Glossary) OutputStats() {
	fmt.Printf("Glossary: numEntries = %d\n", len(*glos))
	return
}

// WriteToFile will serialize the glossary into a binary file.
func (glos *Glossary) WriteToFile(path string) {
	f, _ := os.Create(path)
	e := gob.NewEncoder(f)
	e.Encode(*glos)
	f.Close()
}

// ReadFromFile reads a binary file and de-serializes it into a glossary.
func ReadFromFile(path string) (*Glossary, error) {
	var glos *Glossary
	f, err := os.Open(path)
	if err != nil {
		return nil, err // "Failed to open file!"
	}
	decoder := gob.NewDecoder(f)
	err = decoder.Decode(&glos)
	if err != nil {
		return nil, err // "Failed to decode glossary!"
	}
	f.Close()
	return glos, nil
}
