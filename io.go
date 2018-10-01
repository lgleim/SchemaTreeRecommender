/**

I/O and RDF Parsing

*/

package main

import (
	"compress/bzip2"
	"io"
	"log"
	"os"

	"github.com/knakk/rdf"
)

// All type annotations (types) and properties (properties) for a fixed subject
// equivalent of a transaction in frequent pattern mining
type subjectSummary struct {
	types      []*string
	properties []*string
}

// Reads a RDF Dataset from disk (Subject-gouped NTriples) and emits per-subject summaries
func subjectSummaryReader(fileName string) chan *subjectSummary {
	c := make(chan *subjectSummary)
	go func() {
		file, err := os.Open(fileName)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		bzipReader := bzip2.NewReader(file)                             // Decompression
		tripleDecoder := rdf.NewTripleDecoder(bzipReader, rdf.NTriples) // RDF parsing

		var lastSubj string
		var properties, rdfTypes []*string

		for triple, err := tripleDecoder.Decode(); err != io.EOF; triple, err = tripleDecoder.Decode() {
			if err != nil {
				log.Fatal(err)
			}

			// Skip blank nodes / literals in predicate position
			if triple.Pred.Type() != rdf.TermIRI {
				continue
			}

			currSubj := triple.Subj.String()
			if currSubj != lastSubj { // If this a new subject, emit the previous predicate set and start clean
				if properties != nil {
					c <- &subjectSummary{rdfTypes, properties}
				}

				lastSubj = currSubj
				properties = []*string{}
				rdfTypes = []*string{}
			}

			predIri := triple.Pred.String()
			properties = append(properties, &predIri)

			// also add rdf:type statements to the types list
			if predIri == "http://www.w3.org/1999/02/22-rdf-syntax-ns#type" {
				typeIri := triple.Obj.String()
				rdfTypes = append(rdfTypes, &typeIri)
			}
		}
		close(c)
	}()
	return c
}
