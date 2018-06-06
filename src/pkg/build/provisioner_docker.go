// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/containers/image/copy"
	"github.com/containers/image/docker"
	oci "github.com/containers/image/oci/layout"
	"github.com/containers/image/signature"
	"github.com/containers/image/types"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	imagetools "github.com/opencontainers/image-tools/image"
	"github.com/singularityware/singularity/src/pkg/image"
	"github.com/singularityware/singularity/src/pkg/sylog"
)

// NewDockerProvisioner returns a provisioner that can create a sandbox from a
// docker registry URL. The provisioner uses containers/image for retrieval
// and opencontainers/image-tools for OCI compliant extraction.
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

// DockerProvisioner returns can create a sandbox from a
// docker registry URL. The provisioner uses containers/image for retrieval
// and opencontainers/image-tools for OCI compliant extraction.
type DockerProvisioner struct {
	src       string
	srcRef    types.ImageReference
	tmpfs     string
	tmpfsRef  types.ImageReference
	policyCtx *signature.PolicyContext
}

// Provision a sandbox from a docker container reference using
// source and destination information set on the DockerProvisioner
// struct previously
func (p *DockerProvisioner) Provision(i *image.Sandbox) (err error) {
	defer os.RemoveAll(p.tmpfs)

	err = p.fetch(i)
	if err != nil {
		log.Fatal(err)
		return
	}

	imgConfig, err := p.getConfig()
	if err != nil {
		log.Fatal(err)
		return
	}

	err = p.unpackTmpfs(i)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = p.insertBaseEnv(i)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = p.insertRunScript(i, imgConfig)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = p.insertEnv(i, imgConfig)
	if err != nil {
		log.Fatal(err)
		return
	}

	return nil
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
	rootPath := path.Clean(i.Rootfs())
	if err = makeBaseEnv(rootPath); err != nil {
		sylog.Errorf("%v", err)
	}
	return
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

	if len(ociConfig.Entrypoint) > 0 {
		_, err = f.WriteString("OCI_ENTRYPOINT=\"" + strings.Join(ociConfig.Entrypoint, " ") + "\"\n")
		if err != nil {
			return
		}
	} else {
		_, err = f.WriteString("OCI_ENTRYPOINT=\"\"\n")
		if err != nil {
			return
		}
	}

	if len(ociConfig.Cmd) > 0 {
		_, err = f.WriteString("OCI_CMD=\"" + strings.Join(ociConfig.Cmd, " ") + "\"\n")
		if err != nil {
			return
		}
	} else {
		_, err = f.WriteString("OCI_CMD=\"\"\n")
		if err != nil {
			return
		}
	}

	_, err = f.WriteString(`# ENTRYPOINT only - run entrypoint plus args
if [ -z "$OCI_CMD" ] && [ -n "$OCI_ENTRYPOINT" ]; then
    SINGULARITY_OCI_RUN="${OCI_ENTRYPOINT} $@"
fi

# CMD only - run CMD or override with args
if [ -n "$OCI_CMD" ] && [ -z "$OCI_ENTRYPOINT" ]; then
    if [ $# -gt 0 ]; then
        SINGULARITY_OCI_RUN="$@"
    else
        SINGULARITY_OCI_RUN="${OCI_CMD}"
    fi
fi

# ENTRYPOINT and CMD - run ENTRYPOINT with CMD as default args
# override with user provided args
if [ $# -gt 0 ]; then
    SINGULARITY_OCI_RUN="${OCI_ENTRYPOINT} $@"
else
    SINGULARITY_OCI_RUN="${OCI_ENTRYPOINT} ${OCI_CMD}"
fi

exec $SINGULARITY_OCI_RUN

`)
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
