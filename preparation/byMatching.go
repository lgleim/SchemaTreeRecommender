package preparation

import (
	"bytes"

	recIO "recommender/io"
)

// SplitByPrefixStats are the stats related to the split operation.
// TODO: Maybe these types of Stats returns should also tell use where the files have been stored.
type SplitByPrefixStats struct {
	MiscCount int
	ItemCount int
	PropCount int
}

// SplitByPrefix will take a dataset and decide where to send it to based on a match of the
// beginning of the subject.
// Matches can be of following: item, property, other/miscellaneous
func SplitByPrefix(filePath string) (*SplitByPrefixStats, error) {
	stats := SplitByPrefixStats{}

	// Setup attributes of the wikidata ontology
	var wdItemSubjects = [][]byte{
		[]byte("<http://www.wikidata.org/entity/Q"),
	}
	var wdPropSubjects = [][]byte{
		[]byte("<http://www.wikidata.org/entity/P"),
	}

	// Get a N-Triple parser for the input file.
	tParser, err := recIO.NewTripleParser(filePath)
	if err != nil {
		return nil, err
	}
	defer tParser.Close()

	// Open 3 files, one to nest each type.
	const (
		miscBlock = iota // its 'misc' if no better type is found
		itemBlock = iota
		propBlock = iota
	)
	fileBase := filePath // recIO.TrimCompressionExtension(filePath)

	itemFile := recIO.CreateAndOpenWithGzip(fileBase + ".item.gz")
	defer itemFile.Close()

	propFile := recIO.CreateAndOpenWithGzip(fileBase + ".prop.gz")
	defer propFile.Close()

	miscFile := recIO.CreateAndOpenWithGzip(fileBase + ".misc.gz")
	defer miscFile.Close()

	// Go through all entries and decide on a line-by-line basis.
	for trip, err := tParser.NextTriple(); trip != nil && err == nil; trip, err = tParser.NextTriple() {

		// We can check for equality in the first bytes instead of using actual regex or unicode.
		if startsWithOneOf(trip.Subject, wdItemSubjects) {
			itemFile.Write(trip.Line)
			itemFile.Write([]byte("\r\n")) // have to write the newline
			stats.ItemCount++
		} else if startsWithOneOf(trip.Subject, wdPropSubjects) {
			propFile.Write(trip.Line)
			propFile.Write([]byte("\r\n")) // have to write the newline
			stats.PropCount++
		} else {
			miscFile.Write(trip.Line)
			miscFile.Write([]byte("\r\n")) // have to write the newline
			stats.MiscCount++
		}

	}
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// startsWithOneOf will check if the beginning of a byte-array is equal to one of the byte-arrays
// provided in a list.
func startsWithOneOf(needle []byte, haystack [][]byte) bool {
	for _, hayball := range haystack {
		if bytes.HasPrefix(needle, hayball) {
			return true
		}
	}
	return false
}
