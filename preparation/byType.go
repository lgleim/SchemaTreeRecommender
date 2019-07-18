package preparation

import (
	"bytes"

	recIO "recommender/io"
)

// SplitByTypeStats are the stats related to the split operation.
// TODO: Maybe these types of Stats returns should also tell use where the files have been stored.
type SplitByTypeStats struct {
	MiscCount int
	ItemCount int
	PropCount int
}

// SplitByType will take a dataset and generate smaller datasets for each subject type it finds.
// Types can be of following: item, property, other/miscellaneous.
func SplitByType(filePath string) (*SplitByTypeStats, error) {
	stats := SplitByTypeStats{}

	// Setup attributes of the wikidata ontology
	var wdTypePredicate = []byte("<http://www.w3.org/1999/02/22-rdf-syntax-ns#type>")
	var wdItemObjects = [][]byte{
		[]byte("<http://wikiba.se/ontology#Item>"),
		[]byte("<http://wikiba.se/ontology-beta#Item>"), // included for retro-compatibility
	}
	var wdPropObjects = [][]byte{
		[]byte("<http://wikiba.se/ontology#Property>"),
		[]byte("<http://www.wikidata.org/ontology#Property>"), // included for retro-compatibility
	}

	// Define a N-Triple parser for the input file.
	// The method will open the file twice because of the multiple read-interfaces applied on file.
	var tParser *recIO.TripleParser
	var err error

	// Define the possible types.
	const (
		miscBlock = iota // its 'misc' if no better type is found
		itemBlock = iota
		propBlock = iota
	)

	// Open the file for the first passthrough.
	tParser, err = recIO.NewTripleParser(filePath)
	if err != nil {
		tParser.Close()
		return nil, err
	}

	// Will perform one pass on the entire file to identify subjects and categorize them.
	subjectTypeMap := map[string]int{}
	for trip, err := tParser.NextTriple(); trip != nil && err == nil; trip, err = tParser.NextTriple() {

		// Identify if this entry is trying to describe a type, and get that type.
		// Only register the mapping if the type is 'item' or 'prop. We do not need to store mappings for 'misc'.
		// Also, the same predicate can occur multiple times and with values we ignore. Only take valid objects.
		if bytes.Equal(trip.Predicate, wdTypePredicate) {
			if equalToOneOf(trip.Object, wdItemObjects) {
				subjectTypeMap[string(trip.Subject)] = itemBlock
			} else if equalToOneOf(trip.Object, wdPropObjects) {
				subjectTypeMap[string(trip.Subject)] = propBlock
			}
		}
	}
	if err != nil {
		tParser.Close()
		return nil, err
	}
	tParser.Close()

	// Open 3 files, one to nest each type.
	fileBase := recIO.TrimExtensions(filePath)
	itemFile := recIO.CreateAndOpenWithGzip(fileBase + "-item.nt.gz")
	defer itemFile.Close()
	propFile := recIO.CreateAndOpenWithGzip(fileBase + "-prop.nt.gz")
	defer propFile.Close()
	miscFile := recIO.CreateAndOpenWithGzip(fileBase + "-misc.nt.gz")
	defer miscFile.Close()

	// Open the file for the second passthrough.
	tParser, err = recIO.NewTripleParser(filePath)
	if err != nil {
		tParser.Close()
		return nil, err
	}

	// On the second pass, go through all entries and send them to their file according to the mapping.
	for trip, err := tParser.NextTriple(); trip != nil && err == nil; trip, err = tParser.NextTriple() {
		mapType, mapOk := subjectTypeMap[string(trip.Subject)]

		// Mapped items are guaranteed to have correct block. Misc if no mapping found.
		var curBlockType int
		if mapOk {
			curBlockType = mapType
		} else {
			curBlockType = miscBlock
		}

		// Flush the buffer into one of the 3 files
		switch curBlockType {
		case itemBlock:
			itemFile.Write(trip.Line)
			itemFile.Write([]byte("\r\n"))
			stats.ItemCount++
		case propBlock:
			propFile.Write(trip.Line)
			propFile.Write([]byte("\r\n"))
			stats.PropCount++
		default:
			miscFile.Write(trip.Line)
			miscFile.Write([]byte("\r\n"))
			stats.MiscCount++
		}
	}
	if err != nil {
		tParser.Close()
		return nil, err
	}
	tParser.Close()

	return &stats, nil
}

