/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package build

import (
	"bufio"
	"io"
	"log"
	"regexp"
	"strings"
	"unicode"
)

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
