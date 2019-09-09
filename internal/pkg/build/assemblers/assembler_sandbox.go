// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package assemblers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/build/metadata"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
)

// SandboxAssembler stores data required to assemble the image.
type SandboxAssembler struct {
	// Nothing yet
}

// Assemble creates a Sandbox image from a Bundle
func (a *SandboxAssembler) Assemble(b *types.Bundle, path string) (err error) {
	// todo(sasha): get rid of this completely since we can create sandbox in b.RootfsPath?
	sylog.Infof("Creating sandbox directory...")

	sylog.Infof("Adding labels...")

	// Copy the labels from the %applabels
	for name, l := range b.JSONLabels {
		b.Recipe.ImageData.Labels[name] = make(map[string]string, 1)
		for k, v := range l {
			b.Recipe.ImageData.Labels[name][k] = v
		}
	}

	// Get the schema labels, overidding the old ones
	metadata.GetImageInfoLabels(b.Recipe.ImageData.Labels, nil, b)

	text, err := json.MarshalIndent(b.Recipe.ImageData.Labels, "", "\t")
	if err != nil {
		return fmt.Errorf("unable to marshal json: %s", err)
	}

	err = ioutil.WriteFile(filepath.Join(b.RootfsPath, "/.singularity.d/labels.json"), []byte(text), 0644)
	if err != nil {
		return fmt.Errorf("unable to write to labels file: %s", err)
	}

	// move bundle rootfs to sandboxdir as final sandbox
	sylog.Debugf("Moving sandbox from %v to %v", b.RootfsPath, path)
	if _, err := os.Stat(path); err == nil {
		os.RemoveAll(path)
	}

	var stderr bytes.Buffer
	cmd := exec.Command("mv", b.RootfsPath, path)
	cmd.Stderr = &stderr
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("sandbox assemble failed: %v: %v", err, stderr.String())
	}

	return nil
}
