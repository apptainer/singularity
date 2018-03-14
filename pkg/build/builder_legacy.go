/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.
  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package build

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/golang/glog"
)

// LegacyBuilder is an interface that enables building an image using Singularity 2.x binary
// on the system. This is temporary so we can build while working towards 3.0
type LegacyBuilder struct {
	Definition
	ImagePath string
	Out       io.ReadCloser
	In        io.WriteCloser
	Cmd       *exec.Cmd
	Proc      *os.Process
}

func NewLegacyBuilder(p string, c context.Context, d Definition) (builder *LegacyBuilder, err error) {
	singularity, err := exec.LookPath("singularity")
	if err != nil {
		glog.Fatal("Singularity is not installed on this system")
	}

	f, err := ioutil.TempFile("/tmp/", "Singularity-Definition-")
	if err != nil {
		glog.Fatal(err)
		return nil, err
	}

	d.WriteDefinitionFile(f)

	builder = &LegacyBuilder{
		Definition: d,
		Cmd:        exec.CommandContext(c, singularity, "build", p, f.Name()),
		ImagePath:  p,
	}

	builder.Out, err = builder.Cmd.StdoutPipe()
	if err != nil {
		glog.Fatal(err)
		return nil, err
	}

	builder.In, err = builder.Cmd.StdinPipe()
	if err != nil {
		glog.Fatal(err)
		return nil, err
	}

	return
}

func (b *LegacyBuilder) Build() {
	err := b.Cmd.Start()

	if err != nil {
		glog.Fatal(err)
		return
	}

	b.Proc = b.Cmd.Process
}
