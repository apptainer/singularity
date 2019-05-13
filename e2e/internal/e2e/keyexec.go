// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// PullDefaultPublicKey will pull the public Sylabs Admin key
func PullDefaultPublicKey(t *testing.T) {
	LoadEnv(t, &testenv)

	//func PullDefaultPublicKey(t *testing.T, cmdPath string) {
	argv := []string{"key", "pull", "8883491F4268F173C6E5DC49EDECE4F3F38D871E"}

	execKey := exec.Command(testenv.CmdPath, argv...)

	out, err := execKey.CombinedOutput()
	if err != nil {
		t.Log(string(out))
		t.Fatalf("Unable to pull key: %v", err)
	}
}

// RemoveDefaultPublicKey will pull the public Sylabs Admin key
func RemoveDefaultPublicKey(t *testing.T) {
	LoadEnv(t, &testenv)

	argv := []string{"key", "remove", "8883491F4268F173C6E5DC49EDECE4F3F38D871E"}
	execKey := exec.Command(testenv.CmdPath, argv...)

	out, err := execKey.CombinedOutput()
	if err != nil {
		t.Log(string(out))
		t.Fatalf("Unable to pull key: %v", err)
	}
}

// RunKeyCmd will run a 'singularty key' command, with any args that are set in commands.
func RunKeyCmd(t *testing.T, cmdPath string, commands []string, file, stdin string) (string, []byte, error) {
	argv := []string{"key"}
	argv = append(argv, commands...)

	if file != "" {
		argv = append(argv, file)
	}

	cmd := fmt.Sprintf("%s %s", cmdPath, strings.Join(argv, " "))
	execKey := exec.Command(cmdPath, argv...)

	stdinRun, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer stdinRun.Close()

	_, err = io.WriteString(stdinRun, stdin)
	if err != nil {
		t.Fatalf("%v", err)
	}

	_, err = stdinRun.Seek(0, os.SEEK_SET)
	if err != nil {
		t.Fatalf("%v", err)
	}

	execKey.Stdin = stdinRun

	out, err := execKey.CombinedOutput()

	t.Log("???????EXEC_COMMAND: ", cmd)
	//	t.Log("???????OUTPUT: ", string(out))

	return cmd, out, err
}
