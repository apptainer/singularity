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
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/pkg/errors"
)

// supported autodetection types
const (
	TypeImageLayout = "imageLayout"
	TypeImage       = "image"
	TypeImageZip    = "imageZip"
	TypeManifest    = "manifest"
	TypeImageIndex  = "imageIndex"
	TypeConfig      = "config"
)

// Autodetect detects the validation type for the given path
// or an error if the validation type could not be resolved.
func Autodetect(path string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", errors.Wrapf(err, "unable to access path") // err from os.Stat includes path name
	}

	if fi.IsDir() {
		return TypeImageLayout, nil
	}

	f, err := os.Open(path) // nolint: errcheck, gosec
	if err != nil {
		return "", errors.Wrap(err, "unable to open file") // os.Open includes the filename
	}
	defer f.Close()

	buf, err := ioutil.ReadAll(io.LimitReader(f, 512)) // read some initial bytes to detect content
	if err != nil {
		return "", errors.Wrap(err, "unable to read")
	}

	mimeType := http.DetectContentType(buf)

	switch mimeType {
	case "application/x-gzip", "application/x-rar-compressed", "application/octet-stream":
		return TypeImage, nil
	case "application/zip":
		return TypeImageZip, nil
	}

	return "", errors.New("unknown file type")
}
