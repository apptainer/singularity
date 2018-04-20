/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package build

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"

	"github.com/singularityware/singularity/pkg/config"
	"github.com/singularityware/singularity/pkg/image"
)

func pumpPipe(src *io.PipeReader, dest io.Writer) {
	s := bufio.NewScanner(src)
	s.Split(bufio.ScanBytes)

	for s.Scan() {
		dest.Write(s.Bytes())
	}

	return
}

type SifBuilder struct {
	Def    Definition
	Stdout io.Writer
	Stderr io.Writer

	outsrc *io.PipeReader
	errsrc *io.PipeReader
	sbuild *exec.Cmd
}

func NewSifBuilder(imagePath string, d Definition) (b *SifBuilder, err error) {
	b = &SifBuilder{}

	builderJSON, err := json.Marshal(d)
	b.sbuild = exec.Command(config.BUILDDIR+"/sbuild", "sif", string(builderJSON), imagePath)

	b.outsrc, b.sbuild.Stdout = io.Pipe()
	b.errsrc, b.sbuild.Stderr = io.Pipe()

	b.Stdout = os.Stdout
	b.Stderr = os.Stderr

	return b, err
}

func (b *SifBuilder) Build() (err error) {
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

type sifBuilder struct {
	def   Definition
	image *image.SIF
	path  string
	p     Provisioner
	tmpfs *image.Sandbox
}

func NewSifBuilderJSON(r io.Reader, imagePath string) (b *sifBuilder, err error) {
	var d Definition
	b = &sifBuilder{}

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

func (b *sifBuilder) Build() (err error) {
	b.p.Provision(b.tmpfs)
	img, err := image.SIFFromSandbox(b.tmpfs, b.path)
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
