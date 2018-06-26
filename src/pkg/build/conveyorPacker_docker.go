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
	"github.com/singularityware/singularity/src/pkg/sylog"
)

// DockerConveyor holds stuff that needs to be packed into the bundle
type DockerConveyor struct {
	recipe    Definition
	srcRef    types.ImageReference
	tmpfs     string
	tmpfsRef  types.ImageReference
	policyCtx *signature.PolicyContext
	imgConfig imgspecv1.ImageConfig
}

// DockerConveyorPacker only needs to hold the conveyor to have the needed data to pack
type DockerConveyorPacker struct {
	DockerConveyor
}

// Get downloads container information from Dockerhub
func (c *DockerConveyor) Get(recipe Definition) (err error) {
	policy := &signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}}
	c.policyCtx, err = signature.NewPolicyContext(policy)
	if err != nil {
		return
	}

	c.recipe = recipe

	//prepending slashes to src for ParseReference expected string format
	src := "//" + recipe.Header["from"]
	c.srcRef, err = docker.ParseReference(src)
	if err != nil {
		return
	}

	c.tmpfs, err = ioutil.TempDir("", "temp-oci-")
	if err != nil {
		return
	}

	c.tmpfsRef, err = oci.ParseReference(c.tmpfs + ":" + "tmp")
	if err != nil {
		return
	}

	err = c.fetch()
	if err != nil {
		log.Fatal(err)
		return
	}

	c.imgConfig, err = c.getConfig()
	if err != nil {
		log.Fatal(err)
		return
	}

	return nil
}

// Pack puts relevant objects in a Bundle!
func (cp *DockerConveyorPacker) Pack() (b *Bundle, err error) {
	b, err = NewBundle()
	if err != nil {
		return
	}

	err = cp.unpackTmpfs(b)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = cp.insertBaseEnv(b)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = cp.insertRunScript(b)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = cp.insertEnv(b)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = cp.setBindPoints(b)
	if err != nil {
		log.Fatal(err)
		return
	}

	b.Recipe = cp.recipe

	return b, nil
}

func (c *DockerConveyor) fetch() (err error) {
	err = copy.Image(c.policyCtx, c.tmpfsRef, c.srcRef, &copy.Options{
		ReportWriter: os.Stderr,
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *DockerConveyor) getConfig() (imgspecv1.ImageConfig, error) {
	img, err := c.tmpfsRef.NewImage(nil)
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

func (cp *DockerConveyorPacker) unpackTmpfs(b *Bundle) (err error) {
	refs := []string{"name=tmp"}
	err = imagetools.UnpackLayout(cp.tmpfs, b.Rootfs(), "amd64", refs)
	return err
}

func (cp *DockerConveyorPacker) insertBaseEnv(b *Bundle) (err error) {
	if err = makeBaseEnv(b.Rootfs(), b.Path+"/"+b.FSObjects[".singularity.d"]); err != nil {
		sylog.Errorf("%v", err)
	}
	return
}

func (cp *DockerConveyorPacker) insertRunScript(b *Bundle) (err error) {
	f, err := os.Create(b.Path + "/" + b.FSObjects[".singularity.d"] + "/runscript")
	if err != nil {
		return
	}

	defer f.Close()

	_, err = f.WriteString("#!/bin/sh\n")
	if err != nil {
		return
	}

	if len(cp.imgConfig.Entrypoint) > 0 {
		_, err = f.WriteString("OCI_ENTRYPOINT=\"" + strings.Join(cp.imgConfig.Entrypoint, " ") + "\"\n")
		if err != nil {
			return
		}
	} else {
		_, err = f.WriteString("OCI_ENTRYPOINT=\"\"\n")
		if err != nil {
			return
		}
	}

	if len(cp.imgConfig.Cmd) > 0 {
		_, err = f.WriteString("OCI_CMD=\"" + strings.Join(cp.imgConfig.Cmd, " ") + "\"\n")
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

	err = os.Chmod(b.Path+"/"+b.FSObjects[".singularity.d"]+"/runscript", 0755)
	if err != nil {
		return
	}

	return nil
}

func (cp *DockerConveyorPacker) insertEnv(b *Bundle) (err error) {
	f, err := os.Create(b.Path + "/" + b.FSObjects[".singularity.d"] + "/env/10-docker2singularity.sh")
	if err != nil {
		return
	}

	defer f.Close()

	_, err = f.WriteString("#!/bin/sh\n")
	if err != nil {
		return
	}

	for _, element := range cp.imgConfig.Env {
		_, err = f.WriteString("export " + element + "\n")
		if err != nil {
			return
		}

	}

	f.Sync()

	err = os.Chmod(b.Path+"/"+b.FSObjects[".singularity.d"]+"/env/10-docker2singularity.sh", 0755)
	if err != nil {
		return
	}

	return nil
}

func (cp *DockerConveyorPacker) setBindPoints(b *Bundle) (err error) {
	//add bind points for engine
	b.BindPath = append(b.BindPath, b.Rootfs()+":/")

	return
}
