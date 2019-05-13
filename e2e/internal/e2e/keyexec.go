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
