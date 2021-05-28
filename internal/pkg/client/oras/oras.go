// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2020-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oras

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/reference"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	ocitypes "github.com/containers/image/v5/types"
	"github.com/hpcng/singularity/pkg/image"
	"github.com/hpcng/singularity/pkg/syfs"
	"github.com/hpcng/singularity/pkg/sylog"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	auth "oras.land/oras-go/pkg/auth/docker"
	"oras.land/oras-go/pkg/content"
	orasctx "oras.land/oras-go/pkg/context"
	"oras.land/oras-go/pkg/oras"
)

const (
	// SifDefaultTag is the tag to use when a tag is not specified
	SifDefaultTag = "latest"

	// SifConfigMediaTypeV1 is the config descriptor mediaType
	// Since we only ever send a null config this should not have the
	// format extension appended:
	//   https://github.com/deislabs/oras/#pushing-artifacts-with-single-files
	//   If a null config is passed, the config extension must be removed.
	SifConfigMediaTypeV1 = "application/vnd.sylabs.sif.config.v1"

	// SifLayerMediaTypeV1 is the mediaType for the "layer" which contains the actual SIF file
	SifLayerMediaTypeV1 = "application/vnd.sylabs.sif.layer.v1.sif"

	// SifLayerMediaTypeProto is the mediaType from prototyping and Singularity
	// <3.7 which unfortunately includes a typo and doesn't have a version suffix
	// See: https://github.com/hpcng/singularity/issues/4437
	SifLayerMediaTypeProto = "appliciation/vnd.sylabs.sif.layer.tar"
)

var sifLayerMediaTypes = []string{SifLayerMediaTypeV1, SifLayerMediaTypeProto}

func getResolver(ociAuth *ocitypes.DockerAuthConfig) (remotes.Resolver, error) {
	opts := docker.ResolverOptions{Credentials: genCredfn(ociAuth)}
	if ociAuth != nil && (ociAuth.Username != "" || ociAuth.Password != "") {
		return docker.NewResolver(opts), nil
	}

	cli, err := auth.NewClient(syfs.DockerConf())
	if err != nil {
		sylog.Warningf("Couldn't load auth credential file: %s", err)
		return docker.NewResolver(opts), nil
	}

	return cli.Resolver(context.Background(), &http.Client{}, false)
}

// DownloadImage downloads a SIF image specified by an oci reference to a file using the included credentials
func DownloadImage(imagePath, ref string, ociAuth *ocitypes.DockerAuthConfig) error {
	ref = strings.TrimPrefix(ref, "oras://")
	ref = strings.TrimPrefix(ref, "//")

	spec, err := reference.Parse(ref)
	if err != nil {
		return fmt.Errorf("unable to parse oci reference: %s", err)
	}

	// append default tag if no object exists
	if spec.Object == "" {
		spec.Object = SifDefaultTag
		sylog.Infof("No tag or digest found, using default: %s", SifDefaultTag)
	}

	resolver, err := getResolver(ociAuth)
	if err != nil {
		return fmt.Errorf("while getting resolver: %s", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %s", err)
	}

	store := content.NewFileStore(wd)
	defer store.Close()

	store.AllowPathTraversalOnWrite = true
	// With image caching via download to tmpfile + rename we are now overwriting the temporary file that is created
	// so we have to allow an overwrite here.
	store.DisableOverwrite = false

	allowedMediaTypes := oras.WithAllowedMediaTypes(sifLayerMediaTypes)
	handlerFunc := func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		for _, mt := range sifLayerMediaTypes {
			if desc.MediaType == mt {
				// Ensure descriptor is of a single file
				// AnnotationUnpack indicates that the descriptor is of a directory
				if desc.Annotations[content.AnnotationUnpack] == "true" {
					return nil, fmt.Errorf("descriptor is of a bundled directory, not a SIF image")
				}
				nameOld, _ := content.ResolveName(desc)
				sylog.Debugf("Will pull oras image %s to %s", nameOld, imagePath)
				_ = store.MapPath(nameOld, imagePath)
			}
		}
		return nil, nil
	}
	pullHandler := oras.WithPullBaseHandler(images.HandlerFunc(handlerFunc))

	_, _, err = oras.Pull(orasctx.Background(), resolver, spec.String(), store, allowedMediaTypes, pullHandler)
	if err != nil {
		return fmt.Errorf("unable to pull from registry: %s", err)
	}

	// ensure that we have downloaded a SIF
	if err := ensureSIF(imagePath); err != nil {
		// remove whatever we downloaded if it is not a SIF
		os.RemoveAll(imagePath)
		return err
	}

	// ensure container is executable
	if err := os.Chmod(imagePath, 0755); err != nil {
		return fmt.Errorf("unable to set image perms: %s", err)
	}

	return nil
}

