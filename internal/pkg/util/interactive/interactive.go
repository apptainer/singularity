// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package interactive implements all the functions to interactively interact with users
package interactive

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"golang.org/x/crypto/ssh/terminal"
)

var errInvalidChoice = errors.New("invalid choice")
var errPassphraseMismatch = errors.New("passphrases do not match")
var errTooManyRetries = errors.New("too many retries while getting a passphrase")

// askQuestionUsingGenericDescr reads from a file descriptor (more precisely
// from a *os.File object) one line at a time. The file can be a normal file or
// os.Stdin.
// Note that we could imagine a simpler code but we want to make sure that the
// code works properly in the normal case with the default Stdin and when
// redirecting stdin (for testing or when using pipes).
//
// TODO: use a io.ReadSeeker instead of a *os.File
func askQuestionUsingGenericDescr(f *os.File) (string, error) {
	// Get the initial position in the buffer so we can later seek the correct
	// position based on how much data we read. Doing so, we can still benefit
	// from buffered IO and still have a fine-grain controlover reading
	// operations.
	// Note that we do not check for errirs since some cases (e.g., pipes) will
	// actually not allow to perform a seek. This is intended and basically a
	// no-op in that context.
	pos, _ := f.Seek(0, os.SEEK_CUR)
	// Get the data
	scanner := bufio.NewScanner(f)
	tok := scanner.Scan()
	if !tok {
		return "", scanner.Err()
	}
	response := scanner.Text()
	if err := scanner.Err(); err != nil {
		return "", err
	}
	// We did a buffered read (for good reasons, it is generic), so we make
	// sure we reposition ourselves at the end of the data that was read, not
	// the end of the buffer, so we can make sure that we read the data line
	// by line and do not drop data after a lot more data was read from the
	// file descriptor. In other terms, we may have read a very small subset
	// of the available data and make sure we reposition ourselves at the
	// end of the data we handled, not at the end of the data that was read
	// from the file descriptor.
	strLen := 1 // We always move forward, even if we get an empty response
	if len(response) > 1 {
		strLen += len(response)
	}
	// Note that we do not check for errors since some cases (e.g., pipes)
	// will actually not allow to perform a Seek(). This is intended and
	// will not create a problem.
	f.Seek(pos+int64(strLen), os.SEEK_SET)

	return response, nil
}

// AskQuestion prompts the user with a question and return the response
func AskQuestion(format string, a ...interface{}) (string, error) {
	fmt.Printf(format, a...)
	return askQuestionUsingGenericDescr(os.Stdin)
}

// AskYNQuestion prompts the user expecting an answer that's either "y",
// "n" or a blank, in which case defaultAnswer is returned.
func AskYNQuestion(defaultAnswer, format string, a ...interface{}) (string, error) {
	ans, err := AskQuestion(format, a...)
	if err != nil {
		return "", err
	}

	switch ans := strings.ToLower(ans); ans {
	case "y", "yes":
		return "y", nil

	case "n", "no":
		return "n", nil

	case "":
		return defaultAnswer, nil

	default:
		return "", fmt.Errorf("invalid answer: %q", ans)
	}
}

// AskNumberInRange prompts the user expecting an answer that is a number
// between start and end.
func AskNumberInRange(start, end int, format string, a ...interface{}) (int, error) {
	ans, err := AskQuestion(format, a...)
	if err != nil {
		return 0, err
	}
	fmt.Println("Answer:", ans)

	n, err := strconv.ParseInt(ans, 10, 32)
	if err != nil {
		return 0, err
	}

	m := int(n)

	if m < start || m > end {
		return 0, errInvalidChoice
	}

	return m, nil
}

// AskQuestionNoEcho works like AskQuestion() except it doesn't echo user's input
func AskQuestionNoEcho(format string, a ...interface{}) (string, error) {
	fmt.Printf(format, a...)

	var response string
	var err error
	// Go provides a package for handling terminal and more specifically
	// reading password from terminal. We want to use the package when possible
	// since it gives us an easy and secure way to interactively get the
	// password from the user. However, this is only working when the
	// underlying file descriptor is associated to a VT100 terminal, not with
	// other file descriptors, including when redirecting Stdin to an actual
	// file in the context of testing or in the context of pipes.
	if terminal.IsTerminal(int(os.Stdin.Fd())) {
		var resp []byte
		resp, err = terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", err
		}
		response = string(resp)
	} else {
		response, err = askQuestionUsingGenericDescr(os.Stdin)
		if err != nil {
			return "", err
		}
	}
	fmt.Println("")
	return string(response), nil
}

// GetPassphrase will ask the user for a password with int number of
// retries.
func GetPassphrase(message string, retries int) (string, error) {
	ask := func() (string, error) {
		pass1, err := AskQuestionNoEcho(message)
		if err != nil {
			return "", err
		}

		pass2, err := AskQuestionNoEcho("Retype your passphrase : ")
		if err != nil {
			return "", err
		}

		if pass1 != pass2 {
			return "", errPassphraseMismatch
		}

		return pass1, nil
	}

	for i := 0; i < retries; i++ {
		switch passphrase, err := ask(); err {
		case nil:
			// we got it!
			return passphrase, nil
		case errPassphraseMismatch:
			// retry
			sylog.Warningf("%v", err)
		default:
			// something else went wrong, bail out
			return "", err
		}
	}

	return "", errTooManyRetries
}
