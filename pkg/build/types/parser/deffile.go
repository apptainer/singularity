// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package parser

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/sylabs/singularity/pkg/build/types"
)

var (
	errInvalidSection  = errors.New("invalid section(s) specified")
	errEmptyDefinition = errors.New("Empty definition file")
)

// InvalidSectionError records an error and the sections that caused it.
type InvalidSectionError struct {
	Sections []string
	Err      error
}

func (e *InvalidSectionError) Error() string {
	return e.Err.Error() + ": " + strings.Join(e.Sections, ", ")
}

// IsInvalidSectionError returns a boolean indicating whether the error
// is reporting if a section of the definition is not a standard section
func IsInvalidSectionError(err error) bool {
	switch err.(type) {
	case *InvalidSectionError:
		return true
	}

	return false
}

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

			// We no longer check if the word is a valid section identifier here, since we want to move to
			// a more modular approach where we can parse arbitrary sections
			if inSection {
				// Here we found the end of the section
				return advance, retbuf.Bytes(), nil
			} else if advance == 0 {
				// When advance == 0 and we found a section identifier, that means we have already
				// parsed the header out and left the % as the first character in the data. This means
				// we can now parse into sections.
				retbuf.Write(line)
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

func getSectionName(line string) string {
	// trim % prefix on section name
	line = strings.TrimLeft(line, "%")
	lineSplit := strings.SplitN(strings.ToLower(line), " ", 2)

	return lineSplit[0]
}

// parseTokenSection splits the token into maximum 2 strings separated by a newline,
// and then inserts the section into the sections map
//
func parseTokenSection(tok string, sections map[string]string) error {
	split := strings.SplitN(tok, "\n", 2)
	if len(split) != 2 {
		return fmt.Errorf("Section %v: Could not be split into section name and body", split[0])
	}

	key := getSectionName(split[0])
	if appSections[key] {
		sectionSplit := strings.SplitN(strings.TrimLeft(split[0], "%"), " ", 3)
		if len(sectionSplit) < 2 {
			return fmt.Errorf("App Section %v: Could not be split into section name and app name", sectionSplit[0])
		}

		key = strings.Join(sectionSplit[0:2], " ")
	}

	sections[key] += split[1]

	return nil
}

func doSections(s *bufio.Scanner, d *types.Definition) error {
	sectionsMap := make(map[string]string)

	tok := strings.TrimSpace(s.Text())

	// skip initial token parsing if it is empty after trimming whitespace
	if tok != "" {
		//check if first thing parsed is a header/comment or just a section
		if tok[0] != '%' {
			if err := doHeader(tok, d); err != nil {
				return fmt.Errorf("failed to parse DefFile header: %v", err)
			}
		} else {
			//this is a section
			if err := parseTokenSection(tok, sectionsMap); err != nil {
				return err
			}
		}
	}

	//parse remaining sections while scanner can advance
	for s.Scan() {
		if err := s.Err(); err != nil {
			return err
		}

		tok := s.Text()

		// Parse each token -> section
		if err := parseTokenSection(tok, sectionsMap); err != nil {
			return err
		}
	}

	if err := s.Err(); err != nil {
		return err
	}

	return populateDefinition(sectionsMap, d)
}

func populateDefinition(sections map[string]string, d *types.Definition) (err error) {
	// Files are parsed as a map[string]string
	filesSections := strings.TrimSpace(sections["files"])
	subs := strings.Split(filesSections, "\n")
	var files []types.FileTransport

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

		files = append(files, types.FileTransport{Src: src, Dst: dst})
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

	d.ImageData = types.ImageData{
		ImageScripts: types.ImageScripts{
			Help:        sections["help"],
			Environment: sections["environment"],
			Runscript:   sections["runscript"],
			Test:        sections["test"],
			Startscript: sections["startscript"],
		},
		Labels: labels,
	}
	d.BuildData.Files = files
	d.BuildData.Scripts = types.Scripts{
		Pre:   sections["pre"],
		Setup: sections["setup"],
		Post:  sections["post"],
		Test:  sections["test"],
	}

	// remove standard sections from map
	for s := range validSections {
		delete(sections, s)
	}

	// add remaining sections to CustomData and throw error for invalid section(s)
	if len(sections) != 0 {
		d.CustomData = sections
		var keys []string
		for k := range sections {
			sectionName := strings.Split(k, " ")
			if !appSections[sectionName[0]] {
				keys = append(keys, k)
			}
		}
		if len(keys) > 0 {
			return &InvalidSectionError{keys, errInvalidSection}
		}
	}

	// make sure information was valid by checking if definition is not equal to an empty one
	emptyDef := new(types.Definition)
	// labels is always initialized
	emptyDef.Labels = make(map[string]string)
	if reflect.DeepEqual(d, emptyDef) {
		return fmt.Errorf("parsed definition did not have any valid information")
	}

	return err
}

func doHeader(h string, d *types.Definition) (err error) {
	h = strings.TrimSpace(h)
	toks := strings.Split(h, "\n")
	d.Header = make(map[string]string)

	for _, line := range toks {
		// skip empty or comment lines
		if line = strings.TrimSpace(line); line == "" || strings.Index(line, "#") == 0 {
			continue
		}

		// trim any comments on header lines
		trimLine := strings.Split(line, "#")[0]

		linetoks := strings.SplitN(trimLine, ":", 2)
		if len(linetoks) == 1 {
			return fmt.Errorf("header key %s had no val", linetoks[0])
		}

		key, val := strings.ToLower(strings.TrimSpace(linetoks[0])), strings.TrimSpace(linetoks[1])
		if _, ok := validHeaders[key]; !ok {
			return fmt.Errorf("invalid header keyword found: %s", key)
		}
		d.Header[key] = val
	}

	return
}

// ParseDefinitionFile receives a reader from a definition file
// and parse it into a Definition struct or return error if
// the definition file has a bad section.
func ParseDefinitionFile(r io.Reader) (d types.Definition, err error) {
	d.Raw, err = ioutil.ReadAll(r)
	if err != nil {
		return d, fmt.Errorf("While attempting to read in definition: %v", err)
	}

	s := bufio.NewScanner(bytes.NewReader(d.Raw))
	s.Split(scanDefinitionFile)

	// advance scanner until it returns a useful token or errors
	for s.Scan() && s.Text() == "" && s.Err() == nil {
	}

	if s.Err() != nil {
		log.Println(s.Err())
		return d, s.Err()
	} else if s.Text() == "" {
		return d, errEmptyDefinition
	}

	if err = doSections(s, &d); err != nil {
		return d, err
	}

	return
}

// IsValidDefinition returns whether or not the given file is a valid definition
func IsValidDefinition(source string) (valid bool, err error) {
	defFile, err := os.Open(source)
	if err != nil {
		return false, err
	}

	if s, err := defFile.Stat(); err != nil {
		return false, fmt.Errorf("unable to stat file: %v", err)
	} else if s.IsDir() {
		return false, nil
	}

	defer defFile.Close()

	_, err = ParseDefinitionFile(defFile)
	if err != nil {
		return false, err
	}

	return true, nil
}

// validSections just contains a list of all the valid sections a definition file
// could contain. If any others are found, an error will generate
var validSections = map[string]bool{
	"help":        true,
	"setup":       true,
	"files":       true,
	"labels":      true,
	"environment": true,
	"pre":         true,
	"post":        true,
	"runscript":   true,
	"test":        true,
	"startscript": true,
}

var appSections = map[string]bool{
	"appinstall": true,
	"applabels":  true,
	"appfiles":   true,
	"appenv":     true,
	"apptest":    true,
	"apphelp":    true,
	"apprun":     true,
}

// validHeaders just contains a list of all the valid headers a definition file
// could contain. If any others are found, an error will generate
var validHeaders = map[string]bool{
	"bootstrap":  true,
	"from":       true,
	"includecmd": true,
	"mirrorurl":  true,
	"updateurl":  true,
	"osversion":  true,
	"include":    true,
	"library":    true,
	"registry":   true,
	"namespace":  true,
}
