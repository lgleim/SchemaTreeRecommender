/**

I/O and RDF Parsing

*/

package main

import (
	"compress/bzip2"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	gzip "github.com/klauspost/pgzip"
	"github.com/knakk/rdf"
)

// All type annotations (types) and properties (properties) for a fixed subject
// equivalent of a transaction in frequent pattern mining
type subjectSummary struct {
	types      []*string
	properties []*string
}

func (subj *subjectSummary) String() string {
	mapper := func(strings []*string) string {
		res := "[ "
		for _, s := range strings {
			res += *s + " "
		}
		return res + "]"
	}
	return fmt.Sprintf("{\n  types:      %v\n  properties: %v\n}", mapper(subj.types), mapper(subj.properties))
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

		var reader io.Reader = file
		switch ext := filepath.Ext(fileName); ext {
		case ".bz2":
			reader = bzip2.NewReader(reader) // Decompression
		case ".gz":
			reader, err = gzip.NewReader(reader)
			if err != nil {
				log.Fatal(err)
			}
		}

		tripleDecoder := rdf.NewTripleDecoder(reader, rdf.NTriples) // RDF parsing

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
