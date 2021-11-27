// Copyright (c) 2019-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"regexp"
	"strings"
)

// BindOption represents a bind option with its associated
// value if any.
type BindOption struct {
	Value string `json:"value,omitempty"`
}

const (
	flagOption  = true
	valueOption = false
)

// bindOptions is a map of option strings valid in bind specifications.
// If true, the option is a flag. If false, the option takes a value.
var bindOptions = map[string]bool{
	"ro":        flagOption,
	"rw":        flagOption,
	"image-src": valueOption,
	"id":        valueOption,
}

// BindPath stores a parsed bind path specification. Source and Destination
// paths are required.
type BindPath struct {
	Source      string                 `json:"source"`
	Destination string                 `json:"destination"`
	Options     map[string]*BindOption `json:"options"`
}

// ImageSrc returns the value of the option image-src for a BindPath, or an
// empty string if the option wasn't set.
func (b *BindPath) ImageSrc() string {
	if b.Options != nil && b.Options["image-src"] != nil {
		src := b.Options["image-src"].Value
		if src == "" {
			return "/"
		}
		return src
	}
	return ""
}

// ID returns the value of the option id for a BindPath, or an empty string if
// the option wasn't set.
func (b *BindPath) ID() string {
	if b.Options != nil && b.Options["id"] != nil {
		return b.Options["id"].Value
	}
	return ""
}

// Readonly returns true if the ro option was set for a BindPath.
func (b *BindPath) Readonly() bool {
	return b.Options != nil && b.Options["ro"] != nil
}

// ParseBindPath parses a an array of strings each specifying one or
// more (comma separated) bind paths in src[:dst[:options]] format, and
// returns all encountered bind paths as a slice. Options may be simple
// flags, e.g. 'rw', or take a value, e.g. 'id=2'.
func ParseBindPath(paths []string) ([]BindPath, error) {
	var binds []BindPath

	// there is a better regular expression to handle
	// that directly without all the logic below ...
	// we need to parse various syntax:
	// source1
	// source1:destination1
	// source1:destination1:option1
	// source1:destination1:option1,option2
	// source1,source2
	// source1:destination1:option1,source2
	re := regexp.MustCompile(`([^,^:]+:?)`)

	// with the regex above we get string array:
	// - source1 -> [source1]
	// - source1:destination1 -> [source1:, destination1]
	// - source1:destination1:option1 -> [source1:, destination1:, option1]
	// - source1:destination1:option1,option2 -> [source1:, destination1:, option1, option2]

	for _, path := range paths {
		concatComma := false
		concatColon := false
		bind := ""
		elem := 0

		for _, m := range re.FindAllString(path, -1) {
			s := strings.TrimSpace(m)

			isOption := false

			for option, flag := range bindOptions {
				if flag {
					if s == option {
						isOption = true
						break
					}
				} else {
					if strings.HasPrefix(s, option+"=") {
						isOption = true
						break
					}
				}
			}

			if elem == 2 && !isOption {
				bp, err := newBindPath(bind)
				if err != nil {
					return nil, fmt.Errorf("while getting bind path: %s", err)
				}
				binds = append(binds, bp)
				elem = 0
				bind = ""
			}

			if elem == 0 {
				// escaped commas and colons
				if (len(s) > 0 && s[len(s)-1] == '\\') || concatComma {
					if !concatComma {
						bind += s[:len(s)-1] + ","
					} else {
						bind += s
						elem++
					}
					concatComma = !concatComma
					continue
				} else if (len(s) >= 2 && s[len(s)-2] == '\\' && s[len(s)-1] == ':') || concatColon {
					bind += s
					if concatColon {
						elem++
					}
					concatColon = !concatColon
					continue
				} else if bind == "" {
					bind = s
				}
			}

			isColon := bind != "" && bind[len(bind)-1] == ':'

			// options are taken only if the bind has a source
			// and a destination
			if elem == 2 && isOption {
				if !isColon {
					bind += ","
				}
				bind += s
				continue
			} else if elem > 2 {
				return nil, fmt.Errorf("wrong bind syntax: %s", bind)
			}

			if bind != "" {
				if isColon {
					if elem > 0 {
						bind += s
					}
					elem++
					continue
				}
				bp, err := newBindPath(bind)
				if err != nil {
					return nil, fmt.Errorf("while getting bind path: %s", err)
				}
				binds = append(binds, bp)
				elem = 0
				bind = ""
				continue
			}
			// new bind path
			bind = s
			elem++
		}

		if bind != "" {
			bp, err := newBindPath(bind)
			if err != nil {
				return nil, fmt.Errorf("while getting bind path: %s", err)
			}
			binds = append(binds, bp)
		}
	}

	return binds, nil
}

func splitBy(str string, sep byte) []string {
	var list []string

	re := regexp.MustCompile(fmt.Sprintf(`(?m)([^\\]%c)`, sep))
	cursor := 0

	indexes := re.FindAllStringIndex(str, -1)
	for i, index := range indexes {
		list = append(list, str[cursor:index[1]-1])
		cursor = index[1]
		if len(indexes)-1 == i {
			return append(list, str[cursor:])
		}
	}

	return append(list, str)
}

// newBindPath returns BindPath record based on the provided bind
// string argument and ensures that the options are valid.
func newBindPath(bind string) (BindPath, error) {
	var bp BindPath

	splitted := splitBy(bind, ':')

	bp.Source = strings.ReplaceAll(splitted[0], "\\:", ":")
	if bp.Source == "" {
		return bp, fmt.Errorf("empty bind source for bind path %q", bind)
	}

	bp.Destination = bp.Source

	if len(splitted) > 1 {
		bp.Destination = splitted[1]
	}

	if len(splitted) > 2 {
		bp.Options = make(map[string]*BindOption)

		for _, value := range strings.Split(splitted[2], ",") {
			valid := false
			for optName, isFlag := range bindOptions {
				if isFlag && optName == value {
					bp.Options[optName] = &BindOption{}
					valid = true
					break
				} else if strings.HasPrefix(value, optName+"=") {
					bp.Options[optName] = &BindOption{Value: value[len(optName+"="):]}
					valid = true
					break
				}
			}
			if !valid {
				return bp, fmt.Errorf("%s is not a valid bind option", value)
			}
		}
	}

	return bp, nil
}
