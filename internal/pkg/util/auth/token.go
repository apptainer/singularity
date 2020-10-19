/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE.md file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package auth

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"
)

var (
	// ErrTokenTooShort is returned for token shorter than 200 b
	ErrTokenTooShort = errors.New("token is too short to be valid")
	// ErrTokenToolong is returned for token longer than 4096 b
	ErrTokenToolong = errors.New("token is too large to be valid")
	// ErrEmptyToken is returned for empty token string
	ErrEmptyToken = errors.New("token file is empty")
	// ErrTokenFileNotFound is returned when token file not found
	ErrTokenFileNotFound = errors.New("authentication token file not found")
	// ErrCouldntReadFile is returned for issues when reading file
	ErrCouldntReadFile = errors.New("couldn't read your Sylabs authentication token")
)

// ReadToken reads a sylabs JWT auth token from a file
func ReadToken(tokenPath string) (token string, err error) {
	// check if token file exist
	_, err = os.Stat(tokenPath)
	if os.IsNotExist(err) {
		return "", ErrTokenFileNotFound
	}

	buf, err := ioutil.ReadFile(tokenPath)
	if err != nil {
		return "", ErrCouldntReadFile
	}

	lines := strings.Split(string(buf), "\n")
	if len(lines) < 1 {
		return "", ErrEmptyToken
	}

	// A valid RSA signed token is at least 200 chars with no extra payload
	token = lines[0]
	if len(token) < 200 {
		return "", ErrTokenTooShort
	}

	// A token should never be bigger than 4Kb - if it is we will have problems
	// with header buffers
	if len(token) > 4096 {
		return "", ErrTokenToolong
	}

	return token, nil
}
