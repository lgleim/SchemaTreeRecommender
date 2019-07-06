package splitter

import (
	"bytes"

	recIO "recommender/io"
)

// SplitByType will take a dataset and generate smaller datasets for each subject type it finds.
// Types can be of following: item, property, other/miscellaneous
//
// TODO: Maybe there is a need to remove the type-classifying predicates. It that happens
//       then it should be made as an optional argument.
func SplitByType(filePath string) error {

	// Setup attributes of the wikidata ontology
	var wdTypePredicate = []byte("<http://www.w3.org/1999/02/22-rdf-syntax-ns#type>")
	var wdItemObject = []byte("<http://wikiba.se/ontology-beta#Item>")
	var wdPropObject = []byte("<http://www.wikidata.org/ontology#Property>")

	// Get a N-Triple parser for the input file.
	tParser, err := recIO.NewTripleParser(filePath)
	if err != nil {
		return err
	}
	defer tParser.Close()

	// Open 3 files, one to nest each type.
	const (
		miscBlock = iota // its 'misc' if no better type is found
		itemBlock = iota
		propBlock = iota
	)
	fileBase := recIO.TrimCompressionExtension(filePath)
	itemFile := recIO.CreateAndOpenWithGzip(fileBase + ".item.gz")
	defer itemFile.Close()
	propFile := recIO.CreateAndOpenWithGzip(fileBase + ".prop.gz")
	defer propFile.Close()
	miscFile := recIO.CreateAndOpenWithGzip(fileBase + ".misc.gz")
	defer miscFile.Close()

	// Go through all entries in blocks of subjects. All the entries are stored in a buffer and when a
	// type predicate is found it is noted so that the code knows where to send the block of entries.
	var tempBuffer bytes.Buffer
	var curBlockSubject []byte // subject of the current block, used to check if we proceeded to another block
	curBlockType := miscBlock  // type of the current block
	for trip, err := tParser.NextTriple(); err == nil; trip, err = tParser.NextTriple() {

		// Check if the subject of the block has changed, or it terminated.
		if trip == nil || !bytes.Equal(curBlockSubject, trip.Subject) {

			// Flush the buffer into one of the 3 files
			switch curBlockType {
			case itemBlock:
				tempBuffer.WriteTo(itemFile)
			case propBlock:
				tempBuffer.WriteTo(propFile)
			default:
				tempBuffer.WriteTo(miscFile)
			}

			// Set the new subject to identify this new block
			if trip != nil {
				curBlockSubject = trip.Subject
				curBlockType = miscBlock
			}
		}

		// Stop this loop if no more triples were read.
		if trip == nil {
			break
		}

		// While its a 'misc' block, we hope to find a predicate that identifies the type.
		if curBlockType == miscBlock && bytes.Equal(trip.Predicate, wdTypePredicate) {
			if bytes.Equal(trip.Object, wdItemObject) {
				curBlockType = itemBlock
			} else if bytes.Equal(trip.Object, wdPropObject) {
				curBlockType = propBlock
			}
		}

		// Put this in the buffer
		tempBuffer.Write(trip.Line)
		tempBuffer.Write([]byte("\r\n")) // have to write the newline
	}
	if err != nil {
		return err
	}

	return nil
}
