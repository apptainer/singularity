// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package verify

import (
	"bytes"
	//"encoding/json"
	"fmt"
	"os/exec"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/sylabs/singularity/e2e/internal/e2e"
)

type ctx struct {
	env e2e.TestEnv
}

const containerTesterSIF = "testdata/verify_container_corrupted.sif"

type Key struct {
	Signer KeyEntity `json:"Signer"`
}

// KeyEntity holds all the key info, used for json output.
type KeyEntity struct {
	Name        string `json:"Name"`
	Fingerprint string `json:"Fingerprint"`
	Local       bool   `json:"Local"`
	KeyCheck    bool   `json:"KeyCheck"`
	DataCheck   bool   `json:"DataCheck"`
}

// KeyList is a list of one or more keys.
type KeyList []*Key

func (c *ctx) runVerifyCommand() ([]byte, []byte, error) {
	argv := []string{"verify", "--json", containerTesterSIF}
	cmd := exec.Command(c.env.CmdPath, argv...)

	cmdStdout := bytes.NewBuffer(nil)
	cmdStderr := bytes.NewBuffer(nil)

	cmd.Stdout = cmdStdout
	cmd.Stderr = cmdStderr

	err := cmd.Run()
	if err != nil {
		return cmdStdout.Bytes(), cmdStdout.Bytes(), err
	}

	err = cmd.Wait()
	if err != nil {
		return cmdStdout.Bytes(), cmdStdout.Bytes(), err
	}

	fmt.Println("CMD: ", string(cmdStdout.Bytes()))

	return cmdStdout.Bytes(), cmdStderr.Bytes(), nil
}

func (c *ctx) singularityInspect(t *testing.T) {
	tests := []struct {
		name      string
		keyNum    int
		entity    string
		expectOut string // expectOut should be a string of expected output
	}{
		{
			name:      "corrupted verify test",
			keyNum:    0,
			entity:    "Name",
			expectOut: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Inspect the container, and get the output
			stdout, _, err := c.runVerifyCommand()
			if err != nil {
				fmt.Println("ERRRRRUNNINF: ", err)
				//t.Fatalf("unexpected failure: %s: %s", string(out), err)
			}

			//var key []map[string]map[string]interface{}

			//err = json.Unmarshal(stdout, &key)
			//if err != nil {
			//	fmt.Println("ERROR: ", err)
			//}

			//fmt.Println("BARRRRRRRRRRR: ", key[0]["Signer"]["Name"])
			//fmt.Printf("FFFFFFFFFFOOO: %+v\n", key)

			//var key KeyList

			//v := key[0].Signer.Name

			//	for i := 0; i < len(key.Signer); i++ {
			//		fmt.Println("User Type: " + key.Signer[i].Name)
			//		fmt.Println("User Name: " + key.Signer[i].Fingerprint)
			//	}

			//fmt.Println("INFP___: ", key[0].Signer.Name)
			//fmt.Println("INFP___: ", key[tt.keyNum].Signer)

			// Parse the output

			foo := []string{"SignerKeys", "Signer", "Name"}

			v, err := jsonparser.GetString(stdout, foo...)
			if err != nil {
				t.Fatalf("unable to get expected output from json: %v", err)
			}
			// Compare the output, with the expected output

			fmt.Println("VVV: ", v)

			//if v != tt.expectOut {
			//	t.Fatalf("unexpected failure: got: %s, expecting: %s", v, tt.expectOut)
			//}

		})
	}

}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env: env,
	}

	return func(t *testing.T) {
		t.Run("singularityVerify", c.singularityInspect)
	}
}
