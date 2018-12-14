// Copyright (c) 2018, Sylabs, Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license.  Please
// consult LICENSE file distributed with the sources of this project regarding
// your rights to use or distribute this software.

package shell

import "testing"

func TestArgsQuoted(t *testing.T) {
	var quoteTests = []struct {
		name     string
		input    []string
		expected string
	}{
		{"Single arg", []string{`Hello`}, `"Hello"`},
		{"Two args", []string{`Hello`, `me`}, `"Hello" "me"`},
		{"Three args", []string{`Hello`, `there`, `me`}, `"Hello" "there" "me"`},
		{`Args with escaping`, []string{`Hello`, `\n me`}, `"Hello" "\\n me"`},
	}

	for _, test := range quoteTests {
		t.Run(test.name, func(t *testing.T) {
			quoted := ArgsQuoted(test.input)
			if quoted != test.expected {
				t.Errorf("got %s, expected %s", quoted, test.expected)
			}
		})
	}

}

func TestEscape(t *testing.T) {
	var escapeTests = []struct {
		input    string
		expected string
	}{
		{`Hello \n me`, `Hello \\n me`},
		{`"Hello"`, `\"Hello\"`},
		{"`ls`", "\\`ls\\`"},
		{`$PATH`, `\$PATH`},
	}

	for _, test := range escapeTests {
		t.Run(test.input, func(t *testing.T) {
			escaped := Escape(test.input)
			if escaped != test.expected {
				t.Errorf("got %s, expected %s", escaped, test.expected)
			}
		})
	}

}
