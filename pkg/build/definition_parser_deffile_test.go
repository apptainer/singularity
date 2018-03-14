/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package build

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestParseDefinitionFile(t *testing.T) {
	testFilesOK := map[string]string{
		"docker": "./mock/docker/docker",
	}
	testFilesBAD := map[string]string{
		"bad_section": "./mock/bad_section/bad_section",
	}
	resultDefinition := map[string]Definition{
		"docker": Definition{
			Header: map[string]string{"bootstrap": "docker", "from": "<registry>/<namespace>/<container>:<tag>@<digest>",
				"registry": "http://custom_registry", "namespace": "namespace", "includecmd": "yes"},
			ImageData: imageData{
				imageScripts: imageScripts{
					Help: `Hello Help!
# # double Hashtag comment`,
					Environment: `    VADER=badguy
    LUKE=goodguy
    SOLO=someguy # comment 4
    export VADER LUKE SOLO`,
					Runscript: `    echo "Mock!"
    echo "Arguments received: $*" # This is a very long comment
    exec echo "$@"`,
					Test: ``,
				},
			},
			BuildData: buildData{
				Files: map[string]string{`mock1.txt`: ``, `mock2.txt`: `/opt`},
				buildScripts: buildScripts{
					Pre: ``,
					Setup: `    touch ${SINGULARITY_ROOTFS}/mock.txt
    touch mock.txt

# Some dummy comment 2`,
					Post: `    echo 'this is a command so long that the user had to' \
    'add a new line'
    echo 'export GOPATH=$HOME/go' >> $SINGULARITY_ENVIRONMENT`,
				},
			},
		},
	}

	// Loop through the Deffiles OK
	for k := range testFilesOK {
		t.Logf("=>\tRunning test for Deffile:\t\t[%s]", k)
		f, err := ioutil.TempFile(os.TempDir(), fmt.Sprintf("singularity_parser_test_%s", k))
		if err != nil {
			t.Log(err)
			t.Fail()
		}
		defer os.Remove(f.Name())

		r, err := os.Open(testFilesOK[k])
		if err != nil {
			t.Error(err)
		}
		defer r.Close()

		Df, err := ParseDefinitionFile(r)
		if err != nil {
			t.Log(err)
			t.Fail()
		}

		// And....compare the output (fingers crossed)
		if !reflect.DeepEqual(resultDefinition[k], Df) {
			t.Log("<=\tFailed to parse Deffinition header")
			t.Fail()
		}
	}

	// Loop through the Deffiles BAD (must return error)
	for k, v := range testFilesBAD {
		t.Logf("=>\tRunning test for Bad Deffile:\t\t[%s]", k)
		r, err := os.Open(v)
		if err != nil {
			t.Error(err)
		}
		defer r.Close()

		// Parse must return err and a nil Definition struct
		_, err = ParseDefinitionFile(r)
		if err == nil {
			t.Logf("<=\tFailed to parse Bad Deffinition file:\t[%s]", k)
			t.Log(err)
			t.Fail()
		}
	}
}
