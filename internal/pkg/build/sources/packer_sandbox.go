// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
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

	//p.b.Recipe.ImageData.Labels = make(map[string]map[string]string, 1)
	if p.b.JSONLabels == nil {
		p.b.JSONLabels = make(map[string]string, 1)
	}

	// Read the labels from the sandbox
	jsonLabels, err := ioutil.ReadFile(filepath.Join(rootfs, ".singularity.d/labels.json"))
	if err != nil {
		return nil, fmt.Errorf("unable to read json file: %s", err)
	}

	// Then add them to ImageData
	err = json.Unmarshal(jsonLabels, &p.b.JSONLabels)
	//err = json.Unmarshal(jsonLabels, &p.b.Recipe.ImageData.Labels)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal json labels: %s", err)
	}

	// copy filesystem into bundle rootfs
	sylog.Debugf("Copying file system from %s to %s in Bundle\n", rootfs, p.b.RootfsPath)
	var stderr bytes.Buffer
	cmd := exec.Command("cp", "-r", rootfs+`/.`, p.b.RootfsPath)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cp Failed: %v: %v", err, stderr.String())
	}

	return p.b, nil
}
