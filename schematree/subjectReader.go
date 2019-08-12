/**

I/O and RDF Parsing

*/

package schematree

import (
	"bufio"
	"compress/bzip2"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	rio "recommender/io"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	gzip "github.com/klauspost/pgzip"

	"github.com/biogo/hts/bgzf"
	pb "gopkg.in/cheggaaa/pb.v1"
)

// All type annotations (types) and properties (properties) for a fixed subject
// equivalent of a transaction in frequent pattern mining
type SubjectSummary struct {
	Properties        map[*IItem]uint32
	Str               string // @TODO: Temporarily added the subject names for easier evaluation debugging
	NumPredicates     int
	NumTypePredicates int
}

func (subj *SubjectSummary) String() string {
	var properties string
	for item := range subj.Properties {
		properties += *item.Str + " "
	}
	return fmt.Sprintf("{\n  types:      [ %v ]\n  properties: [ %v ]\n}", 0, len(subj.Properties)) //TODO count types
}

// SubjectSummaryReader reads a RDF Dataset from disk (in N-Triples format) which is expected to be
// grouped by subjects. For each subject group, the method will build a SubjectSummary structure and
// send it to a handler function.
// It will always detect types, but may choose to ignore them.
//
// todo: The parsing is done a subset of N-Triple format files. If subjects, predicates or objects contain
//       any spaces, even if inside quotes, it will break.
func SubjectSummaryReader(
	fileName string, // path to the file that should be parsed
	pMap propMap, // maps of properties that the schematree recognizes
	handler func(s *SubjectSummary), // handler function that gets executed after a SubjectSummary is completed
	firstN uint64, // stop after N subjects are read; setting this to zero will read all entries
	willConvertTypes bool, // true if the reader should convert identified type entries into TypeProperties.
) (subjectCount uint64) {
	// IO setup
	reader, err := rio.UniversalReader(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	// set up concurrent handler routines
	concurrency := runtime.NumCPU() // * 4    (should be fine with NumCPU since thats num of logical cpus and has no IO operation)
	summaries := make(chan *SubjectSummary)
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			for s := range summaries {
				handler(s)
			}
			wg.Done()
		}()
	}

	// parse file
	var isPrefix, skip bool
	var line, token []byte
	var lastSubj string
	var bytesProcessed int
	scanner := bufio.NewReaderSize(reader, 4*1024*1024) // 4MB line Buffer
	var summary *SubjectSummary
	//summary := &SubjectSummary{Properties: make(map[*IItem]uint32)}
	typeProps := []*IItem{pMap.get("http://www.wikidata.org/prop/direct/P31")}

	for line, isPrefix, err = scanner.ReadLine(); err == nil; line, isPrefix, err = scanner.ReadLine() {
		if isPrefix {
			fmt.Printf("Line Buffer too small!!! Line prefix: %v\n", string(line[:200]))
			skip = true
			continue
		}
		if skip { // !isPrefix follows implicitly
			skip = false
			continue
		}

		// process subject
		bytesProcessed, token = firstWord(line)

		// if r, _ := utf8.DecodeRune(token); r == '#' { // line is a comment
		if len(token) == 0 || token[0] == '#' { // line is a comment
			continue
		}

		// If this a new subject, emit the previous predicate set and start clean
		if lastSubj != string(token) { // should only be allocated on stack - c.f. https://github.com/golang/go/issues/11777
			if lastSubj != "" {
				summaries <- summary
				if subjectCount++; firstN > 0 && subjectCount >= firstN {
					break
				}
			}

			lastSubj = string(token) // allocate string (on heap)
			summary = &SubjectSummary{Properties: make(map[*IItem]uint32), Str: lastSubj}
		}

		// process predicate
		line = line[bytesProcessed:]
		bytesProcessed, token = firstWord(line)

		// c.f. https://www.mediawiki.org/wiki/Wikibase/Indexing/RDF_Dump_Format#Prefixes_used
		if strings.HasPrefix(string(token), "http://www.wikidata.org/prop/") &&
			!strings.HasPrefix(string(token), "http://www.wikidata.org/prop/direct/") {
			continue
		}

		predicate := pMap.get(string(token))

		summary.Properties[predicate]++

		// Count the number of predicates found for that subject. Unfortunately
		// it is NOT the number of unique predicates. Having multiple equal
		// predicates would be better but there is no easy way to calculate the
		// number of unique type predicates without actually converting and
		// storing them into the treeMap.
		// This might make some impact because the TypeProp is usually used multiple
		// times, one for each type the subject has.
		summary.NumPredicates++

		// Detect type properties to add them to the counters.
		for _, typeProp := range typeProps {
			if predicate == typeProp {
				summary.NumTypePredicates++

				// If set to convert types, then read the object to generate a type property from it.
				if willConvertTypes {
					line = line[bytesProcessed:]
					bytesProcessed, token = firstWord(line)
					tokenStr := "t#" + string(token) // prefix t# identifies properties that represent types
					pType := pMap.get(tokenStr)
					summary.Properties[pType]++
				}
				break
			}
		}
	}

	// dispatch last summary
	if len(summary.Properties) > 0 {
		summaries <- summary
	}

	if err != nil && err != io.EOF {
		log.Fatalf("Scanner encountered error while trying to parse triples: %v\n", err)
	}
	close(summaries)
	wg.Wait()

	return
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

// UniversalReader opens a file for read mode. It is able to automatically decompress
// files that use either GZ, BGZ or BZ2. It will also display a loading bar on stdout.
func UniversalReader(fileName string) (reader io.ReadCloser, err error) {
	if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) == 0 {
		fmt.Println("Reading data from stdin")
		reader = os.Stdin
	} else {
		var file *os.File
		file, err = os.Open(fileName)
		if err != nil {
			return
		}

		var stat os.FileInfo
		stat, err = file.Stat()
		if err != nil {
			return
		}
		if stat.IsDir() {
			err = errors.New("Reading entire directories is not yet possible")
			return
		}
		fmt.Printf("Reading data from file '%v'. Progress w.r.t. on-disk-size: \n", fileName)

		// create and start progress bar
		bar := pb.New(int(stat.Size())).SetUnits(pb.U_BYTES).SetRefreshRate(500 * time.Millisecond).Start()
		bar.ShowElapsedTime = true
		bar.ShowSpeed = true
		reader = bar.NewProxyReader(file)

		// decompress stream if applicable
		switch ext := filepath.Ext(fileName); ext {
		case ".bz2":
			reader = ioutil.NopCloser(bzip2.NewReader(reader)) // Decompression
		case ".gz":
			reader, err = gzip.NewReaderN(reader, 8*1024*1024, 48) // readahead
		case ".bgz":
			reader, err = bgzf.NewReader(reader, 0)
			//case ".zst", ".zstd":
			//reader = releaseCloser{gozstd.NewReader(reader)}
		}

		if err != nil {
			return
		}

		reader = finishCloser{reader, bar, file}
	}
	return
}

type finishCloser struct {
	io.ReadCloser
	bar  *pb.ProgressBar
	file *os.File
}

func (r finishCloser) Close() error {
	r.bar.Finish()
	defer r.file.Close()
	return r.ReadCloser.Close()
}
