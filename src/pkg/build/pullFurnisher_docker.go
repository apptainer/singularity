// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/containers/image/copy"
	"github.com/containers/image/docker"
	oci "github.com/containers/image/oci/layout"
	"github.com/containers/image/signature"
	"github.com/containers/image/types"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	imagetools "github.com/opencontainers/image-tools/image"
	//"github.com/singularityware/singularity/src/pkg/image"
	"github.com/singularityware/singularity/src/pkg/sylog"
)


type DockerPuller struct {
	srcRef    types.ImageReference
	tmpfs     string
	tmpfsRef  types.ImageReference
	policyCtx *signature.PolicyContext
	imgConfig imgspecv1.ImageConfig
}

type DockerPullFurnisher struct {
	DockerPuller
}


func (p *DockerPuller) Pull(src string) (err error) {
	policy := &signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}}
	p.policyCtx, err = signature.NewPolicyContext(policy)
	if err != nil {
		return
	}

	p.srcRef, err = docker.ParseReference(src)
	if err != nil {
		return
	}

	p.tmpfs, err = ioutil.TempDir("", "temp-oci-")
	if err != nil {
		return
	}

	p.tmpfsRef, err = oci.ParseReference(p.tmpfs + ":" + "tmp")
	if err != nil {
		return
	}

	err = p.fetch()
	if err != nil {
		log.Fatal(err)
		return
	}

	p.imgConfig, err = p.getConfig()
	if err != nil {
		log.Fatal(err)
		return
	}

	return nil
}

// Furnish populates the Kitchen with relevant objects!
func (pf *DockerPullFurnisher) Furnish() (k *Kitchen, err error) {
	k, err = NewKitchen()
	if err != nil {
		return
	}

	err = pf.unpackTmpfs(k)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = pf.insertBaseEnv(k)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = pf.insertRunScript(k)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = pf.insertEnv(k)
	if err != nil {
		log.Fatal(err)
		return
	}

	return k, nil
}

func (p *DockerPuller) fetch() (err error) {
	err = copy.Image(p.policyCtx, p.tmpfsRef, p.srcRef, &copy.Options{
		ReportWriter: os.Stderr,
	})
	if err != nil {
		return err
	}

	return nil
}

func (p *DockerPuller) getConfig() (imgspecv1.ImageConfig, error) {
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

func (pf *DockerPullFurnisher) unpackTmpfs(k *Kitchen) (err error) {
	refs := []string{"name=tmp"}
	err = imagetools.UnpackLayout(pf.tmpfs, k.Rootfs(), "amd64", refs)
	return err
}

func (pf *DockerPullFurnisher) insertBaseEnv(k *Kitchen) (err error) {
	if err = makeBaseEnv(k.Rootfs()); err != nil {
		sylog.Errorf("%v", err)
	}
	return
}

func (pf *DockerPullFurnisher) insertRunScript(k *Kitchen) (err error) {
	f, err := os.Create(k.Rootfs() + "/.singularity.d/runscript")
	if err != nil {
		return
	}

	defer f.Close()

	_, err = f.WriteString("#!/bin/sh\n")
	if err != nil {
		return
	}

	if len(pf.imgConfig.Entrypoint) > 0 {
		_, err = f.WriteString("OCI_ENTRYPOINT=\"" + strings.Join(pf.imgConfig.Entrypoint, " ") + "\"\n")
		if err != nil {
			return
		}
	} else {
		_, err = f.WriteString("OCI_ENTRYPOINT=\"\"\n")
		if err != nil {
			return
		}
	}

	if len(pf.imgConfig.Cmd) > 0 {
		_, err = f.WriteString("OCI_CMD=\"" + strings.Join(pf.imgConfig.Cmd, " ") + "\"\n")
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

	err = os.Chmod(k.Rootfs()+"/.singularity.d/runscript", 0755)
	if err != nil {
		return
	}

	return nil
}

func (pf *DockerPullFurnisher) insertEnv(k *Kitchen) (err error) {
	f, err := os.Create(k.Rootfs() + "/.singularity.d/env/10-docker2singularity.sh")
	if err != nil {
		return
	}

	defer f.Close()

	_, err = f.WriteString("#!/bin/sh\n")
	if err != nil {
		return
	}

	for _, element := range pf.imgConfig.Env {
		_, err = f.WriteString("export " + element + "\n")
		if err != nil {
			return
		}

	}

	f.Sync()

	err = os.Chmod(k.Rootfs() + "/.singularity.d/env/10-docker2singularity.sh", 0755)
	if err != nil {
		return
	}

	return nil
}
