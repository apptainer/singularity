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
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

const indexPath = "index.json"

func listReferences(w walker) ([]v1.Descriptor, error) {
	var descs []v1.Descriptor
	var index v1.Index

	if err := w.walk(func(path string, info os.FileInfo, r io.Reader) error {
		if info.IsDir() || filepath.Clean(path) != indexPath {
			return nil
		}

		if err := json.NewDecoder(r).Decode(&index); err != nil {
			return err
		}
		descs = index.Manifests

		return nil
	}); err != nil {
		return nil, err
	}

	return descs, nil
}

func findDescriptor(w walker, names []string) ([]v1.Descriptor, error) {
	var descs []v1.Descriptor
	var index v1.Index
	dpath := "index.json"

	if err := w.find(dpath, func(path string, r io.Reader) error {
		if err := json.NewDecoder(r).Decode(&index); err != nil {
			return err
		}

		descs = index.Manifests
		for _, name := range names {
			argsParts := strings.Split(name, "=")
			if len(argsParts) != 2 {
				return fmt.Errorf("each ref must contain two parts")
			}

			switch argsParts[0] {
			case "name":
				for i := 0; i < len(descs); i++ {
					if descs[i].Annotations[v1.AnnotationRefName] != argsParts[1] {
						descs = append(descs[:i], descs[i+1:]...)
					}
				}
			case "platform.os":
				for i := 0; i < len(descs); i++ {
					if descs[i].Platform != nil && index.Manifests[i].Platform.OS != argsParts[1] {
						descs = append(descs[:i], descs[i+1:]...)
					}
				}
			case "digest":
				for i := 0; i < len(descs); i++ {
					if string(descs[i].Digest) != argsParts[1] {
						descs = append(descs[:i], descs[i+1:]...)
					}
				}
			default:
				return fmt.Errorf("criteria %q unimplemented", argsParts[0])
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	if len(descs) == 0 {
		return nil, fmt.Errorf("index.json: descriptor retrieved by refs %v is not match", names)
	} else if len(descs) > 1 {
		return nil, fmt.Errorf("index.json: descriptor retrieved by refs %v is not unique", names)
	}

	return descs, nil
}

func validateDescriptor(d *v1.Descriptor, w walker, mts []string) error {
	var found bool
	for _, mt := range mts {
		if d.MediaType == mt {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("invalid descriptor MediaType %q", d.MediaType)
	}

	if err := d.Digest.Validate(); err != nil {
		return err
	}

	// Copy the contents of the layer in to the verifier
	verifier := d.Digest.Verifier()
	numBytes, err := w.get(*d, verifier)
	if err != nil {
		return err
	}

	if err != nil {
		return errors.Wrap(err, "error generating hash")
	}

	if numBytes != d.Size {
		return errors.New("size mismatch")
	}

	if !verifier.Verified() {
		return errors.New("digest mismatch")
	}

	return nil
}
