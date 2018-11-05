/**

I/O and RDF Parsing

*/

package main

import (
	"bufio"
	"compress/bzip2"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"unicode/utf8"

	gzip "github.com/klauspost/pgzip"
)

// All type annotations (types) and properties (properties) for a fixed subject
// equivalent of a transaction in frequent pattern mining
type subjectSummary struct {
	types      []*iType
	properties iList
}

func (subj *subjectSummary) String() string {
	var types, properties string
	for _, item := range subj.types {
		types += *item.Str + " "
	}
	for _, item := range subj.properties {
		properties += *item.Str + " "
	}
	return fmt.Sprintf("{\n  types:      [ %v ]\n  properties: [ %v ]\n}", types, properties)
}

// Reads a RDF Dataset from disk (Subject-gouped NTriples) and emits per-subject summaries
func subjectSummaryReader(
	fileName string,
	propMap *propMap,
	typeMap *typeMap,
	handler func(s subjectSummary),
	firstN uint64,
) {
	// setting up IO
	var scanner *bufio.Scanner
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		fmt.Println("Reading data from stdin")

		scanner = bufio.NewScanner(os.Stdin)
	} else {
		fmt.Printf("Reading data from file '%v'\n", fileName)

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
		scanner = bufio.NewScanner(reader)
	}

	// Parsing file
	var line, token []byte
	var lastSubj string
	var bytesProcessed int
	var subjectCount uint64
	pMap, tMap := *propMap, *typeMap
	rdfType := pMap.get("http://www.w3.org/1999/02/22-rdf-syntax-ns#type")
	summary := &subjectSummary{[]*iType{}, []*iItem{}}

	for scanner.Scan() {
		line = scanner.Bytes()

		// process subject
		bytesProcessed, token = firstWord(line)

		// if r, _ := utf8.DecodeRune(token); r == '#' { // line is a comment
		if token[0] == '#' { // line is a comment
			continue
		}

		// If this a new subject, emit the previous predicate set and start clean
		if lastSubj != string(token) { // should only be allocated on stack - c.f. https://github.com/golang/go/issues/11777
			if lastSubj != "" {
				go handler(*summary)
				if subjectCount++; firstN > 0 && subjectCount >= firstN {
					break
				}
			}

			lastSubj = string(token) // allocate string (on heap)
			summary = &subjectSummary{[]*iType{}, []*iItem{}}
		}

		// process predicate
		line = line[bytesProcessed:]
		bytesProcessed, token = firstWord(line)

		predicate := pMap.get(string(token))
		summary.properties = append(summary.properties, predicate)

		// rdf:type statements are also added to the types list
		if predicate == rdfType {
			// process object
			line = line[bytesProcessed:]
			bytesProcessed, token = firstWord(line)

			summary.types = append(summary.types, tMap.get(string(token)))
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Scanner encountered error while trying to parse triples: %v\n", err)
	}

}

// Adapted from 'ScanWords' in https://golang.org/src/bufio/scan.go
//
// firstWord returns the first space-separated word of text, with
// surrounding spaces & angle brackets deleted.
func firstWord(data []byte) (advance int, token []byte) {
	// Skip leading spaces.
	start := 0
	for width := 0; start < len(data); start += width {
		// var isSpace bool
		// if isSpace, width = nextRuneIsSpaceOrBracket(data[start:]); !isSpace {
		// 	break
		// }

		if data[start] < utf8.RuneSelf {
			if !isSpaceOrBracket(rune(data[start])) {
				break
			}
			width = 1
		} else {
			var r rune
			r, width = utf8.DecodeRune(data[start:])
			if !isSpaceOrBracket(r) {
				break
			}
		}
	}
	// Scan until space, marking end of word.
	for width, i := 0, start; i < len(data); i += width {
		// Fast path 1: ASCII.
		if data[i] < utf8.RuneSelf {
			if isSpaceOrBracket(rune(data[i])) {
				return i + 1, data[start:i]
			}
			width = 1
		} else {
			var r rune
			r, width = utf8.DecodeRune(data[i:])
			if isSpaceOrBracket(r) {
				return i + width, data[start:i]
			}
		}
	}
	// If we're at EOL, we have a final, non-empty, non-terminated word. Return it.
	if len(data) > start {
		return len(data), data[start:]
	}
	// Request more data.
	return start, nil
}

// Adapted from 'isSpace' in https://golang.org/src/bufio/scan.go
//
// isSpace reports whether the character is a Unicode white space character .
func isSpaceOrBracket(r rune) bool {
	if r <= '\u00FF' {
		// Obvious ASCII ones: \t through \r plus space. Plus two Latin-1 oddballs.
		switch r {
		case ' ', '\t', '\n', '\v', '\f', '\r':
			return true
		case '\u0085', '\u00A0':
			return true
		case '<', '>': // n-triples IRI brackets
			return true
		}
		return false
	}
	// High-valued ones.
	if '\u2000' <= r && r <= '\u200a' {
		return true
	}
	switch r {
	case '\u1680', '\u2028', '\u2029', '\u202f', '\u205f', '\u3000':
		return true
	}
	return false
}
