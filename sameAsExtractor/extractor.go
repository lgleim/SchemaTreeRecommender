package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"recommender/schematree"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strings"
	"unicode/utf8"

	gzip "github.com/klauspost/pgzip"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	fileName := flag.String("file", "experiments/10M.nt.gz", "the file to parse")
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile := flag.String("memprofile", "", "write memory profile to `file`")
	traceFile := flag.String("trace", "", "write execution trace to `file`")

	// parse commandline arguments/flags
	flag.Parse()

	// write cpu profile to file
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	// write cpu profile to file
	if *memprofile != "" {
		defer func() {
			f, err := os.Create(*memprofile)
			if err != nil {
				log.Fatal("could not create memory profile: ", err)
			}
			runtime.GC() // get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatal("could not write memory profile: ", err)
			}
			f.Close()
		}()
	}

	// write cpu profile to file
	if *traceFile != "" {
		f, err := os.Create(*traceFile)
		if err != nil {
			log.Fatal("could not create trace file: ", err)
		}
		if err := trace.Start(f); err != nil {
			log.Fatal("could not start tracing: ", err)
		}
		defer trace.Stop()
	}

	// Set up file reader
	reader, err := schematree.UniversalReader(*fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	scanner := bufio.NewReaderSize(reader, 4*1024*1024) // 4MB line Buffer

	// Set up writer
	fName := strings.TrimSuffix(*fileName, filepath.Ext(*fileName))
	f, err := os.Create(fName + "_equivalences.nt.gz")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	equivalences, _ := gzip.NewWriterLevel(f, gzip.BestCompression)
	defer equivalences.Close()

	// parse file
	var isPrefix, skip bool
	var line, token []byte
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

		// process subject
		bytesProcessed, token = firstWord(line)

		if len(token) == 0 || token[0] == '#' { // line is a comment
			continue
		}

		// process predicate
		_, token = firstWord(line[bytesProcessed:])

		switch strings.ToLower(string(token)) {
		case "http://www.w3.org/2002/07/owl#sameas",
			"http://www.w3.org/2002/07/owl#equivalentproperty",
			"http://www.w3.org/2002/07/owl#equivalentclass":
			_, err = equivalences.Write(line)
			io.WriteString(equivalences, "\n")
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	if err != nil && err != io.EOF {
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
