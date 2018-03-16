/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package build

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/image/copy"
	"github.com/containers/image/docker"
	oci "github.com/containers/image/oci/layout"
	"github.com/containers/image/signature"
	"github.com/containers/image/types"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	imagetools "github.com/opencontainers/image-tools/image"
	"github.com/singularityware/singularity/pkg/image"
)

// ===== Docker =====
func NewDockerProvisioner(src string) (p *DockerProvisioner, err error) {
	p = &DockerProvisioner{}

	policy := &signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}}
	p.policyCtx, err = signature.NewPolicyContext(policy)
	if err != nil {
		return &DockerProvisioner{}, err
	}

	p.srcRef, err = docker.ParseReference(src)
	if err != nil {
		return &DockerProvisioner{}, err
	}

	p.tmpfs, err = ioutil.TempDir("", "temp-oci-")
	if err != nil {
		return &DockerProvisioner{}, err
	}

	p.tmpfsRef, err = oci.ParseReference(p.tmpfs + ":" + "tmp")
	if err != nil {
		return &DockerProvisioner{}, err
	}

	return p, nil
}

type DockerProvisioner struct {
	src       string
	srcRef    types.ImageReference
	tmpfs     string
	tmpfsRef  types.ImageReference
	policyCtx *signature.PolicyContext
}

func (p *DockerProvisioner) Provision(i *image.Sandbox) {
	defer os.RemoveAll(p.tmpfs)

	err := p.fetch(i)
	if err != nil {
		return
	}

	imgConfig, err := p.getConfig()
	if err != nil {
		return
	}

	err = p.unpackTmpfs(i)
	if err != nil {
		return
	}

	err = p.insertBaseEnv(i)
	if err != nil {
		return
	}

	err = p.insertRunScript(i, imgConfig)
	if err != nil {
		return
	}

	err = p.insertEnv(i, imgConfig)
	if err != nil {
		return
	}

}

func (p *DockerProvisioner) fetch(i *image.Sandbox) (err error) {
	err = copy.Image(p.policyCtx, p.tmpfsRef, p.srcRef, &copy.Options{
		ReportWriter: os.Stderr,
	})
	if err != nil {
		return err
	}

	return nil
}

func (p *DockerProvisioner) getConfig() (imgspecv1.ImageConfig, error) {
	img, err := p.tmpfsRef.NewImage(nil)
	if err != nil {
		return imgspecv1.ImageConfig{}, err
	}
	defer img.Close()

	imgSpec, err := img.OCIConfig()
	if err != nil {
		return imgspecv1.ImageConfig{}, err
	}

	return imgSpec.Config, nil
}

func (p *DockerProvisioner) unpackTmpfs(i *image.Sandbox) (err error) {
	refs := []string{"name=tmp"}
	err = imagetools.UnpackLayout(p.tmpfs, i.Rootfs(), "amd64", refs)
	return err
}

func (p *DockerProvisioner) insertBaseEnv(i *image.Sandbox) (err error) {
	f, err := os.Open("environment.tar")
	if err != nil {
		return
	}

	defer f.Close()

	err = Untar(i.Rootfs(), f)

	return nil
}

func (p *DockerProvisioner) insertRunScript(i *image.Sandbox, ociConfig imgspecv1.ImageConfig) (err error) {
	f, err := os.Create(i.Rootfs() + "/.singularity.d/runscript")
	if err != nil {
		return
	}

	defer f.Close()

	_, err = f.WriteString("#!/bin/sh\n")
	if err != nil {
		return
	}

	_, err = f.WriteString(strings.Join(ociConfig.Entrypoint, " "))
	if err != nil {
		return
	}

	_, err = f.WriteString(" ")
	if err != nil {
		return
	}

	_, err = f.WriteString(strings.Join(ociConfig.Cmd, " "))
	if err != nil {
		return
	}

	_, err = f.WriteString("\n")
	if err != nil {
		return
	}

	f.Sync()

	err = os.Chmod(i.Rootfs()+"/.singularity.d/runscript", 0755)
	if err != nil {
		return
	}

	return nil
}

func (p *DockerProvisioner) insertEnv(i *image.Sandbox, ociConfig imgspecv1.ImageConfig) (err error) {
	f, err := os.Create(i.Rootfs() + "/.singularity.d/env/10-docker2singularity.sh")
	if err != nil {
		return
	}

	defer f.Close()

	_, err = f.WriteString("#!/bin/sh\n")
	if err != nil {
		return
	}

	for _, element := range ociConfig.Env {
		_, err = f.WriteString("export " + element + "\n")
		if err != nil {
			return
		}

	}

	f.Sync()

	err = os.Chmod(i.Rootfs()+"/.singularity.d/env/10-docker2singularity.sh", 0755)
	if err != nil {
		return
	}

	return nil
}

//
// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func Untar(dst string, r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	defer gzr.Close()
	if err != nil {
		return err
	}

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer f.Close()

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
		}
	}
}