// UploadImage uploads the image specified by path and pushes it to the provided oci reference,
// it will use credentials if supplied
func UploadImage(path, ref string, ociAuth *ocitypes.DockerAuthConfig) error {
	// ensure that are uploading a SIF
	if err := ensureSIF(path); err != nil {
		return err
	}

	ref = strings.TrimPrefix(ref, "oras://")
	ref = strings.TrimPrefix(ref, "//")

	spec, err := reference.Parse(ref)
	if err != nil {
		return fmt.Errorf("unable to parse oci reference: %s", err)
	}

	// Hostname() will panic if there is no '/' in the locator
	// explicitly check for this and fail in order to prevent panic
	// this case will only occur for incorrect uris
	if !strings.Contains(spec.Locator, "/") {
		return fmt.Errorf("not a valid oci object uri: %s", ref)
	}

	// append default tag if no object exists
	if spec.Object == "" {
		spec.Object = SifDefaultTag
		sylog.Infof("No tag or digest found, using default: %s", SifDefaultTag)
	}

	resolver, err := getResolver(ociAuth)
	if err != nil {
		return fmt.Errorf("while getting resolver: %s", err)
	}

	store := content.NewFileStore("")
	defer store.Close()

	conf, err := store.Add("$config", SifConfigMediaTypeV1, "/dev/null")
	if err != nil {
		return fmt.Errorf("unable to add manifest config to FileStore: %s", err)
	}
	conf.Annotations = nil

	// Get the filename from path and use it as the name in the file store
	name := filepath.Base(path)

	desc, err := store.Add(name, SifLayerMediaTypeV1, path)
	if err != nil {
		return fmt.Errorf("unable to add SIF file to FileStore: %s", err)
	}

	descriptors := []ocispec.Descriptor{desc}

	// First push with our null config of the SIF config type. This is the
	// approach given in most oras CLI and code examples and works with the
	// majority of registries.
	if _, err := oras.Push(orasctx.Background(), resolver, spec.String(), store, descriptors, oras.WithConfig(conf)); err == nil {
		return nil
	}

	// If we fail, try to push without a config at all. This will work with e.g.
	// Harbor 2.2. Unfortunately the error we get when we need to retry this way
	// isn't useful to be more specific on when this is needed.
	sylog.Debugf("ORAS push not accepted, retrying without config for registry compatibility")
	if _, err := oras.Push(orasctx.Background(), resolver, spec.String(), store, descriptors); err != nil {
		return fmt.Errorf("unable to push: %s", err)
	}

	return nil
}

// ensureSIF checks for a SIF image at filepath and returns an error if it is not, or an error is encountered
func ensureSIF(filepath string) error {
	img, err := image.Init(filepath, false)
	if err != nil {
		return fmt.Errorf("could not open image %s for verification: %s", filepath, err)
	}
	defer img.File.Close()

	if img.Type != image.SIF {
		return fmt.Errorf("%q is not a SIF", filepath)
	}

	return nil
}

// ImageSHA returns the sha256 digest of the SIF layer of the OCI manifest
// oci spec dictates only sha256 and sha512 are supported at time creation for this function
// sha512 is currently optional for implementations, this function will return an error when
// encountering such digests.
// https://github.com/opencontainers/image-spec/blob/master/descriptor.md#registered-algorithms
func ImageSHA(ctx context.Context, uri string, ociAuth *ocitypes.DockerAuthConfig) (string, error) {
	ref := strings.TrimPrefix(uri, "oras://")
	ref = strings.TrimPrefix(ref, "//")

	resolver, err := getResolver(ociAuth)
	if err != nil {
		return "", fmt.Errorf("while getting resolver: %s", err)
	}

	_, desc, err := resolver.Resolve(ctx, ref)
	if err != nil {
		return "", fmt.Errorf("while resolving reference: %v", err)
	}

	// ensure that we received an image manifest descriptor
	if desc.MediaType != ocispec.MediaTypeImageManifest {
		return "", fmt.Errorf("could not get image manifest, received mediaType: %s", desc.MediaType)
	}

	fetcher, err := resolver.Fetcher(ctx, ref)
	if err != nil {
		return "", fmt.Errorf("while creating fetcher for reference: %v", err)
	}

	rc, err := fetcher.Fetch(ctx, desc)
	if err != nil {
		return "", fmt.Errorf("while fetching manifest: %v", err)
	}
	defer rc.Close()

	b, err := ioutil.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("while reading manifest: %v", err)
	}

	var man ocispec.Manifest
	if err := json.Unmarshal(b, &man); err != nil {
		return "", fmt.Errorf("while unmarshalling manifest: %v", err)
	}

	// search image layers for sif image and return sha
	for _, l := range man.Layers {
		for _, t := range sifLayerMediaTypes {
			if l.MediaType == t {
				// only allow sha256 digests
				if l.Digest.Algorithm() != digest.SHA256 {
					return "", fmt.Errorf("SIF layer found with incorrect digest algorithm: %s", l.Digest.Algorithm())
				}
				return l.Digest.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no layer found corresponding to SIF image")
}

// ImageHash returns the appropriate hash for a provided image file
//   e.g. sha256:<sha256>
func ImageHash(filePath string) (result string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	result, _, err = sha256sum(file)

	return result, err
}

// sha256sum computes the sha256sum of the specified reader; caller is
// responsible for resetting file pointer. 'nBytes' indicates number of
// bytes read from reader
func sha256sum(r io.Reader) (result string, nBytes int64, err error) {
	hash := sha256.New()
	nBytes, err = io.Copy(hash, r)
	if err != nil {
		return "", 0, err
	}

	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nBytes, nil
}

func genCredfn(ociAuth *ocitypes.DockerAuthConfig) func(string) (string, string, error) {
	return func(_ string) (string, string, error) {
		if ociAuth != nil {
			return ociAuth.Username, ociAuth.Password, nil
		}

		return "", "", nil
	}
}
