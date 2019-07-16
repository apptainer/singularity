// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package interactive

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func generateQuestionInput(t *testing.T, input string) (*os.File, *os.File) {
	// Each line of the string represents a virtual different answer from a user
	testBytes := []byte(input)

	// we create a temporary file that will act as Stdin
	testFile, err := ioutil.TempFile("", "inputTest")
	if err != nil {
		t.Fatalf("failed to create temporary file: %s", err)
	}

	// Write the data that we will later on need
	_, err = testFile.Write(testBytes)
	if err != nil {
		testFile.Close()
		os.Remove(testFile.Name())
		t.Fatalf("failed to write to %s: %s", testFile.Name(), err)
	}

	// Reposition to the beginning of file to ensure there is something to read
	_, err = testFile.Seek(0, os.SEEK_SET)
	if err != nil {
		testFile.Close()
		os.Remove(testFile.Name())
		t.Fatalf("failed to seek to beginning of file %s: %s", testFile.Name(), err)
	}

	// Redirect Stdin
	savedStdin := os.Stdin
	os.Stdin = testFile

	return testFile, savedStdin
}

func TestAskNYQuestion(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	const (
		defaultAnswer = "default"
	)

	tests := []struct {
		name           string
		input          string
		expectedOutput string
		shallPass      bool
	}{
		{
			name:           "y",
			input:          "y",
			expectedOutput: "y",
			shallPass:      true,
		},
		{
			name:           "yes",
			input:          "yes",
			expectedOutput: "y",
			shallPass:      true,
		},
		{
			name:           "n",
			input:          "n",
			expectedOutput: "n",
			shallPass:      true,
		},
		{
			name:           "no",
			input:          "no",
			expectedOutput: "n",
			shallPass:      true,
		},
		{
			name:           "default",
			input:          "",
			expectedOutput: defaultAnswer,
			shallPass:      true,
		},
		{
			name:           "invalid",
			input:          "abc",
			expectedOutput: "",
			shallPass:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile, savedStdin := generateQuestionInput(t, tt.input)
			defer tempFile.Close()
			defer os.RemoveAll(tempFile.Name())
			defer func() {
				os.Stdin = savedStdin
			}()

			res, err := AskYNQuestion(defaultAnswer, "")
			if tt.shallPass == true && (err != nil || res != tt.expectedOutput) {
				t.Fatalf("test %s failed while expected to pass", tt.name)
			}
			if tt.shallPass == false && (err == nil || res != tt.expectedOutput) {
				t.Fatalf("test %s passed while expected to fail", tt.name)
			}
		})
	}

}

func TestAskNumberInRange(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	rangeStart := 0
	rangeEnd := 10

	tests := []struct {
		name           string
		input          string
		expectedOutput int
		shallPass      bool
	}{
		{
			name:           "in-range number",
			input:          "5",
			expectedOutput: 5,
			shallPass:      true,
		},
		{
			name:           "out of range number",
			input:          "12",
			expectedOutput: 0,
			shallPass:      false,
		},
		{
			name:           "invalid type (string)",
			input:          "abc",
			expectedOutput: 0,
			shallPass:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile, savedStdin := generateQuestionInput(t, tt.input)
			defer tempFile.Close()
			defer os.RemoveAll(tempFile.Name())
			defer func() {
				os.Stdin = savedStdin
			}()

			res, err := AskNumberInRange(rangeStart, rangeEnd, "")
			if tt.shallPass == true && (err != nil || res != tt.expectedOutput) {
				t.Fatalf("test %s failed while expected to pass", tt.name)
			}
			if tt.shallPass == false && (err == nil || res != tt.expectedOutput) {
				t.Fatalf("test %s passed while expected to fail", tt.name)
			}
		})
	}
}

