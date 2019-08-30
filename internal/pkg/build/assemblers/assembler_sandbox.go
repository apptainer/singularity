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
	sylog.Infof("Creating sandbox directory...")

	jsonLabels := make(map[string]string, 1)
	// Copy the labels
	for k, v := range b.Recipe.ImageData.Labels {
		jsonLabels[k] = v
	}

	sylog.Infof("Adding labels...")

	metadata.GetImageInfoLabels(jsonLabels, b)

	text, err := json.MarshalIndent(jsonLabels, "", "\t")
	if err != nil {
		return fmt.Errorf("unable to marshal json: %s", err)
	}

	err = ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/labels.json"), []byte(text), 0644)
	if err != nil {
		return fmt.Errorf("unable to write to labels file: %s", err)
	}

	// move bundle rootfs to sandboxdir as final sandbox
	sylog.Debugf("Moving sandbox from %v to %v", b.Rootfs(), path)
	if _, err := os.Stat(path); err == nil {
		os.RemoveAll(path)
	}

	var stderr bytes.Buffer
	cmd := exec.Command("mv", b.Rootfs(), path)
	cmd.Stderr = &stderr
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("sandbox assemble failed: %v: %v", err, stderr.String())
	}

	return nil
}
