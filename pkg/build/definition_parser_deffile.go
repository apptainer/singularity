/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package build

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"
	"unicode"

	"github.com/golang/glog"
)

// scanSections is the SplitFunc for the scanner that will parse the deffile. It will split into tokens
// that designated by a line starting with %
// If there are any Golang devs reading this, please improve your documentation for this. It's awful.
func scanSections(data []byte, atEOF bool) (advance int, token []byte, err error) {
	var inSection bool = false
	var retbuf bytes.Buffer
	advance = 0

	l := len(data)

	for advance < l {
		// We are essentially a pretty wrapper to bufio.ScanLines.
		a, line, err := bufio.ScanLines(data, atEOF)
		if err != nil && err != bufio.ErrFinalToken {
			return 0, nil, err
		} else if line == nil { // If ScanLines returns a nil line, it needs more data. Send req for more data
			return 0, nil, nil // Returning 0, nil, nil requests Scanner.Scan() method find more data or EOF
		}

		_, word, err := bufio.ScanWords(line, true) // Tokenize the line into words
		if err != nil && err != bufio.ErrFinalToken {
			return 0, nil, err
		}

		// Check if the first word starts with % sign
		if word != nil && word[0] == '%' {
			// If the word starts with %, it's a section identifier
			_, ok := validSections[string(word[1:])] // Validate that the section identifier is valid

			if !ok {
				// Invalid Section Identifier
				return 0, nil, errors.New(fmt.Sprintf("Invalid section identifier found: %s", string(word)))
			} else {
				// Valid Section Identifier
				if inSection {
					// Here we found the end of the section
					return advance, retbuf.Bytes(), nil
				} else {
					// Here is the start of a section, write the section into the return buffer and
					// flag that we've found the start of a section
					retbuf.Write(word[1:])
					retbuf.WriteString("\n")
					inSection = true
				}
			}
		} else {
			// This line is not a section identifier
			if inSection {
				// If we're inside of a section,
				retbuf.Write(line)
				retbuf.WriteString("\n")
			}
		}

		// Shift the advance retval and the data slice to the next line
		advance += a
		data = data[a:]
		if a == 0 {
			break
		}
	}

	if !atEOF {
		return 0, nil, nil
	} else {
		return advance, retbuf.Bytes(), nil
	}
}

func doSections(definition *Definition, r io.Reader, done chan error) (sections map[string]string, err error) {
	s := bufio.NewScanner(r)
	s.Split(scanSections)

	sections = make(map[string]string)

	for s.Scan() {
		b := s.Bytes()

		for i := 0; i < len(b); i++ {
			if b[i] == '\n' {
				sections[string(b[:i])] = strings.TrimRightFunc(string(b[i+1:]), unicode.IsSpace)
				break
			}
		}
	}

	if s.Err() != nil {
		log.Println(s.Err())
		done <- s.Err()
		return nil, s.Err()
	}

	definition.ImageData = imageData{
		imageScripts: imageScripts{
			Help:        sections["help"],
			Environment: sections["environment"],
			Runscript:   sections["runscript"],
			Test:        sections["test"],
		},
	}
	definition.BuildData.buildScripts = buildScripts{
		Pre:   sections["pre"],
		Setup: sections["setup"],
		Post:  sections["post"],
	}

	done <- err
	return
}

// scanHeader is a SplitFunc to extract header tokens, token format: "key:val"
func scanHeader(data []byte, atEOF bool) (advance int, token []byte, err error) {
	var retbuf bytes.Buffer

	advance = 0
	l := len(data)

	for advance < l {
		a, line, err := bufio.ScanLines(data, atEOF)
		if err != nil && err != bufio.ErrFinalToken {
			return 0, nil, err
		} else if line == nil { // If ScanLines returns a nil line, it needs more data. Send req for more data
			return 0, nil, nil // Returning 0, nil, nil requests Scanner.Scan() method find more data or EOF
		}

		advance += a
		words := strings.SplitN(string(line), ":", 2)

		hkey := strings.ToLower(strings.TrimRightFunc(words[0], unicode.IsSpace))
		if _, ok := validHeaders[hkey]; ok {
			retbuf.WriteString(hkey)
			retbuf.WriteString(":")
			retbuf.WriteString(strings.TrimSpace(words[1]))

			return advance, retbuf.Bytes(), nil
		}

		data = data[a:]
		if a == 0 {
			break
		}
	}

	if !atEOF {
		return 0, nil, nil
	}
	return advance, nil, nil

}

func doHeader(definition *Definition, r io.Reader, done chan error) (header map[string]string, err error) {
	s := bufio.NewScanner(r)
	s.Split(scanHeader)

	header = make(map[string]string)

	for s.Scan() {
		tok := strings.SplitN(s.Text(), ":", 2)
		header[tok[0]] = tok[1]
	}

	if s.Err() != nil {
		glog.Fatal(s.Err())
		done <- err
		return nil, s.Err()
	}

	definition.Header = header

	done <- err
	return
}

func ParseDefinitionFile(r io.Reader) (Definition, error) {
	c1 := make(chan error)
	c2 := make(chan error)

	definition := Definition{}

	reader, err := ioutil.ReadAll(r)
	if err != nil {
		return Definition{}, err
	}

	// clone the bytes for parse rutines
	hr := bytes.NewReader(reader)
	sr := bytes.NewReader(reader)

	// Parse rutines
	go doHeader(&definition, hr, c1)
	go doSections(&definition, sr, c2)

	// We’ll use select to await both of these values simultaneously
	// if one of the parser rutines returns error, ParseDefinitionFile
	// will break and return an empty Definition with the error
	for i := 0; i < 2; i++ {
		select {
		case headerErr := <-c1:
			if headerErr != nil {
				return Definition{}, headerErr
			}
		case sectionsErr := <-c2:
			if sectionsErr != nil {
				return Definition{}, sectionsErr
			}
		}
	}

	return definition, nil
}

func writeSectionIfExists(w io.Writer, ident string, s string) {
	if len(s) > 0 {
		w.Write([]byte("%"))
		w.Write([]byte(ident))
		w.Write([]byte("\n"))
		w.Write([]byte(s))
		w.Write([]byte("\n"))
	}
}

// WriteDefinitionFile is a helper func to output a Definition struct
// into a definition file.
func (d *Definition) WriteDefinitionFile(w io.Writer) {

	for k, v := range d.Header {
		w.Write([]byte(k))
		w.Write([]byte(": "))
		w.Write([]byte(v))
		w.Write([]byte("\n"))
	}

	writeSectionIfExists(w, "help", d.ImageData.Help)
	writeSectionIfExists(w, "environment", d.ImageData.Environment)
	writeSectionIfExists(w, "runscript", d.ImageData.Runscript)
	writeSectionIfExists(w, "test", d.ImageData.Test)
	writeSectionIfExists(w, "pre", d.BuildData.Pre)
	writeSectionIfExists(w, "setup", d.BuildData.Setup)
	writeSectionIfExists(w, "post", d.BuildData.Post)
}
