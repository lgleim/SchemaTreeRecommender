package main

import (
	"bufio"
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"schematree"
	"unicode/utf8"

	gzip "github.com/klauspost/pgzip"
)

type propMap map[string]*string

func (m propMap) get(iri []byte) (item *string) {
	item, ok := m[string(iri)]
	if !ok {
		tmp := string(iri)
		item = &tmp
		m[tmp] = item
	}
	return
}

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

	type set map[*string]bool
	eqMap := make(map[*string]*set)
	propMap := make(propMap)
	uniqueSets := make(map[*set]bool)

	f, err := os.Open(*fileName + ".sets.gob")
	defer f.Close()

	if err == nil {
		log.Println("Loading evaluation results from previous run!")
		r, err := gzip.NewReader(f)
		if err != nil {
			log.Fatalln("Failed to open archive!", err)
		}
		defer r.Close()
		decoder := gob.NewDecoder(r)

		err = decoder.Decode(&uniqueSets)
		if err != nil {
			log.Fatalln("Failed to decode stats!", err)
		}

		for set := range uniqueSets {
			for term := range *set {
				propMap[*term] = term
				eqMap[term] = set
			}
		}

	} else {
		// Set up file reader
		reader, err := schematree.UniversalReader(*fileName)
		if err != nil {
			log.Fatal(err)
		}
		defer reader.Close()

		scanner := bufio.NewReaderSize(reader, 4*1024*1024) // 4MB line Buffer

		// parse file
		var subOK, objOK bool
		var line, subject, object []byte
		var bytesProcessed int
		var subSS, objSS *set

		for line, _, err = scanner.ReadLine(); err == nil; line, _, err = scanner.ReadLine() {
			// process subject
			bytesProcessed, subject = firstWord(line)

			// process predicate
			line = line[bytesProcessed:]
			bytesProcessed, _ = firstWord(line)

			// process object
			line = line[bytesProcessed:]
			_, object = firstWord(line)

			sub := propMap.get(subject)
			obj := propMap.get(object)

			subSS, subOK = eqMap[sub]
			objSS, objOK = eqMap[obj]

			if subOK && objOK {
				// both equivalence sets exist
				if subSS != objSS {
					// merge previously distinct equivalence sets
					(*subSS)[obj] = true    // add new equivalence to subject set
					for x := range *objSS { // add transitive equivalences from old object set
						(*subSS)[x] = true
					}
					eqMap[obj] = subSS // update object reference to merged set
				}
			} else if subOK {
				(*subSS)[obj] = true // add new equivalence to subject set
				eqMap[obj] = subSS   // add object reference
			} else if objOK {
				(*objSS)[sub] = true // add new equivalence to object set
				eqMap[sub] = objSS   // add subject reference
			} else {
				// neither exists, create a new set
				s := &set{sub: true, obj: true}
				eqMap[sub] = s
				eqMap[obj] = s
			}
		}

		if err != nil && err != io.EOF {
			log.Fatalf("Scanner encountered error while trying to parse triples: %v\n", err)
		}

		// make set of unique equivalence sets
		for _, set := range eqMap {
			uniqueSets[set] = true
		}

		// gob serialize
		fmt.Println("Storing results: GOB Serialization")
		f, _ := os.Create(*fileName + ".sets.gob")
		w := gzip.NewWriter(f)
		defer w.Close()
		e := gob.NewEncoder(w)
		e.Encode(uniqueSets)
		f.Close()

		// json serialize
		fmt.Println("Storing results: JSON Serialization")
		f, _ = os.Create(*fileName + ".sets.json")
		defer f.Close()

		w = gzip.NewWriter(f)
		defer w.Close()

		for set := range uniqueSets {
			w.Write([]byte("["))
			l := len(*set)
			i := 0
			for uri := range *set {
				w.Write([]byte("\"" + *uri + "\""))
				if i++; i < l {
					w.Write([]byte(","))
				}
			}
			w.Write([]byte("]\n"))
		}
	}
	fmt.Println("Done.")

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
