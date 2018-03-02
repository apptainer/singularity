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
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"unicode"
)

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
}

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
				return 0, nil, fmt.Errorf("Invalid section identifier found: %s", string(word))
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

func doSections(r io.Reader) (sections map[string]string, err error) {
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
		log.Fatal(s.Err())
		return nil, s.Err()
	}

	fmt.Println("=======Sections=======")
	for k, v := range sections {
		fmt.Printf("Section[%s]:\n%s\n\n", k, v)
	}

	return
}

// validHeaders just contains a list of all the valid headers a definition file
// could contain. If any others are found, an error will generate
var validHeaders = map[string]bool{
	"bootstrap":  true,
	"from":       true,
	"registry":   true,
	"namespace":  true,
	"includecmd": true,
	"mirrorurl":  true,
	"osversion":  true,
	"include":    true,
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
	} else {
		return advance, nil, nil
	}
}

func doHeader(r io.Reader) (header map[string]string, err error) {
	s := bufio.NewScanner(r)
	s.Split(scanHeader)

	header = make(map[string]string)

	fmt.Println("========Header========")
	for s.Scan() {
		tok := strings.SplitN(s.Text(), ":", 2)
		header[tok[0]] = tok[1]
		fmt.Printf("header[%s] = %s\n", tok[0], tok[1])
	}

	if s.Err() != nil {
		log.Fatal(s.Err())
		return nil, s.Err()
	}

	return
}

func ParseDefinitionFile(f *os.File) (Definition, error) {
	header, err := doHeader(f)

	f.Seek(0, 0)
	sections, err := doSections(f)

	def := Definition{
		Header: header,
		ImageData: imageData{
			imageScripts: imageScripts{
				Help:        sections["help"],
				Environment: sections["environment"],
				Runscript:   sections["runscript"],
				Test:        sections["test"],
			},
		},
		BuildData: buildData{
			buildScripts: buildScripts{
				Pre:   sections["pre"],
				Setup: sections["setup"],
				Post:  sections["post"],
			},
		},
	}

	return def, err
}

func writeSectionIfExists(w io.Writer, ident string, s string) {
	if len(s) > 0 {
		fmt.Printf("section[%s]:\n%s\n\n", ident, s)
		w.Write([]byte("%"))
		w.Write([]byte(ident))
		w.Write([]byte("\n"))
		w.Write([]byte(s))
		w.Write([]byte("\n"))
	}
}

