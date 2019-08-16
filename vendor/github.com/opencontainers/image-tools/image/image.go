// Copyright 2016 The Linux Foundation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package image

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

// ValidateLayout walks through the given file tree and validates the manifest
// pointed to by the given refs or returns an error if the validation failed.
func ValidateLayout(src string, refs []string, out *log.Logger) error {
	return validate(newPathWalker(src), refs, out)
}

// ValidateZip walks through the given file tree and validates the manifest
// pointed to by the given refs or returns an error if the validation failed.
func ValidateZip(src string, refs []string, out *log.Logger) error {
	return validate(newZipWalker(src), refs, out)
}

// ValidateFile opens the tar file given by the filename, then calls ValidateReader
func ValidateFile(tarFile string, refs []string, out *log.Logger) error {
	f, err := os.Open(tarFile) // nolint: errcheck, gosec
	if err != nil {
		return errors.Wrap(err, "unable to open file")
	}
	defer f.Close()

	return Validate(f, refs, out)
}

// Validate walks through a tar stream and validates the manifest.
// * Check that all refs point to extant blobs
// * Checks that all referred blobs are valid
// * Checks that mime-types are correct
// returns error on validation failure
func Validate(r io.ReadSeeker, refs []string, out *log.Logger) error {
	return validate(newTarWalker(r), refs, out)
}

var validRefMediaTypes = []string{
	v1.MediaTypeImageManifest,
	v1.MediaTypeImageIndex,
}

func validate(w walker, refs []string, out *log.Logger) error {
	var descs []v1.Descriptor
	var err error

	if err = layoutValidate(w); err != nil {
		return err
	}

	if len(refs) == 0 {
		out.Print("No ref specified, verify all refs")
		descs, err = listReferences(w)
		if err != nil {
			return err
		}
		if len(descs) == 0 {
			// TODO(runcom): ugly, we'll need a better way and library
			// to express log levels.
			// see https://github.com/opencontainers/image-spec/issues/288
			out.Print("WARNING: no descriptors found")
			return nil
		}
	} else {
		descs, err = findDescriptor(w, refs)
		if err != nil {
			return err
		}
	}

	for _, desc := range descs {
		d := &desc
		if err = validateDescriptor(d, w, validRefMediaTypes); err != nil {
			return err
		}

		if d.MediaType == validRefMediaTypes[0] {
			m, err := findManifest(w, d)
			if err != nil {
				return err
			}

			if err := validateManifest(m, w); err != nil {
				return err
			}
		}

		if d.MediaType == validRefMediaTypes[1] {
			index, err := findIndex(w, d)
			if err != nil {
				return err
			}

			if err := validateIndex(index, w); err != nil {
				return err
			}

			if len(index.Manifests) == 0 {
				fmt.Println("warning: no manifests found")
				return nil
			}

			for _, manifest := range index.Manifests {
				m, err := findManifest(w, &manifest)
				if err != nil {
					return err
				}

				if err := validateManifest(m, w); err != nil {
					return err
				}
			}
		}
	}

	if out != nil && len(refs) > 0 {
		out.Printf("reference %v: OK", refs)
	}

	return nil
}

// UnpackLayout walks through the file tree given by src and, using the layers
// specified in the manifest pointed to by the given ref, unpacks all layers in
// the given destination directory or returns an error if the unpacking failed.
func UnpackLayout(src, dest, platform string, refs []string) error {
	return unpack(newPathWalker(src), dest, platform, refs)
}

// UnpackZip opens and walks through the zip file given by src and, using the layers
// specified in the manifest pointed to by the given ref, unpacks all layers in
// the given destination directory or returns an error if the unpacking failed.
func UnpackZip(src, dest, platform string, refs []string) error {
	return unpack(newZipWalker(src), dest, platform, refs)
}

// UnpackFile opens the file pointed by tarFileName and calls Unpack on it.
func UnpackFile(tarFileName, dest, platform string, refs []string) error {
	f, err := os.Open(tarFileName) // nolint: errcheck, gosec
	if err != nil {
		return errors.Wrap(err, "unable to open file")
	}
	defer f.Close()

	return Unpack(f, dest, platform, refs)
}

// Unpack walks through the tar stream and, using the layers specified in
// the manifest pointed to by the given ref, unpacks all layers in the given
// destination directory or returns an error if the unpacking failed.
// The destination will be created if it does not exist.
func Unpack(r io.ReadSeeker, dest, platform string, refs []string) error {
	return unpack(newTarWalker(r), dest, platform, refs)
}