// SplitByTypeInBlocks is a faster implementation of SplitByType, using only a single pass, but assumes
// that subjects are always found in contiguous lines.
//
// TODO: Maybe there is a need to remove the type-classifying predicates. It that happens
//       then it should be made as an optional argument.
func SplitByTypeInBlocks(filePath string) (*SplitByTypeStats, error) {
	stats := SplitByTypeStats{}

	// Setup attributes of the wikidata ontology
	var wdTypePredicate = []byte("<http://www.w3.org/1999/02/22-rdf-syntax-ns#type>")
	var wdItemObjects = [][]byte{
		[]byte("<http://wikiba.se/ontology#Item>"),
		[]byte("<http://wikiba.se/ontology-beta#Item>"), // included for retro-compatibility
	}
	var wdPropObjects = [][]byte{
		[]byte("<http://wikiba.se/ontology#Property>"),
		[]byte("<http://www.wikidata.org/ontology#Property>"), // included for retro-compatibility
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
	fileBase := recIO.TrimExtensions(filePath)

	itemFile := recIO.CreateAndOpenWithGzip(fileBase + "-item.nt.gz")
	defer itemFile.Close()

	propFile := recIO.CreateAndOpenWithGzip(fileBase + "-prop.nt.gz")
	defer propFile.Close()

	miscFile := recIO.CreateAndOpenWithGzip(fileBase + "-misc.nt.gz")
	defer miscFile.Close()

	// Go through all entries in blocks of subjects. All the entries are stored in a buffer and when a
	// type predicate is found it is noted so that the code knows where to send the block of entries.
	var tempBuffer bytes.Buffer
	var curBlockSubject []byte // subject of the current block, used to check if we proceeded to another block
	tempCount := 0
	curBlockType := miscBlock // type of the current block
	for trip, err := tParser.NextTriple(); err == nil; trip, err = tParser.NextTriple() {

		// Check if the subject of the block has changed, or it terminated.
		if trip == nil || !bytes.Equal(curBlockSubject, trip.Subject) {

			// Flush the buffer into one of the 3 files
			switch curBlockType {
			case itemBlock:
				tempBuffer.WriteTo(itemFile)
				stats.ItemCount += tempCount
			case propBlock:
				tempBuffer.WriteTo(propFile)
				stats.PropCount += tempCount
			default:
				tempBuffer.WriteTo(miscFile)
				stats.MiscCount += tempCount
			}

			// Set the new subject to identify this new block
			if trip != nil {
				curBlockSubject = trip.Subject
				curBlockType = miscBlock
				tempCount = 0
			}
		}

		// Stop this loop if no more triples were read.
		if trip == nil {
			break
		}

		// While its a 'misc' block, we hope to find a predicate that identifies the type.
		if curBlockType == miscBlock && bytes.Equal(trip.Predicate, wdTypePredicate) {
			if equalToOneOf(trip.Object, wdItemObjects) {
				curBlockType = itemBlock
			} else if equalToOneOf(trip.Object, wdPropObjects) {
				curBlockType = propBlock
			}
		}

		// Put this in the buffer
		tempBuffer.Write(trip.Line)
		tempBuffer.Write([]byte("\r\n")) // have to write the newline
		tempCount++
	}
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// equalToOneOf will check if a byte-array is equal to one of the byte-arrays provided in a list.
func equalToOneOf(needle []byte, haystack [][]byte) bool {
	for _, hayball := range haystack {
		if bytes.Equal(needle, hayball) {
			return true
		}
	}
	return false
}
