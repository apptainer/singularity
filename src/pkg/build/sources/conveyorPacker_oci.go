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
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"
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
	sytypes "github.com/singularityware/singularity/src/pkg/build/types"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/fs"
)

// OCIConveyorPacker holds stuff that needs to be packed into the bundle
type OCIConveyorPacker struct {
	srcRef    types.ImageReference
	b         *sytypes.Bundle
	tmpfsRef  types.ImageReference
	cacheRef  types.ImageReference
	policyCtx *signature.PolicyContext
	imgConfig imgspecv1.ImageConfig
}

// Get downloads container information from the specified source
func (cp *OCIConveyorPacker) Get(b *sytypes.Bundle) (err error) {

	cp.b = b

	policy := &signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}}
	cp.policyCtx, err = signature.NewPolicyContext(policy)
	if err != nil {
		return
	}

	switch b.Recipe.Header["bootstrap"] {
	case "docker":
		ref := "//" + b.Recipe.Header["from"]
		cp.srcRef, err = docker.ParseReference(ref)
	case "docker-archive":
		cp.srcRef, err = dockerarchive.ParseReference(b.Recipe.Header["from"])
	case "docker-daemon":
		cp.srcRef, err = dockerdaemon.ParseReference(b.Recipe.Header["from"])
	case "oci":
		cp.srcRef, err = oci.ParseReference(b.Recipe.Header["from"])
	case "oci-archive":
		if os.Geteuid() == 0 {
			// As root, the direct oci-archive handling will work
			cp.srcRef, err = ociarchive.ParseReference(b.Recipe.Header["from"])
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

	// Our cache dir is an OCI directory. We are using this as a 'blob pool'
	// storing all incoming containers under unique tags, which are a hash of
	// their source URI.
	tag := fmt.Sprintf("%x", sha256.Sum256([]byte(b.Recipe.Header["bootstrap"]+b.Recipe.Header["from"])))

	// Use "~/.singularity/cache/oci" which will not clash with any 2.x cache
	// directory.
	usr, err := user.Current()
	if err != nil {
		sylog.Fatalf("Couldn't determine user home directory: %v", err)
	}

	var cacheDir string
	if cacheDir = os.Getenv("SINGULARITY_CACHEDIR"); cacheDir != "" {
		cacheDir = path.Join(os.Getenv("SINGULARITY_CACHEDIR"), "oci")
	} else {
		cacheDir = path.Join(usr.HomeDir, ".singularity", "cache", "oci")
	}

	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		sylog.Debugf("Creating oci cache directory: %s", cacheDir)
		if err := fs.MkdirAll(cacheDir, 0755); err != nil {
			sylog.Fatalf("Couldn't create oci cache directory: %v", err)
		}
	}

	cp.cacheRef, err = oci.ParseReference(cacheDir + ":" + tag)
	if err != nil {
		return
	}

	// To to do the RootFS extraction we also have to have a location that
	// contains *only* this image
	cp.tmpfsRef, err = oci.ParseReference(cp.b.Path + ":" + "tmp")

	err = cp.fetch()
	if err != nil {
		log.Fatal(err)
		return
	}

	cp.imgConfig, err = cp.getConfig()
	if err != nil {
		log.Fatal(err)
		return
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
	// First we are fetching into the cache
	err = copy.Image(context.Background(), cp.policyCtx, cp.cacheRef, cp.srcRef, &copy.Options{
		ReportWriter: sylog.Writer(),
	})
	if err != nil {
		return err
	}
	// Now we have to fetch from cache into a clean, single image OCI dir
	// so that the rootfs extraction will work
	err = copy.Image(context.Background(), cp.policyCtx, cp.tmpfsRef, cp.cacheRef, &copy.Options{})
	if err != nil {
		return err
	}

	return nil
}

func (cp *OCIConveyorPacker) getConfig() (imgspecv1.ImageConfig, error) {
	img, err := cp.cacheRef.NewImage(context.Background(), nil)
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
		_, err = f.WriteString("export " + element + "\n")
		if err != nil {
			return
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
