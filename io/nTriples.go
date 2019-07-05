package io

import (
	"bufio"
	"fmt"
	"io"
	"unicode"
	"unicode/utf8"
)

// Triple represents an RDF entry in the N-Triple file.
// TODO: Check if having []byte does not make problems with the buffer they are pointing to. What
//       happens to the triple if the buffer flushes?
type Triple struct { // TODO: Maybe I can use string instead of byte[]
	Subject   []byte
	Predicate []byte
	Object    []byte
	Line      []byte // Holds the entire line including terminating dot and newline
}

// TripleParser reads an internal file and produces triples from it.
type TripleParser struct {
	reader  io.ReadCloser
	scanner *bufio.Reader
}

// NewTripleParser opens a file and returns the relevant triple parser that will produce triples.
func NewTripleParser(filePath string) (*TripleParser, error) {

	// IO setup
	reader, err := UniversalReader(filePath)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewReaderSize(reader, 4*1024*1024) // 4MB line Buffer

	return &TripleParser{reader, scanner}, nil
}

// NextTriple returns the next triple that is read from the internal file.
//
// TODO: The parsing is a subset of the actual N-Triple syntax.
// TODO: Integrate with the loading bar.
func (tp *TripleParser) NextTriple() (*Triple, error) {

	// read one line from the file
	origLine, isPrefix, err := tp.scanner.ReadLine()
	if err != nil && err != io.EOF { // misc error
		return nil, err
	} else if err == io.EOF { // file has ended
		return nil, nil
	}

	// whenever a line is to be skipped, it needs to deliver the next line instead

	// skip because line too big
	if isPrefix {
		fmt.Printf("Line Buffer too small!!! Line prefix: %v\n", string(origLine[:200]))
		return tp.NextTriple()
	}

	// process subject
	subjectBytes, subjectToken := retrieveToken(origLine)
	line := origLine[subjectBytes:]

	// skip if line is empty of a comment
	if len(subjectToken) == 0 || subjectToken[0] == '#' {
		return tp.NextTriple()
	}

	// process predicate
	predicateBytes, predicateToken := retrieveToken(line)
	line = line[predicateBytes:]

	// process object
	// TODO: Check if strings with spaces work, like:  "New York"@gb
	_, objectToken := retrieveToken(line)

	// return the triple with all information
	return &Triple{subjectToken, predicateToken, objectToken, origLine}, nil
}

// Close closes the handlers for the scanner and underlying file.
func (tp *TripleParser) Close() error {
	defer tp.reader.Close()
	return nil
}

// Retrieves a token from a N-Triple entry.
func retrieveToken(data []byte) (advance int, token []byte) {
	var sigil rune        // current rune that is being checked
	var width int         // width of sigil
	var start, length int // delimiters for the token

	// Skip leading spaces.
	for start < len(data) {
		sigil, width = identifyRune(data[start:])
		if unicode.IsSpace(sigil) { // Could also use the isSpaceOrBracket method (without the brackets)
			start += width
		} else {
			break
		}
	}

	// Catch the entire token, while paying attention to IRIREF brackets and literal quotes.
	var nesting rune
	var escaping bool
	for ; start+length < len(data); length += width {
		sigil, width = identifyRune(data[start+length:])

		// need to check if currently escaping to allow \" inside a quoted literal
		if sigil == '\'' {
			escaping = true
			continue
		}

		// non-nesting state: can enter a nesting state and spaces will terminate token
		if nesting == 0 {
			if sigil == '<' || sigil == '"' {
				nesting = sigil
			} else if escaping == false && unicode.IsSpace(sigil) {
				break
			}
		} else { // nesting state: can exit nesting state
			if escaping == false && ((nesting == '<' && sigil == '>') || (nesting == '"' && sigil == '"')) {
				nesting = 0
			}
		}

		// escaping mode ends one iteration after it starts
		if escaping == true {
			escaping = false
		}
	}

	// if we're at EOL, and we have a final, non-empty, non-terminated word. Return it.
	if start+length >= len(data) {
		return len(data), data[start:]
	}

	// return the token
	return start + length, data[start : start+length]
}

// Identify rune and width.
// Uses a special procedure to check for 1-width ASCII runes.
// TODO: Check if this special procedure is really faster.
func identifyRune(data []byte) (sigil rune, width int) {
	if data[0] < utf8.RuneSelf {
		return rune(data[0]), 1
	}
	return utf8.DecodeRune(data)
}
