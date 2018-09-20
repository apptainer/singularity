// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package types

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"unicode"

	"github.com/sylabs/singularity/src/pkg/sylog"
)

// scanDefinitionFile is the SplitFunc for the scanner that will parse the deffile. It will split into tokens
// that designated by a line starting with %
//
// Scanner behavior:
//     1. The *first* time `s.Text()` is non-nil (which can be after infinitely many calls to
//        `s.Scan()`), that text is *guaranteed* to be the header, unless the header doesnt exist.
//		  In that case it returns the first section it finds.
//     2. The next `n` times that `s.Text()` is non-nil (again, each could take many calls to
//        `s.Scan()`), that text is guaranteed to be one specific section of the definition file.
//     3. Once the input buffer is completely scanned, `s.Text()` will either be nil or non-nil
//        (in which case `s.Text()` contains the last section found of the input buffer) *and*
//        `s.Err()` will be non-nil with an `bufio.ErrFinalToken` returned. This is where scanning can completely halt.
//
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

func insertSection(b []byte, sections map[string]string) {
	for i := 0; i < len(b); i++ {
		if b[i] == '\n' {
			sections[string(b[:i])] = strings.TrimRightFunc(string(b[i+1:]), unicode.IsSpace)
			break
		}
	}
}

func doSections(s *bufio.Scanner, d *Definition) (err error) {

	sections := make(map[string]string)

	token := strings.TrimSpace(s.Text())
	//check if first thing parsed is a header or just a section
	if strings.ToLower(token[0:9]) == "bootstrap" {
		if err = doHeader(token, d); err != nil {
			sylog.Warningf("failed to parse DefFile header: %v\n", err)
			return
		}
	} else {
		//this is a section
		insertSection([]byte(token), sections)
	}

	//parse remaining sections while scanner can advance
	for s.Scan() {
		if err = s.Err(); err != nil {
			return
		}

		b := s.Bytes()
		insertSection(b, sections)
	}

	if err = s.Err(); err != nil {
		return
	}

	// Files are parsed as a map[string]string
	filesSections := strings.TrimSpace(sections["files"])
	subs := strings.Split(filesSections, "\n")
	var files []FileTransport

	for _, line := range subs {

		if line = strings.TrimSpace(line); line == "" || strings.Index(line, "#") == 0 {
			continue
		}
		var src, dst string
		lineSubs := strings.SplitN(line, " ", 2)
		if len(lineSubs) < 2 {
			src = strings.TrimSpace(lineSubs[0])
			dst = ""
		} else {
			src = strings.TrimSpace(lineSubs[0])
			dst = strings.TrimSpace(lineSubs[1])
		}

		files = append(files, FileTransport{src, dst})
	}

	// labels are parsed as a map[string]string
	labelsSections := strings.TrimSpace(sections["labels"])
	subs = strings.Split(labelsSections, "\n")
	labels := make(map[string]string)

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

		labels[key] = val
	}

	d.ImageData = ImageData{
		ImageScripts: ImageScripts{
			Help:        sections["help"],
			Environment: sections["environment"],
			Runscript:   sections["runscript"],
			Test:        sections["test"],
			Startscript: sections["startscript"],
		},
		Labels: labels,
	}
	d.BuildData.Files = files
	d.BuildData.Scripts = Scripts{
		Pre:   sections["pre"],
		Setup: sections["setup"],
		Post:  sections["post"],
		Test:  sections["test"],
	}

	// make sure information was valid by checking if definition is not equal to an empty one
	emptyDef := new(Definition)
	//labels is always initialized
	emptyDef.Labels = make(map[string]string)
	if reflect.DeepEqual(d, emptyDef) {
		return fmt.Errorf("parsed definition did not have any valid information")
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

		//remove any comments on header lines
		trimLine := strings.Split(line, "#")[0]

		linetoks := strings.SplitN(trimLine, ":", 2)
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

	if err = doSections(s, &d); err != nil {
		return d, fmt.Errorf("failed to parse DefFile sections: %v", err)
	}

	return
}

func canGetHeader(r io.Reader) (ok bool, err error) {
	var d Definition

	s := bufio.NewScanner(r)
	s.Split(scanDefinitionFile)

	for s.Scan() && s.Text() == "" && s.Err() == nil {
	}

	if s.Err() != nil {
		log.Println(s.Err())
		return false, s.Err()
	} else if s.Text() == "" {
		return false, errors.New("Empty definition file")
	}

	if err = doHeader(s.Text(), &d); err != nil {
		sylog.Warningf("failed to parse DefFile header: %v\n", err)
		return
	}

	return true, nil
}

func writeSectionIfExists(w io.Writer, ident string, s string) {
	if len(s) > 0 {
		w.Write([]byte("%"))
		w.Write([]byte(ident))
		w.Write([]byte("\n"))
		w.Write([]byte(s))
		w.Write([]byte("\n\n"))
	}
}

func writeFilesIfExists(w io.Writer, f []FileTransport) {

	if len(f) > 0 {

		w.Write([]byte("%"))
		w.Write([]byte("files"))
		w.Write([]byte("\n"))

		for _, ft := range f {
			w.Write([]byte("\t"))
			w.Write([]byte(ft.Src))
			w.Write([]byte("\t"))
			w.Write([]byte(ft.Dst))
			w.Write([]byte("\n"))
		}
		w.Write([]byte("\n"))
	}
}

func writeLabelsIfExists(w io.Writer, l map[string]string) {

	if len(l) > 0 {

		w.Write([]byte("%"))
		w.Write([]byte("labels"))
		w.Write([]byte("\n"))

		for k, v := range l {
			w.Write([]byte("\t"))
			w.Write([]byte(k))
			w.Write([]byte(" "))
			w.Write([]byte(v))
			w.Write([]byte("\n"))
		}
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
	w.Write([]byte("\n"))

	writeLabelsIfExists(w, d.ImageData.Labels)
	writeFilesIfExists(w, d.BuildData.Files)

	writeSectionIfExists(w, "help", d.ImageData.Help)
	writeSectionIfExists(w, "environment", d.ImageData.Environment)
	writeSectionIfExists(w, "runscript", d.ImageData.Runscript)
	writeSectionIfExists(w, "test", d.ImageData.Test)
	writeSectionIfExists(w, "startscript", d.ImageData.Startscript)
	writeSectionIfExists(w, "pre", d.BuildData.Pre)
	writeSectionIfExists(w, "setup", d.BuildData.Setup)
	writeSectionIfExists(w, "post", d.BuildData.Post)
}
