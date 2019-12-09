package preparation

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	gzip "github.com/klauspost/pgzip"

	recIO "recommender/io"
)

// SplitBySampling splits a dataset file into two by taking out every Nth entry.
// Taken from the original splitter without modifications.
//
// Note that this method assumes that all subjects are defined in contiguous lines.
func SplitBySampling(fileName string, oneInN int64) error {

	// Set up file reader
	reader, err := recIO.UniversalReader(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	scanner := bufio.NewReaderSize(reader, 4*1024*1024) // 4MB line Buffer

	// Set up training set writer
	fName := recIO.TrimExtensions(fileName)
	trainingSet, err := os.Create(fName + "-1in" + strconv.FormatInt(oneInN, 10) + "-train.nt.gz")
	if err != nil {
		log.Fatal(err)
	}
	defer trainingSet.Close()

	wTrain, _ := gzip.NewWriterLevel(trainingSet, gzip.BestCompression)
	// wTrain := gozstd.NewWriterLevel(trainingSet, 19)
	// defer wTrain.Release()
	defer wTrain.Close()

	// Set up test set writer
	testSet, err := os.Create(fName + "-1in" + strconv.FormatInt(oneInN, 10) + "-test.nt.gz")
	if err != nil {
		log.Fatal(err)
	}
	defer testSet.Close()

	wTest, _ := gzip.NewWriterLevel(testSet, gzip.BestCompression)
	// wTest := gozstd.NewWriterLevel(testSet, 19)
	// defer wTest.Release()
	defer wTest.Close()

	// prepare dynamic writer switching
	var wRing uint16
	testModulo := uint16(oneInN)

	// parse file
	var isPrefix, skip bool
	var line, token []byte
	var lastSubj string
	var bytesProcessed int

	for line, isPrefix, err = scanner.ReadLine(); err == nil; line, isPrefix, err = scanner.ReadLine() {
		// skip overlong lines
		if isPrefix {
			fmt.Printf("Line Buffer too small!!! Line prefix: %v\n", string(line[:200]))
			skip = true
			continue
		}
		if skip { // !isPrefix follows implicitly
			skip = false
			continue
		}

		// extract subject
		bytesProcessed, token = firstWord(line)

		if len(token) == 0 || token[0] == '#' { // line is a comment
			continue
		}

		if lastSubj != string(token) { // Processing a new subject
			wRing = (wRing + 1) % testModulo
			lastSubj = string(token) // allocate string (on heap)
		}

		////// Wikidata specific processing ///// >>>>>
		// process predicate
		_, token = firstWord(line[bytesProcessed:])

		// c.f. https://www.mediawiki.org/wiki/Wikibase/Indexing/RDF_Dump_Format#Prefixes_used
		if strings.HasPrefix(string(token), "http://www.wikidata.org/prop/") &&
			!strings.HasPrefix(string(token), "http://www.wikidata.org/prop/direct/") {
			continue
		}
		////// Wikidata specific processing ///// <<<<<<

		if wRing == 0 {
			_, err = wTest.Write(line)
			io.WriteString(wTest, "\n")
		} else {
			_, err = wTrain.Write(line)
			io.WriteString(wTrain, "\n")
		}
		if err != nil {
			log.Fatal(err)
		}

	}

	if err != nil && err != io.EOF {
		log.Fatalf("Scanner encountered error while trying to parse triples: %v\n", err)
	}
	return nil
}

// Adapted from 'ScanWords' in https://golang.org/src/bufio/scan.go
//
// firstWord returns the first space-separated word of text, with
// surrounding spaces & angle brackets deleted.
func firstWord(data []byte) (advance int, token []byte) {
	// Skip leading spaces.
	start := 0
	for width := 0; start < len(data); start += width {
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
		case ' ', '\t', '\n', '\v', '\f', '\r', '<', '>':
			return true
		case '\u0085', '\u00A0':
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
