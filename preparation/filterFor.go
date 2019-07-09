package preparation

import (
	"bytes"

	recIO "recommender/io"
)

// FilterForSchematree creates a filtered version of a dataset to make it better for
// usage when building schematrees.
// TODO: In future, such hard-coded predicates should probably not exist.
func FilterForSchematree(filePath string) error {
	var removalPredicates = [][]byte{
		[]byte("<http://www.w3.org/1999/02/22-rdf-syntax-ns#type>"),
		[]byte("<http://www.w3.org/2000/01/rdf-schema#label>"),
		[]byte("<http://www.w3.org/2004/02/skos/core#prefLabel>"),
		[]byte("<http://www.w3.org/2004/02/skos/core#altLabel>"),
		[]byte("<http://schema.org/name>"),
		[]byte("<http://schema.org/description>"),
	}
	return filterByPredicate(filePath, removalPredicates)
}

// FilterForGlossary creates a filtered version of a dataset to make it better for
// usage when building glossaries.
func FilterForGlossary(filePath string) error {
	var removalPredicates = [][]byte{
		[]byte("<http://www.w3.org/1999/02/22-rdf-syntax-ns#type>"),
		[]byte("<http://www.w3.org/2000/01/rdf-schema#label>"),
		[]byte("<http://www.w3.org/2004/02/skos/core#prefLabel>"),
	}
	return filterByPredicate(filePath, removalPredicates)
}

// FilterForEvaluation creates a filtered version of a dataset to make it faster when
// executing the evaluation.
func FilterForEvaluation(filePath string) error {
	var removalPredicates = [][]byte{
		[]byte("<http://www.w3.org/1999/02/22-rdf-syntax-ns#type>"),
		[]byte("<http://www.w3.org/2000/01/rdf-schema#label>"),
		[]byte("<http://www.w3.org/2004/02/skos/core#prefLabel>"),
		[]byte("<http://www.w3.org/2004/02/skos/core#altLabel>"),
		[]byte("<http://schema.org/name>"),
		[]byte("<http://schema.org/description>"),
	}
	return filterByPredicate(filePath, removalPredicates)
}

// FilterByPredicate will create a filtered file by removing all entries that contain a predicate
// listed in the removelPredicates argument.
func filterByPredicate(filePath string, removalPredicates [][]byte) error {

	// Get a N-Triple parser for the input file.
	tParser, err := recIO.NewTripleParser(filePath)
	if err != nil {
		return err
	}
	defer tParser.Close()

	// Open file.
	fileBase := filePath // recIO.TrimCompressionExtension(filePath)
	filteredFile := recIO.CreateAndOpenWithGzip(fileBase + ".filtered.gz")
	defer filteredFile.Close()

	// Go through all entries in blocks of subjects.
	for trip, err := tParser.NextTriple(); trip != nil && err == nil; trip, err = tParser.NextTriple() {

		// Check if the subject of the block has changed, or it terminated.
		toRemove := false
		for _, pred := range removalPredicates {
			if bytes.Equal(pred, trip.Predicate) {
				toRemove = true
				break
			}
		}

		// Put it in the filtered file if not to be removed.
		if toRemove == false {
			filteredFile.Write(trip.Line)
			filteredFile.Write([]byte("\r\n")) // have to write the newline
		}
	}
	if err != nil {
		return err
	}

	return nil
}
