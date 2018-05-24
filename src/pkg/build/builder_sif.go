// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"

	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/image"
	"github.com/singularityware/singularity/src/pkg/sif"
)

func pumpPipe(src *io.PipeReader, dest io.Writer) {
	s := bufio.NewScanner(src)
	s.Split(bufio.ScanBytes)

	for s.Scan() {
		dest.Write(s.Bytes())
	}

	return
}

// SIFBuilder is an interface that enables building a SIF image.
type SIFBuilder struct {
	Def    Definition
	Stdout io.Writer
	Stderr io.Writer

	outsrc *io.PipeReader
	errsrc *io.PipeReader
	sbuild *exec.Cmd
}

// NewSIFBuilder creates a new SIFBuilder.
func NewSIFBuilder(imagePath string, d Definition) (b *SIFBuilder, err error) {
	b = &SIFBuilder{}

	builderJSON, err := json.Marshal(d)
	b.sbuild = exec.Command(buildcfg.SBINDIR+"/sbuild", "sif", string(builderJSON), imagePath)

	b.outsrc, b.sbuild.Stdout = io.Pipe()
	b.errsrc, b.sbuild.Stderr = io.Pipe()

	b.Stdout = os.Stdout
	b.Stderr = os.Stderr

	return b, err
}

// Build completes a build. The supplied context can be used for cancellation.
func (b *SIFBuilder) Build(ctx context.Context) (err error) {
	err = b.sbuild.Start()
	if err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- b.sbuild.Wait()
	}()

	// Pump the build's stdout/stderr to os.stdout && os.stderr
	go pumpPipe(b.outsrc, b.Stdout)
	go pumpPipe(b.errsrc, b.Stderr)

	select {
	case err := <-done:
		b.outsrc.Close()
		b.errsrc.Close()

		if err != nil {
			return err
		}

		fmt.Println("Build Succeeded")
		return nil
	}
}

// SIFBuilder2 is an interface that enables building a SIF image.
type SIFBuilder2 struct {
	def   Definition
	image *sif.SIF
	path  string
	p     Provisioner
	tmpfs *image.Sandbox
}

// NewSifBuilderJSON creates a new SIFBuilder2 using the supplied JSON.
func NewSifBuilderJSON(r io.Reader, imagePath string) (b *SIFBuilder2, err error) {
	var d Definition
	b = &SIFBuilder2{}

	decoder := json.NewDecoder(r)

	for {
		if err = decoder.Decode(&d); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
	}

	b.def = d
	b.path = imagePath
	b.tmpfs, err = image.TempSandbox(path.Base(imagePath) + "-")
	if err != nil {
		return b, err
	}

	uri := d.Header["bootstrap"] + "://" + d.Header["from"]
	b.p, err = NewProvisionerFromURI(uri)

	return b, err
}

// Build completes a build. The supplied context can be used for cancellation.
func (b *SIFBuilder2) Build(ctx context.Context) (err error) {
	b.p.Provision(b.tmpfs)
	img, err := sif.FromSandbox(b.tmpfs, b.path)
	if err != nil {
		return err
	}

	b.image = img

	return nil
}

/*
func NewSifBuilder(imagePath string, d Definition) (b *SifBuilder, err error) {
	r, w, _ := os.Pipe()
	er, ew, _ := os.Pipe()

	b = &SifBuilder{
		Definition: d,
		path:       imagePath,
		Out:        io.MultiReader(er, r),
		outWrite:   w,
		errWrite:   ew,
		outRead:    r,
		errRead:    er,
	}

	b.tmpfs, err = image.TempSandbox(path.Base(imagePath) + "-")
	if err != nil {
		return b, err
	}

	uri := d.Header["bootstrap"] + "://" + d.Header["from"]
	b.p, err = NewProvisionerFromURI(uri)

	return b, err
}*/

/*
// Build provisions a temporaly file system from a definition file
// and build a SIF afterwards.
func (b *SifBuilder) Build() (err error) {
	oldstdout := os.Stdout
	oldstderr := os.Stderr

	os.Stdout = b.outWrite
	os.Stderr = b.errWrite

	defer func() {
		os.Stdout = oldstdout
		os.Stderr = oldstderr
		//b.outRead.Close()
		b.outWrite.Close()
		//b.errRead.Close()
		b.errWrite.Close()
	}()

	err := b.p.Provision(b.tmpfs)
	if err != nil {
		return err
	}

	img, err := image.SIFFromSandbox(b.tmpfs, b.path)
	if err != nil {
		return err
	}

	b.Image = img

	return nil
}

func (b *SifBuilder) createSifFile() (err error) {
	return nil
}

func (b *SifBuilder) mksquashfs() (err error) {
	return nil
}
*/
