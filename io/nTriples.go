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
	Line      []byte // Holds the entire line including terminating dot (but no newline)
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
// Arguments:
//   numTokens int : by adding the optional argument you can decide how many tokens
//                   should be parsed. With 0, no tokens are parsed. With 1, 2 or 3
//                   all tokens up to including Subject, Predicate and Object are
//                   parsed. Non-parsed tokens are nil slices.
//
// Example:
//    triple, err := tripleParser.NextTriple(2)  // consume line and parse subject and predicate.
//
// TODO: The parsing is a subset of the actual N-Triple syntax.
func (tp *TripleParser) NextTriple(argNumTokens ...int) (*Triple, error) {

	// parse the optional arguments
	var numTokens int
	if len(argNumTokens) > 0 {
		numTokens = argNumTokens[0]
	} else {
		numTokens = 3 // per default, all tokens are parsed
	}

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

	// early termination if no tokens are to be parsed.
	if numTokens == 0 {
		return &Triple{nil, nil, nil, origLine}, nil
	}

	// process subject
	subjectBytes, subjectToken := retrieveToken(origLine)
	line := origLine[subjectBytes:]

	// skip if line is empty of a comment
	if len(subjectToken) == 0 || subjectToken[0] == '#' {
		return tp.NextTriple()
	}

	// early termination if only one token should be parsed
	if numTokens == 1 {
		return &Triple{subjectToken, nil, nil, origLine}, nil
	}

	// process predicate
	predicateBytes, predicateToken := retrieveToken(line)
	line = line[predicateBytes:]

	// early termination if only two tokens should be parsed
	if numTokens == 2 {
		return &Triple{subjectToken, predicateToken, nil, origLine}, nil
	}

	// process object
	// TODO: Check if strings with spaces work, like:  "New York"@gb
	_, objectToken := retrieveToken(line)

	// return the triple with all information
	return &Triple{subjectToken, predicateToken, objectToken, origLine}, nil
}

// Close the handlers for the scanner and underlying file.
func (tp *TripleParser) Close() error {
	defer tp.reader.Close()
	return nil
}

// Output the values of a triple to stdout.
func (t *Triple) Output() {
	fmt.Println("( " + string(t.Subject) + " , " + string(t.Predicate) + " , " + string(t.Object) + " )")
}

// InterpreteLangLiteral will interprete a literal with language tag and return the
// text and language code.
// It is assumed that the token has no leading nor trailing spaces.
func InterpreteLangLiteral(token []byte) (text []byte, lang []byte) {
	var sigil rune // current rune that is being checked
	var width int  // width of sigil
	var start int  // remember the position where the literal text started

	// Similar approach to reading tokens, but with less nesting options.
	// Will detect the limits of the quoted string, and then use the rest and lang tag.
	var quoting bool
	var escaping bool
	for pos := 0; pos < len(token); pos += width {
		sigil, width = identifyRune(token[pos:])

		// if escaping, just jump to next rune
		if escaping == true {
			escaping = false
			continue
		}

		// a backslash will start escaping, no parsing will ever be done with this
		if sigil == '\\' {
			escaping = true
			continue
		}

		// non-nesting state: can enter a nesting state and spaces will terminate token
		if quoting == false && sigil == '"' { // should always enter this at first position
			quoting = true
			start = pos + width
		} else if quoting == true && sigil == '"' { // exit the quoted string, text is found
			quoting = false
			text = token[start:pos]
		} else if quoting == false && sigil == '@' { // tells us that lang tags are being used
			lang = token[pos+width:]
			break
		}
	}

	return // naked return
}

// InterpreteIriRef will interprete an IRIREF and return the string that is enclosed by the
// tag delimiters.
// It is assumed that the token has no leading nor trailing spaces.
func InterpreteIriRef(token []byte) (text []byte) {
	var firstSigil, lastSigil rune

	// Because I assume no leading and trailing spaces, I can use this shortcut.
	firstSigil, _ = identifyRune(token[0:])
	lastSigil, _ = identifyRune(token[len(token)-1:])
	if firstSigil == '<' && lastSigil == '>' {
		text = token[1 : len(token)-1]
	} else {
		text = token
	}

	return // naked return
}

// Retrieves a token from a N-Triple entry.
// TODO: Test multiple escaping cases
// TODO: This could be improved by allowing an argument to specify the max number of tokens to parse.
//       If only the subject is needed than parsing the predicate and object becomes unnecessary.
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

		// if escaping, just jump to next rune
		if escaping == true {
			escaping = false
			continue
		}

		// a backslash will start escaping, no parsing will ever be done with this
		if sigil == '\\' {
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
			if (nesting == '<' && sigil == '>') || (nesting == '"' && sigil == '"') {
				nesting = 0
			}
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
