// Copyright (c) 2018, Sylabs, Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license.  Please
// consult LICENSE.md file distributed with the sources of this project regarding
// your rights to use or distribute this software.

package shell

import "strings"

// ArgsQuoted concatenates a slice of string shell args, quoting each item
func ArgsQuoted(a []string) (quoted string) {
	for _, val := range a {
		quoted = quoted + `"` + Escape(val) + `" `
	}
	quoted = strings.TrimRight(quoted, " ")
	return
}

// Escape performs escaping of shell quotes, backticks and $ characters
func Escape(s string) string {
	escaped := strings.Replace(s, `\`, `\\`, -1)
	escaped = strings.Replace(escaped, `"`, `\"`, -1)
	escaped = strings.Replace(escaped, "`", "\\`", -1)
	escaped = strings.Replace(escaped, `$`, `\$`, -1)
	return escaped
}