func (d *Definition) WriteDefinitionFile(w io.Writer) {
	fmt.Println("=======BEGIN DEFINITION FILE WRITE=======")
	for k, v := range d.Header {
		fmt.Printf("header[%s] = %s\n", k, v)
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

	fmt.Println("========END DEFINITION FILE WRITE========")
}

/* ==================================================================*/

var (
	tokenComment = regexp.MustCompile(`#.*$`)
	headerKeys   = []string{
		"Bootstrap", "From", "Registry",
		"Namespace", "IncludeCmd", "MirrorURL",
		"OSVersion", "Include"}
	sectionsKeys = []string{
		"%help", "%setup", "%files",
		"%labels", "%environment", "%post",
		"%runscript", "%test"}
	sectionsParsers = map[string]parseSection{
		"%help":        sectionHelp,
		"%setup":       sectionSetup,
		"%files":       sectionFiles,
		"%labels":      sectionLabels,
		"%environment": sectionEnv,
		"%post":        sectionPost,
		"%runscript":   sectionRunscript,
		"%test":        sectionTest,
	}
)

type parseSection func(*Deffile, []string, int, string)

// DefaultEscapeToken is the default escape token
const DefaultEscapeToken = "\\"

// Deffile holds the entirety of the definition file, a header and the
// sections that were defined
type Deffile struct {
	// Header contains the information for what source to bootstrap from
	Header map[string]string
	Sections
}

// Sections contains each of the %sections defined in the def file
type Sections struct {
	help      string
	setup     string
	files     map[string]string
	labels    map[string]string
	env       string
	post      string
	runscript string
	test      string
}

// Header contains the information for what source to bootstrap from
type header struct {
	Lines []string
}

// ParseDefFile reads the contents of a deffile and returns it as a parsed Deffile
func ParseDefFile(r io.Reader) (Deffile, error) {
	lines, err := cleanUpFile(r)
	if err != nil {
		return Deffile{}, err
	}
	Df, err := parseLines(lines)
	if err != nil {
		return Deffile{}, err
	}
	return Df, err
}

// cleanUpFile removes comments, escape characters
// and white spaces from deffile and converts text to []string
func cleanUpFile(r io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		// Trim Blank lines
		if line == "" {
			//jump empty lines
			continue
		}
		// parse the escape character for long commands
		if lineHasEscapeChar(line) {
			line = parseEscape(scanner, line)
		}
		// Trim comments (if present)
		if lineHasComment(line) {
			line = trimComments(line)
			if line != "" {
				lines = append(lines, line)
			}
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return lines, nil
}

func parseLines(lines []string) (Deffile, error) {
	Df := Deffile{
		Header: make(map[string]string),
		Sections: Sections{
			files:  make(map[string]string),
			labels: make(map[string]string),
		}}

	for i, line := range lines {
		if key, b := isHeader(line); b {
			value := strings.TrimPrefix(line, key+":")
			Df.Header[key] = trimWhitespace(value)
		} else if section, b := isSection(line); b {
			prsr := sectionsParsers[section]
			prsr(&Df, lines, i, line)
		}
	}
	return Df, nil
}

func isHeader(line string) (string, bool) {
	for _, k := range headerKeys {
		if strings.Contains(line, k) {
			return k, true
		}
	}
	return "", false
}

func isSection(line string) (string, bool) {
	for _, key := range sectionsKeys {
		if strings.Contains(line, key) {
			return key, true
		}
	}
	return "", false
}

// func parseSection(lines []string, i int, line string) string {
// 	var commands string
// 	for _, line := range lines[i+1:] {
// 		if _, b := isSection(line); b {
// 			break
// 		}
// 		commands = commands + "\n" + line
// 	}
// 	return commands
// }

func sectionSetup(def *Deffile, lines []string, i int, line string) {
	var setup string
	for _, line := range lines[i+1:] {
		if _, b := isSection(line); b {
			break
		}
		setup = setup + "\n" + line
	}
	def.Sections.setup = setup
}

func sectionHelp(def *Deffile, lines []string, i int, line string) {
	var help string
	for _, line := range lines[i+1:] {
		if _, b := isSection(line); b {
			break
		}
		help = help + "\n" + line
	}
	def.Sections.help = help
}

func sectionPost(def *Deffile, lines []string, i int, line string) {
	var post string
	for _, line := range lines[i+1:] {
		if _, b := isSection(line); b {
			break
		}
		post = post + "\n" + line
	}
	def.Sections.post = post
}

func sectionTest(def *Deffile, lines []string, i int, line string) {
	var test string
	for _, line := range lines[i+1:] {
		if _, b := isSection(line); b {
			break
		}
		test = test + "\n" + line
	}
	def.Sections.test = test
}

func sectionEnv(def *Deffile, lines []string, i int, line string) {
	var env string
	for _, line := range lines[i+1:] {
		if _, b := isSection(line); b {
			break
		}
		env = env + "\n" + line
	}
	def.Sections.env = env
}

func sectionLabels(def *Deffile, lines []string, i int, line string) {
	for _, line := range lines[i+1:] {
		if _, b := isSection(line); b {
			break
		}
		ref := strings.Split(line, " ")
		def.Sections.labels[ref[0]] = ref[1]
	}
}

func sectionFiles(def *Deffile, lines []string, i int, line string) {
	for _, line := range lines[i+1:] {
		if _, b := isSection(line); b {
			break
		}
		ref := strings.Split(line, " ")
		if len(ref) >= 2 {
			def.Sections.files[ref[0]] = ref[1]
			continue
		}
		def.Sections.files[ref[0]] = ""
	}
}

func sectionRunscript(def *Deffile, lines []string, i int, line string) {
	var runScript string
	for _, line := range lines[i+1:] {
		if _, b := isSection(line); b {
			break
		}
		runScript = runScript + "\n" + line
	}
	def.Sections.runscript = runScript
}

// parseEscape parses the escape character for long commands
func parseEscape(scanner *bufio.Scanner, line string) string {
	line = strings.TrimSuffix(line, DefaultEscapeToken)
	for scanner.Scan() {
		if lineHasEscapeChar(scanner.Text()) {
			newLine := parseEscape(scanner, scanner.Text())
			line = line + strings.TrimSpace(newLine)
			continue
		}
		line = line + strings.TrimSpace(scanner.Text())
		break
	}
	return line
}

func trimWhitespace(src string) string {
	return strings.TrimLeftFunc(src, unicode.IsSpace)
}

func lineHasComment(line string) bool {
	return tokenComment.MatchString(trimWhitespace(line))
}

func lineHasEscapeChar(line string) bool {
	return strings.HasSuffix(line, DefaultEscapeToken)
}

func trimComments(src string) string {
	return tokenComment.ReplaceAllString(src, "")
}
