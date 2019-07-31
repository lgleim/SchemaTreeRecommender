package preparation

import (
	"bytes"

	recIO "recommender/io"
)

// FilterForSchematree creates a filtered version of a dataset to make it better for
// usage when building schematrees.
//
// todo: In future, such hard-coded predicates should probably not exist.
func FilterForSchematree(filePath string) (*FilterStats, error) {
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
//
// todo: In the future it could use a filter-in mechanism where only specific predicates
//       are sent to the generated file, instead of filter-out which includes all the
//       statements except the ones listed. Filter-in should use the same predicates that
//       are used by the Glossary building step and gives the user a better perception
//       of what is actually used by the glossary. With filter-in, we know that every
//       statement in our generated file is also used in the construction of the glossary.
//       With filter-out there can still be many statements that are silently ignored by
//       the building step.
func FilterForGlossary(filePath string) (*FilterStats, error) {
	var removalPredicates = [][]byte{
		[]byte("<http://www.w3.org/1999/02/22-rdf-syntax-ns#type>"),
		[]byte("<http://www.w3.org/2000/01/rdf-schema#label>"),
		[]byte("<http://www.w3.org/2004/02/skos/core#prefLabel>"),
	}
	return filterByPredicate(filePath, removalPredicates)
}

// FilterForEvaluation creates a filtered version of a dataset to make it faster when
// executing the evaluation.
func FilterForEvaluation(filePath string) (*FilterStats, error) {
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

// FilterStats are the stats related to the filter operation.
// TODO: Maybe these types of Stats returns should also tell use where the files have been stored.
type FilterStats struct {
	KeptCount int
	LostCount int
}

// FilterByPredicate will create a filtered file by removing all entries that contain a predicate
// listed in the removelPredicates argument.
func filterByPredicate(filePath string, removalPredicates [][]byte) (*FilterStats, error) {
	stats := FilterStats{}

	// Get a N-Triple parser for the input file.
	tParser, err := recIO.NewTripleParser(filePath)
	if err != nil {
		return nil, err
	}
	defer tParser.Close()

	// Open file.
	fileBase := recIO.TrimExtensions(filePath)
	filteredFile := recIO.CreateAndOpenWithGzip(fileBase + "-filtered.nt.gz")
	defer filteredFile.Close()

	// Go through all entries in blocks of subjects.
	for trip, err := tParser.NextTriple(); trip != nil && err == nil; trip, err = tParser.NextTriple() {

		// Check if the subject of the block has changed, or it terminated.
		toRemove := false
		for _, pred := range removalPredicates {
			if bytes.Equal(pred, trip.Predicate) {
				toRemove = true
				stats.LostCount++
				break
			}
		}

		// Put it in the filtered file if not to be removed.
		if toRemove == false {
			filteredFile.Write(trip.Line)
			filteredFile.Write([]byte("\r\n")) // have to write the newline
			stats.KeptCount++
		}
	}
	if err != nil {
		return nil, err
	}

	return &stats, nil
}
