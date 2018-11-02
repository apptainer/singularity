/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE.md file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package auth

import (
	"io/ioutil"
	"os"
	"strings"
)

const (
	// WarningTokenTooShort Warning return for token shorter than 200 b
	WarningTokenTooShort = "Token is too short to be valid"
	// WarningTokenToolong Warning return for token longer than 4096 b
	WarningTokenToolong = "Token is too large to be valid"
	// WarningEmptyToken Warning return for empty token string
	WarningEmptyToken = "Token file is empty"
	// WarningTokenFileNotFound token file not found
	WarningTokenFileNotFound = "Authentication token file not found"
	// WarningCouldntReadFile Warning return for issues when reading file
	WarningCouldntReadFile = "Couldn't read your Sylabs authentication token"
)

// ReadToken reads a sylabs JWT auth token from a file
func ReadToken(tokenPath string) (token, warning string) {
	// check if token file exist
	_, err := os.Stat(tokenPath)
	if os.IsNotExist(err) {
		return "", WarningTokenFileNotFound
	}

	buf, err := ioutil.ReadFile(tokenPath)
	if err != nil {
		return "", WarningCouldntReadFile
	}

	lines := strings.Split(string(buf), "\n")
	if len(lines) < 1 {
		return "", WarningEmptyToken
	}

	// A valid RSA signed token is at least 200 chars with no extra payload
	token = lines[0]
	if len(token) < 200 {
		return "", WarningTokenTooShort
	}

	// A token should never be bigger than 4Kb - if it is we will have problems
	// with header buffers
	if len(token) > 4096 {
		return "", WarningTokenToolong
	}

	return
}