func TestAskQuestion(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	// Each line of the string represents a virtual different answer from a user
	testStr := "test test test\ntest2 test2\n\ntest3"
	tempFile, savedStdin := generateQuestionInput(t, testStr)
	defer tempFile.Close()
	defer os.RemoveAll(tempFile.Name())
	defer func() {
		os.Stdin = savedStdin
	}()

	// Actual test, run the test with the first line
	output, err := AskQuestion("Question test: ")
	if err != nil {
		t.Fatalf("failed to get response from AskQuestion(): %s", err)
	}

	// We analyze the result. We always make sure we do not get the '\n'
	firstAnswer := testStr[:strings.Index(testStr, "\n")]
	restAnswer := testStr[len(firstAnswer)+1:]
	if output != firstAnswer {
		t.Fatal("AskQuestion() returned", output, "instead of", firstAnswer)
	}

	// Test with the second line
	output, err = AskQuestion("Question test 2: ")
	if err != nil {
		t.Fatalf("failed to get response: %s", err)
	}

	// We analyze the result
	secondAnswer := restAnswer[:strings.Index(restAnswer, "\n")]
	if output != secondAnswer {
		t.Fatalf("AskQuestion() returned: %s instead of: %s", output, secondAnswer)
	}

	// Test with the third line
	output, err = AskQuestion("Question test 3: ")
	if err != nil {
		t.Fatalf("failed to get response: %s", err)
	}

	// We analyze the result
	if output != "" {
		t.Fatalf("AskQuestion() returned: %s instead of an empty string", output)
	}

	// Test with the final line
	output, err = AskQuestion("Question test 4: ")
	if err != nil {
		t.Fatalf("failed to get response: %s", err)
	}

	finalAnswer := restAnswer[len(secondAnswer)+2:] // We have to account for 2 "\n"
	if output != finalAnswer {
		t.Fatalf("AskQuestion() returned: %s instead of: %s", output, finalAnswer)
	}
}

func TestAskQuestionNoEcho(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	testStr := "test test\ntest2 test2 test2\n\ntest3 test3 test3 test3"
	tempFile, savedStdin := generateQuestionInput(t, testStr)
	defer tempFile.Close()
	defer os.RemoveAll(tempFile.Name())
	defer func() {
		os.Stdin = savedStdin
	}()

	// Test AskQuestionNoEcho(), starting with the first line
	output, err := AskQuestionNoEcho("Test question 1: ")
	if err != nil {
		t.Fatalf("failed to get output from AskQuestionNoEcho(): %s", err)
	}

	// Analyze the result
	firstAnswer := testStr[:strings.Index(testStr, "\n")]
	restAnswer := testStr[len(firstAnswer)+1:] // Ignore "\n"
	if output != firstAnswer {
		t.Fatalf("AskQuestionNoEcho() returned %s instead of %s", output, firstAnswer)
	}

	// Test with the second line
	output, err = AskQuestionNoEcho("Test question 2: ")
	if err != nil {
		t.Fatalf("failed to get output from AskQuestionNoEcho(): %s", err)
	}

	// We analyze the answer
	secondAnswer := restAnswer[:strings.Index(restAnswer, "\n")]
	if output != secondAnswer {
		t.Fatalf("AskQuestionNoEcho() returned %s instead of %s", output, secondAnswer)
	}

	// Test with third line
	output, err = AskQuestionNoEcho("Test question 3: ")
	if err != nil {
		t.Fatalf("failed to get output from AskQuestionNoEcho(): %s", err)
	}

	// We analyze the answer
	if output != "" {
		t.Fatalf("AskQuestionNoEcho() returned %s instead of an empty string", output)
	}

	// Test with the final line
	output, err = AskQuestionNoEcho("Test question 4: ")
	if err != nil {
		t.Fatalf("failed to get output from AskQuestionNoEcho(): %s", err)
	}

	finalAnswer := restAnswer[len(secondAnswer)+2:] // We have to account for 2 "\n"
	if output != finalAnswer {
		t.Fatalf("AskQuestionNoEcho() returned %s instead of %s", output, finalAnswer)
	}
}

func TestGetPassphrase(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		name      string
		input     string
		shallPass bool
	}{
		{
			name:      "valid case",
			input:     "mypassphrase\nmypassphrase\n",
			shallPass: true,
		},
		{
			name:      "unmatching passphrases",
			input:     "mypassphrase\nsomethingelse\n",
			shallPass: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file that will act as input from stdin
			tempFile, savedStdin := generateQuestionInput(t, tt.input)
			defer tempFile.Close()
			defer os.RemoveAll(tempFile.Name())
			defer func() {
				os.Stdin = savedStdin
			}()

			pass, err := GetPassphrase("Test: ", 1)
			if tt.shallPass && (err != nil || pass != "mypassphrase") {
				t.Fatalf("test %s is expected to succeed but failed: %s", tt.name, err)
			}
			if !tt.shallPass && err == nil {
				t.Fatalf("invalid case %s succeeded", tt.name)
			}
		})
	}
}
