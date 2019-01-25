// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
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
	ociclient "github.com/sylabs/singularity/internal/pkg/client/oci"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/shell"
	sytypes "github.com/sylabs/singularity/pkg/build/types"
)

// OCIConveyorPacker holds stuff that needs to be packed into the bundle
type OCIConveyorPacker struct {
	srcRef    types.ImageReference
	b         *sytypes.Bundle
	tmpfsRef  types.ImageReference
	policyCtx *signature.PolicyContext
	imgConfig imgspecv1.ImageConfig
	sysCtx    *types.SystemContext
}

// Get downloads container information from the specified source
func (cp *OCIConveyorPacker) Get(b *sytypes.Bundle) (err error) {

	cp.b = b

	policy := &signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}}
	cp.policyCtx, err = signature.NewPolicyContext(policy)
	if err != nil {
		return err
	}

	cp.sysCtx = &types.SystemContext{
		OCIInsecureSkipTLSVerify:    cp.b.Opts.NoHTTPS,
		DockerInsecureSkipTLSVerify: cp.b.Opts.NoHTTPS,
		DockerAuthConfig:            cp.b.Opts.DockerAuthConfig,
		OSChoice:                    "linux",
	}

	// add registry and namespace to reference if specified
	ref := b.Recipe.Header["from"]
	if b.Recipe.Header["namespace"] != "" {
		ref = b.Recipe.Header["namespace"] + "/" + ref
	}
	if b.Recipe.Header["registry"] != "" {
		ref = b.Recipe.Header["registry"] + "/" + ref
	}
	sylog.Debugf("Reference: %v", ref)

	switch b.Recipe.Header["bootstrap"] {
	case "docker":
		ref = "//" + ref
		cp.srcRef, err = docker.ParseReference(ref)
	case "docker-archive":
		cp.srcRef, err = dockerarchive.ParseReference(ref)
	case "docker-daemon":
		cp.srcRef, err = dockerdaemon.ParseReference(ref)
	case "oci":
		cp.srcRef, err = oci.ParseReference(ref)
	case "oci-archive":
		if os.Geteuid() == 0 {
			// As root, the direct oci-archive handling will work
			cp.srcRef, err = ociarchive.ParseReference(ref)
		} else {
			// As non-root we need to do a dumb tar extraction first
			tmpDir, err := ioutil.TempDir("", "temp-oci-")
			if err != nil {
				return fmt.Errorf("could not create temporary oci directory: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			refParts := strings.SplitN(b.Recipe.Header["from"], ":", 2)
			err = cp.extractArchive(refParts[0], tmpDir)
			if err != nil {
				return fmt.Errorf("error extracting the OCI archive file: %v", err)
			}
			// We may or may not have had a ':tag' in the source to handle
			if len(refParts) == 2 {
				cp.srcRef, err = oci.ParseReference(tmpDir + ":" + refParts[1])
			} else {
				cp.srcRef, err = oci.ParseReference(tmpDir)
			}
		}

	default:
		return fmt.Errorf("OCI ConveyorPacker does not support %s", b.Recipe.Header["bootstrap"])
	}

	if err != nil {
		return fmt.Errorf("Invalid image source: %v", err)
	}

	// Grab the modified source ref from the cache
	cp.srcRef, err = ociclient.ConvertReference(cp.srcRef, cp.sysCtx)
	if err != nil {
		return err
	}

	// To to do the RootFS extraction we also have to have a location that
	// contains *only* this image
	cp.tmpfsRef, err = oci.ParseReference(cp.b.Path + ":" + "tmp")

	err = cp.fetch()
	if err != nil {
		return err
	}

	cp.imgConfig, err = cp.getConfig()
	if err != nil {
		return err
	}

	return nil
}

// Pack puts relevant objects in a Bundle!
func (cp *OCIConveyorPacker) Pack() (*sytypes.Bundle, error) {
	err := cp.unpackTmpfs()
	if err != nil {
		return nil, fmt.Errorf("While unpacking tmpfs: %v", err)
	}

	err = cp.insertBaseEnv()
	if err != nil {
		return nil, fmt.Errorf("While inserting base environment: %v", err)
	}

	err = cp.insertRunScript()
	if err != nil {
		return nil, fmt.Errorf("While inserting runscript: %v", err)
	}

	err = cp.insertEnv()
	if err != nil {
		return nil, fmt.Errorf("While inserting docker specific environment: %v", err)
	}

	return cp.b, nil
}

func (cp *OCIConveyorPacker) fetch() (err error) {
	// cp.srcRef contains the cache source reference
	err = copy.Image(context.Background(), cp.policyCtx, cp.tmpfsRef, cp.srcRef, &copy.Options{
		ReportWriter: ioutil.Discard,
		SourceCtx:    cp.sysCtx,
	})
	if err != nil {
		return err
	}

	return nil
}

func (cp *OCIConveyorPacker) getConfig() (imgspecv1.ImageConfig, error) {
	img, err := cp.srcRef.NewImage(context.Background(), cp.sysCtx)
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
func (cp *OCIConveyorPacker) extractArchive(src string, dst string) error {
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

func (cp *OCIConveyorPacker) unpackTmpfs() (err error) {
	refs := []string{"name=tmp"}
	err = imagetools.UnpackLayout(cp.b.Path, cp.b.Rootfs(), "amd64", refs)
	return err
}

func (cp *OCIConveyorPacker) insertBaseEnv() (err error) {
	if err = makeBaseEnv(cp.b.Rootfs()); err != nil {
		sylog.Errorf("%v", err)
	}
	return
}

func (cp *OCIConveyorPacker) insertRunScript() (err error) {
	f, err := os.Create(cp.b.Rootfs() + "/.singularity.d/runscript")
	if err != nil {
		return
	}

	defer f.Close()

	_, err = f.WriteString("#!/bin/sh\n")
	if err != nil {
		return
	}

	if len(cp.imgConfig.Entrypoint) > 0 {
		_, err = f.WriteString("OCI_ENTRYPOINT='" + shell.ArgsQuoted(cp.imgConfig.Entrypoint) + "'\n")
		if err != nil {
			return
		}
	} else {
		_, err = f.WriteString("OCI_ENTRYPOINT=''\n")
		if err != nil {
			return
		}
	}

	if len(cp.imgConfig.Cmd) > 0 {
		_, err = f.WriteString("OCI_CMD='" + shell.ArgsQuoted(cp.imgConfig.Cmd) + "'\n")
		if err != nil {
			return
		}
	} else {
		_, err = f.WriteString("OCI_CMD=''\n")
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

# Evaluate shell expressions first and set arguments accordingly,
# then execute final command as first container process
eval "set ${SINGULARITY_OCI_RUN}"
exec "$@"

`)
	if err != nil {
		return
	}

	f.Sync()

	err = os.Chmod(cp.b.Rootfs()+"/.singularity.d/runscript", 0755)
	if err != nil {
		return
	}

	return nil
}

func (cp *OCIConveyorPacker) insertEnv() (err error) {
	f, err := os.Create(cp.b.Rootfs() + "/.singularity.d/env/10-docker2singularity.sh")
	if err != nil {
		return
	}

	defer f.Close()

	_, err = f.WriteString("#!/bin/sh\n")
	if err != nil {
		return
	}

	for _, element := range cp.imgConfig.Env {

		envParts := strings.SplitN(element, "=", 2)
		if len(envParts) == 1 {
			_, err = f.WriteString("export " + shell.Escape(element) + "\n")
			if err != nil {
				return
			}
		} else {
			_, err = f.WriteString("export " + envParts[0] + "=\"" + shell.Escape(envParts[1]) + "\"\n")
			if err != nil {
				return
			}
		}
	}

	f.Sync()

	err = os.Chmod(cp.b.Rootfs()+"/.singularity.d/env/10-docker2singularity.sh", 0755)
	if err != nil {
		return
	}

	return nil
}

// CleanUp removes any tmpfs owned by the conveyorPacker on the filesystem
func (cp *OCIConveyorPacker) CleanUp() {
	os.RemoveAll(cp.b.Path)
}
