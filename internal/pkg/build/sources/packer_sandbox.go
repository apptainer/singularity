// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
)

// SandboxPacker holds the locations of where to pack from and to
// Ext3Packer holds the locations of where to back from and to, aswell as image offset info
type SandboxPacker struct {
	srcdir string
	b      *types.Bundle
}

// Pack puts relevant objects in a Bundle!
func (p *SandboxPacker) Pack() (*types.Bundle, error) {
	rootfs := p.srcdir

	// TODO: FIXME: !!!

	fmt.Printf("\n\n\nNEWDATA FOR SANDBOX\n")

	//
	// Open the SIF
	//

	inspectDataJSON := make(map[string]map[string]string, 1)
	inspectDataJSON["labels"] = make(map[string]string, 1)

	p.b.Recipe.ImageData.Labels = make(map[string]string, 1)

	foobar, err := ioutil.ReadFile(filepath.Join(rootfs, ".singularity.d/labels.json"))
	if err != nil {
		return nil, fmt.Errorf("unable to read json file: %s", err)
	}
	fmt.Print(string(foobar))

	//fmt.Printf("BARTMP: %+v\n", p.b)

	err = json.Unmarshal(foobar, &p.b.Recipe.ImageData.Labels)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal json labels: %s", err)
	}

	//	for _, v := range sifData {
	//		metaData := v.GetData(&fimg)
	//		err := json.Unmarshal(metaData, &b.Recipe.ImageData.Labels)
	//		if err != nil {
	//			sylog.Fatalf("Unable to get json: %s", err)
	//		}
	//
	//		//		var hrOut map[string]*json.RawMessage
	//		//		err := json.Unmarshal(metaData, &hrOut)
	//		//		if err != nil {
	//		//			sylog.Fatalf("Unable to get json: %s", err)
	//		//		}
	//		//		//inspectData += "== labels ==\n"
	//		//		for k := range hrOut {
	//		//			fmt.Printf("INFOOOOO: %s: %s\n", k, string(*hrOut[k]))
	//		//			inspectDataJSON["labels"][k] = string(*hrOut[k])
	//		//			b.Recipe.ImageData.Labels[k] = string(*hrOut[k])
	//		//		}
	//
	//	}
	//
	// copy filesystem into bundle rootfs
	sylog.Debugf("Copying file system from %s to %s in Bundle\n", rootfs, p.b.Rootfs())
	var stderr bytes.Buffer
	cmd := exec.Command("cp", "-r", rootfs+`/.`, p.b.Rootfs())
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cp Failed: %v: %v", err, stderr.String())
	}

	return p.b, nil
}
