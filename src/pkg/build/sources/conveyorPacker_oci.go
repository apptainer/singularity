// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/image/copy"
	"github.com/containers/image/docker"
	dockerarchive "github.com/containers/image/docker/archive"
	dockerdaemon "github.com/containers/image/docker/daemon"
	ociarchive "github.com/containers/image/oci/archive"
	oci "github.com/containers/image/oci/layout"
	"github.com/containers/image/signature"
	"github.com/containers/image/types"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	imagetools "github.com/opencontainers/image-tools/image"
	//"github.com/singularityware/singularity/src/pkg/image"
	"github.com/singularityware/singularity/src/pkg/sylog"
)

// OCIConveyor holds stuff that needs to be packed into the bundle
type OCIConveyor struct {
	recipe    Definition
	srcRef    types.ImageReference
	tmpfs     string
	tmpfsRef  types.ImageReference
	policyCtx *signature.PolicyContext
	imgConfig imgspecv1.ImageConfig
}

// OCIConveyorPacker only needs to hold the conveyor to have the needed data to pack
type OCIConveyorPacker struct {
	OCIConveyor
}

// Get downloads container information from the specified source
func (c *OCIConveyor) Get(recipe Definition) (err error) {
	policy := &signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}}
	c.policyCtx, err = signature.NewPolicyContext(policy)
	if err != nil {
		return
	}

	c.recipe = recipe

	switch recipe.Header["bootstrap"] {
	case "docker":
		ref := "//" + recipe.Header["from"]
		c.srcRef, err = docker.ParseReference(ref)
	case "docker-archive":
		c.srcRef, err = dockerarchive.ParseReference(recipe.Header["from"])
	case "docker-daemon":
		c.srcRef, err = dockerdaemon.ParseReference(recipe.Header["from"])
	case "oci":
		c.srcRef, err = oci.ParseReference(recipe.Header["from"])
	case "oci-archive":
		if os.Geteuid() == 0 {
			// As root, the direct oci-archive handling will work
			c.srcRef, err = ociarchive.ParseReference(recipe.Header["from"])
		} else {
			// As non-root we need to do a dumb tar extraction first
			tmpDir, err := ioutil.TempDir("", "temp-oci-")
			if err != nil {
				return fmt.Errorf("could not create temporary oci directory: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			refParts := strings.SplitN(recipe.Header["from"], ":", 2)
			err = c.extractArchive(refParts[0], tmpDir)
			if err != nil {
				return fmt.Errorf("error extracting the OCI archive file: %v", err)
			}
			// We may or may not have had a ':tag' in the source to handle
			if len(refParts) == 2 {
				c.srcRef, err = oci.ParseReference(tmpDir + ":" + refParts[1])
			} else {
				c.srcRef, err = oci.ParseReference(tmpDir)
			}
		}

	default:
		return fmt.Errorf("OCI ConveyerPacker does not support %s", recipe.Header["bootstrap"])
	}

	if err != nil {
		return fmt.Errorf("Invalid image source: %v", err)
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
func (cp *OCIConveyorPacker) Pack() (b *Bundle, err error) {
	b, err = NewBundle(cp.tmpfs)
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

	b.Recipe = cp.recipe

	return b, nil
}

func (c *OCIConveyor) fetch() (err error) {
	err = copy.Image(context.Background(), c.policyCtx, c.tmpfsRef, c.srcRef, &copy.Options{
		ReportWriter: os.Stderr,
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *OCIConveyor) getConfig() (imgspecv1.ImageConfig, error) {
	img, err := c.tmpfsRef.NewImage(context.Background(), nil)
	if err != nil {
		return imgspecv1.ImageConfig{}, err
	}
	defer img.Close()

	imgSpec, err := img.OCIConfig(context.Background())
	if err != nil {
		return imgspecv1.ImageConfig{}, err
	}

	return imgSpec.Config, nil
}

// Perform a dumb tar(gz) extraction with no chown, id remapping etc.
// This is needed for non-root handling of `oci-archive` as the extraction
// by containers/archive is failing when uid/gid don't match local machine
// and we're not root
func (c *OCIConveyor) extractArchive(src string, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	header, err := r.Peek(10) //read a few bytes without consuming
	if err != nil {
		return err
	}
	gzipped := strings.Contains(http.DetectContentType(header), "x-gzip")

	if gzipped {
		r, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer r.Close()
	}

	tr := tar.NewReader(r)

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

		// ZipSlip protection - don't escape from dst
		target := filepath.Join(dst, header.Name)
		if !strings.HasPrefix(target, filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("%s: illegal extraction path", target)
		}

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

func (cp *OCIConveyorPacker) unpackTmpfs(b *Bundle) (err error) {
	refs := []string{"name=tmp"}
	err = imagetools.UnpackLayout(cp.tmpfs, b.Rootfs(), "amd64", refs)
	return err
}

func (cp *OCIConveyorPacker) insertBaseEnv(b *Bundle) (err error) {
	if err = makeBaseEnv(b.Rootfs()); err != nil {
		sylog.Errorf("%v", err)
	}
	return
}

func (cp *OCIConveyorPacker) insertRunScript(b *Bundle) (err error) {
	f, err := os.Create(b.Rootfs() + "/.singularity.d/runscript")
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

	err = os.Chmod(b.Rootfs()+"/.singularity.d/runscript", 0755)
	if err != nil {
		return
	}

	return nil
}

func (cp *OCIConveyorPacker) insertEnv(b *Bundle) (err error) {
	f, err := os.Create(b.Rootfs() + "/.singularity.d/env/10-docker2singularity.sh")
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

	err = os.Chmod(b.Rootfs()+"/.singularity.d/env/10-docker2singularity.sh", 0755)
	if err != nil {
		return
	}

	return nil
}
