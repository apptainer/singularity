// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"unicode"

	"github.com/singularityware/singularity/src/pkg/sylog"
)

// scanDefinitionFile is the SplitFunc for the scanner that will parse the deffile. It will split into tokens
// that designated by a line starting with %
// If there are any Golang devs reading this, please improve your documentation for this. It's awful.
func scanDefinitionFile(data []byte, atEOF bool) (advance int, token []byte, err error) {
	inSection := false
	var retbuf bytes.Buffer
	advance = 0

	l := len(data)

	for advance < l {
		// We are essentially a pretty wrapper to bufio.ScanLines.
		a, line, err := bufio.ScanLines(data[advance:], atEOF)
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
				return 0, nil, fmt.Errorf("invalid section identifier found: %s", string(word))
			}

			// Valid Section Identifier
			if inSection {
				// Here we found the end of the section
				return advance, retbuf.Bytes(), nil
			} else if advance == 0 {
				// When advance == 0 and we found a section identifier, that means we have already
				// parsed the header out and left the % as the first character in the data. This means
				// we can now parse into sections.
				retbuf.Write(word[1:])
				retbuf.WriteString("\n")
				inSection = true
			} else {
				// When advance != 0, that means we found the start of a section but there is
				// data before it. We return the data up to the first % and that is the header
				return advance, retbuf.Bytes(), nil
			}
		} else {
			// This line is not a section identifier
			retbuf.Write(line)
			retbuf.WriteString("\n")
		}

		// Shift the advance retval to the next line
		advance += a
		if a == 0 {
			break
		}
	}

	if !atEOF {
		return 0, nil, nil
	}

	return advance, retbuf.Bytes(), nil

}

func doSections(s *bufio.Scanner, d *Definition) (err error) {
	sections := make(map[string]string)

	for s.Scan() {
		if err = s.Err(); err != nil {
			return
		}

		b := s.Bytes()
		for i := 0; i < len(b); i++ {
			if b[i] == '\n' {
				sections[string(b[:i])] = strings.TrimRightFunc(string(b[i+1:]), unicode.IsSpace)
				break
			}
		}
	}

	if err = s.Err(); err != nil {
		return
	}

	// Files are parsed as a map[string]string
	filesSections := strings.TrimSpace(sections["files"])
	subs := strings.Split(filesSections, "\n")
	files := make(map[string]string)

	for _, line := range subs {
		if line = strings.TrimSpace(line); line == "" || strings.Index(line, "#") == 0 {
			continue
		}
		var key, val string
		lineSubs := strings.SplitN(line, " ", 2)
		if len(lineSubs) < 2 {
			key = strings.TrimSpace(lineSubs[0])
			val = ""
		} else {
			key = strings.TrimSpace(lineSubs[0])
			val = strings.TrimSpace(lineSubs[1])
		}

		files[key] = val
	}

	d.ImageData = ImageData{
		ImageScripts: ImageScripts{
			Help:        sections["help"],
			Environment: sections["environment"],
			Runscript:   sections["runscript"],
			Test:        sections["test"],
		},
	}
	d.BuildData.Files = files
	d.BuildData.Scripts = Scripts{
		Pre:   sections["pre"],
		Setup: sections["setup"],
		Post:  sections["post"],
	}

	return
}

func doHeader(h string, d *Definition) (err error) {
	h = strings.TrimSpace(h)
	toks := strings.Split(h, "\n")
	d.Header = make(map[string]string)

	for _, line := range toks {
		if line = strings.TrimSpace(line); line == "" || strings.Index(line, "#") == 0 {
			continue
		}

		linetoks := strings.SplitN(line, ":", 2)
		key, val := strings.ToLower(strings.TrimSpace(linetoks[0])), strings.TrimSpace(linetoks[1])
		if _, ok := validHeaders[key]; !ok {
			return fmt.Errorf("invalid header keyword found: %s", key)
		}
		d.Header[key] = val
	}

	return
}

// ParseDefinitionFile recieves a reader from a definition file
// and parse it into a Definition struct or return error if
// the definition file has a bad section.
func ParseDefinitionFile(r io.Reader) (d Definition, err error) {
	s := bufio.NewScanner(r)
	s.Split(scanDefinitionFile)

	for s.Scan() && s.Text() == "" && s.Err() == nil {
	}

	if s.Err() != nil {
		log.Println(s.Err())
		return d, s.Err()
	} else if s.Text() == "" {
		return d, errors.New("Empty definition file")
	}

	if err = doHeader(s.Text(), &d); err != nil {
		sylog.Warningf("failed to parse DefFile header: %v\n", err)
		return
	}
	if err = doSections(s, &d); err != nil {
		sylog.Warningf("failed to parse DefFile sections: %v\n", err)
	}

	return
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