func unpack(w walker, dest, platform string, refs []string) error {
	if err := layoutValidate(w); err != nil {
		return err
	}

	descs, err := findDescriptor(w, refs)
	if err != nil {
		return err
	}

	ref := &descs[0]
	if err = validateDescriptor(ref, w, validRefMediaTypes); err != nil {
		return err
	}

	if ref.MediaType == validRefMediaTypes[0] {
		m, err := findManifest(w, ref)
		if err != nil {
			return err
		}

		if err := validateManifest(m, w); err != nil {
			return err
		}

		return unpackManifest(m, w, dest)
	}

	if ref.MediaType == validRefMediaTypes[1] {
		index, err := findIndex(w, ref)
		if err != nil {
			return err
		}

		if err = validateIndex(index, w); err != nil {
			return err
		}

		manifests, err := filterManifest(w, index.Manifests, platform)
		if err != nil {
			return err
		}

		for _, m := range manifests {
			return unpackManifest(m, w, dest)
		}
	}

	return nil
}

// CreateRuntimeBundleLayout walks through the file tree given by src and
// creates an OCI runtime bundle in the given destination dest
// or returns an error if the unpacking failed.
func CreateRuntimeBundleLayout(src, dest, root, platform string, refs []string) error {
	return createRuntimeBundle(newPathWalker(src), dest, root, platform, refs)
}

// CreateRuntimeBundleZip opens and walks through the zip file given by src
// and creates an OCI runtime bundle in the given destination dest
// or returns an error if the unpacking failed.
func CreateRuntimeBundleZip(src, dest, root, platform string, refs []string) error {
	return createRuntimeBundle(newZipWalker(src), dest, root, platform, refs)
}

// CreateRuntimeBundleFile opens the file pointed by tarFile and calls
// CreateRuntimeBundle.
func CreateRuntimeBundleFile(tarFile, dest, root, platform string, refs []string) error {
	f, err := os.Open(tarFile) // nolint: errcheck, gosec
	if err != nil {
		return errors.Wrap(err, "unable to open file")
	}
	defer f.Close()

	return createRuntimeBundle(newTarWalker(f), dest, root, platform, refs)
}

// CreateRuntimeBundle walks through the given tar stream and
// creates an OCI runtime bundle in the given destination dest
// or returns an error if the unpacking failed.
func CreateRuntimeBundle(r io.ReadSeeker, dest, root, platform string, refs []string) error {
	return createRuntimeBundle(newTarWalker(r), dest, root, platform, refs)
}

func createRuntimeBundle(w walker, dest, rootfs, platform string, refs []string) error {
	if err := layoutValidate(w); err != nil {
		return err
	}

	descs, err := findDescriptor(w, refs)
	if err != nil {
		return err
	}

	ref := &descs[0]
	if err = validateDescriptor(ref, w, validRefMediaTypes); err != nil {
		return err
	}

	if ref.MediaType == validRefMediaTypes[0] {
		m, err := findManifest(w, ref)
		if err != nil {
			return err
		}

		if err := validateManifest(m, w); err != nil {
			return err
		}

		return createBundle(w, m, dest, rootfs)
	}

	if ref.MediaType == validRefMediaTypes[1] {
		index, err := findIndex(w, ref)
		if err != nil {
			return err
		}

		if err = validateIndex(index, w); err != nil {
			return err
		}

		manifests, err := filterManifest(w, index.Manifests, platform)
		if err != nil {
			return err
		}

		for _, m := range manifests {
			return createBundle(w, m, dest, rootfs)
		}
	}

	return nil
}

func createBundle(w walker, m *v1.Manifest, dest, rootfs string) (retErr error) {
	c, err := findConfig(w, &m.Config)
	if err != nil {
		return err
	}

	if _, err = os.Stat(dest); err != nil {
		if os.IsNotExist(err) {
			if err2 := os.MkdirAll(dest, 0750); err2 != nil {
				return err2
			}
			defer func() {
				if retErr != nil {
					if err3 := os.RemoveAll(dest); err3 != nil {
						fmt.Printf("Failed to clean up %q: %s\n", dest, err3.Error())
					}
				}
			}()
		} else {
			return err
		}
	}

	if err = unpackManifest(m, w, filepath.Join(dest, rootfs)); err != nil {
		return err
	}

	spec, err := runtimeSpec(c, rootfs)
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(dest, "config.json"))
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(spec)
}

// filertManifest returns a filtered list of manifests
func filterManifest(w walker, Manifests []v1.Descriptor, platform string) ([]*v1.Manifest, error) {
	var manifests []*v1.Manifest

	argsParts := strings.Split(platform, ":")
	if len(argsParts) != 2 {
		return manifests, fmt.Errorf("platform must have os and arch when reftype is index")
	}

	if len(Manifests) == 0 {
		fmt.Println("warning: no manifests found")
		return manifests, nil
	}

	for _, manifest := range Manifests {
		m, err := findManifest(w, &manifest)
		if err != nil {
			return manifests, err
		}

		if err := validateManifest(m, w); err != nil {
			return manifests, err
		}
		if strings.EqualFold(manifest.Platform.OS, argsParts[0]) && strings.EqualFold(manifest.Platform.Architecture, argsParts[1]) {
			manifests = append(manifests, m)
		}
	}

	if len(manifests) == 0 {
		return manifests, fmt.Errorf("there is no matching manifest")
	}

	return manifests, nil
}
