/*
 * umoci: Umoci Modifies Open Containers' Images
 * Copyright (C) 2016, 2017, 2018 SUSE LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dir

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/openSUSE/umoci/oci/cas"
	"github.com/opencontainers/go-digest"
	imeta "github.com/opencontainers/image-spec/specs-go"
	ispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/sys/unix"
)

const (
	// ImageLayoutVersion is the version of the image layout we support. This
	// value is *not* the same as imagespec.Version, and the meaning of this
	// field is still under discussion in the spec. For now we'll just hardcode
	// the value and hope for the best.
	ImageLayoutVersion = "1.0.0"

	// blobDirectory is the directory inside an OCI image that contains blobs.
	blobDirectory = "blobs"

	// indexFile is the file inside an OCI image that contains the top-level
	// index.
	indexFile = "index.json"

	// layoutFile is the file in side an OCI image the indicates what version
	// of the OCI spec the image is.
	layoutFile = "oci-layout"
)

// blobPath returns the path to a blob given its digest, relative to the root
// of the OCI image. The digest must be of the form algorithm:hex.
func blobPath(digest digest.Digest) (string, error) {
	if err := digest.Validate(); err != nil {
		return "", errors.Wrapf(err, "invalid digest: %q", digest)
	}

	algo := digest.Algorithm()
	hash := digest.Hex()

	if algo != cas.BlobAlgorithm {
		return "", errors.Errorf("unsupported algorithm: %q", algo)
	}

	return filepath.Join(blobDirectory, algo.String(), hash), nil
}

type dirEngine struct {
	path     string
	temp     string
	tempFile *os.File
}

func (e *dirEngine) ensureTempDir() error {
	if e.temp == "" {
		tempDir, err := ioutil.TempDir(e.path, ".umoci-")
		if err != nil {
			return errors.Wrap(err, "create tempdir")
		}

		// We get an advisory lock to ensure that GC() won't delete our
		// temporary directory here. Once we get the lock we know it won't do
		// anything until we unlock it or exit.

		e.tempFile, err = os.Open(tempDir)
		if err != nil {
			return errors.Wrap(err, "open tempdir for lock")
		}
		if err := unix.Flock(int(e.tempFile.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
			return errors.Wrap(err, "lock tempdir")
		}

		e.temp = tempDir
	}
	return nil
}

// verify ensures that the image is valid.
func (e *dirEngine) validate() error {
	content, err := ioutil.ReadFile(filepath.Join(e.path, layoutFile))
	if err != nil {
		if os.IsNotExist(err) {
			err = cas.ErrInvalid
		}
		return errors.Wrap(err, "read oci-layout")
	}

	var ociLayout ispec.ImageLayout
	if err := json.Unmarshal(content, &ociLayout); err != nil {
		return errors.Wrap(err, "parse oci-layout")
	}

	// XXX: Currently the meaning of this field is not adequately defined by
	//      the spec, nor is the "official" value determined by the spec.
	if ociLayout.Version != ImageLayoutVersion {
		return errors.Wrap(cas.ErrInvalid, "layout version is not supported")
	}

	// Check that "blobs" and "index.json" exist in the image.
	// FIXME: We also should check that blobs *only* contains a cas.BlobAlgorithm
	//        directory (with no subdirectories) and that refs *only* contains
	//        files (optionally also making sure they're all JSON descriptors).
	if fi, err := os.Stat(filepath.Join(e.path, blobDirectory)); err != nil {
		if os.IsNotExist(err) {
			err = cas.ErrInvalid
		}
		return errors.Wrap(err, "check blobdir")
	} else if !fi.IsDir() {
		return errors.Wrap(cas.ErrInvalid, "blobdir is not a directory")
	}

	if fi, err := os.Stat(filepath.Join(e.path, indexFile)); err != nil {
		if os.IsNotExist(err) {
			err = cas.ErrInvalid
		}
		return errors.Wrap(err, "check index")
	} else if fi.IsDir() {
		return errors.Wrap(cas.ErrInvalid, "index is a directory")
	}

	return nil
}

// PutBlob adds a new blob to the image. This is idempotent; a nil error
// means that "the content is stored at DIGEST" without implying "because
// of this PutBlob() call".
func (e *dirEngine) PutBlob(ctx context.Context, reader io.Reader) (digest.Digest, int64, error) {
	if err := e.ensureTempDir(); err != nil {
		return "", -1, errors.Wrap(err, "ensure tempdir")
	}

	digester := cas.BlobAlgorithm.Digester()

	// We copy this into a temporary file because we need to get the blob hash,
	// but also to avoid half-writing an invalid blob.
	fh, err := ioutil.TempFile(e.temp, "blob-")
	if err != nil {
		return "", -1, errors.Wrap(err, "create temporary blob")
	}
	tempPath := fh.Name()
	defer fh.Close()

	writer := io.MultiWriter(fh, digester.Hash())
	size, err := io.Copy(writer, reader)
	if err != nil {
		return "", -1, errors.Wrap(err, "copy to temporary blob")
	}
	fh.Close()

	// Get the digest.
	path, err := blobPath(digester.Digest())
	if err != nil {
		return "", -1, errors.Wrap(err, "compute blob name")
	}

	// Move the blob to its correct path.
	path = filepath.Join(e.path, path)
	if err := os.Rename(tempPath, path); err != nil {
		return "", -1, errors.Wrap(err, "rename temporary blob")
	}

	return digester.Digest(), int64(size), nil
}

// GetBlob returns a reader for retrieving a blob from the image, which the
// caller must Close(). Returns os.ErrNotExist if the digest is not found.
func (e *dirEngine) GetBlob(ctx context.Context, digest digest.Digest) (io.ReadCloser, error) {
	path, err := blobPath(digest)
	if err != nil {
		return nil, errors.Wrap(err, "compute blob path")
	}
	fh, err := os.Open(filepath.Join(e.path, path))
	return fh, errors.Wrap(err, "open blob")
}

// PutIndex sets the index of the OCI image to the given index, replacing the
// previously existing index. This operation is atomic; any readers attempting
// to access the OCI image while it is being modified will only ever see the
// new or old index.
func (e *dirEngine) PutIndex(ctx context.Context, index ispec.Index) error {
	if err := e.ensureTempDir(); err != nil {
		return errors.Wrap(err, "ensure tempdir")
	}

	// We copy this into a temporary index to ensure the atomicity of this
	// operation.
	fh, err := ioutil.TempFile(e.temp, "index-")
	if err != nil {
		return errors.Wrap(err, "create temporary index")
	}
	tempPath := fh.Name()
	defer fh.Close()

	// Encode the index.
	if err := json.NewEncoder(fh).Encode(index); err != nil {
		return errors.Wrap(err, "write temporary index")
	}
	fh.Close()

	// Move the blob to its correct path.
	path := filepath.Join(e.path, indexFile)
	if err := os.Rename(tempPath, path); err != nil {
		return errors.Wrap(err, "rename temporary index")
	}
	return nil
}

// GetIndex returns the index of the OCI image. Return ErrNotExist if the
// digest is not found. If the image doesn't have an index, ErrInvalid is
// returned (a valid OCI image MUST have an image index).
//
// It is not recommended that users of cas.Engine use this interface directly,
// due to the complication of properly handling references as well as correctly
// handling nested indexes. casext.Engine provides a wrapper for cas.Engine
// that implements various reference resolution functions that should work for
// most users.
func (e *dirEngine) GetIndex(ctx context.Context) (ispec.Index, error) {
	content, err := ioutil.ReadFile(filepath.Join(e.path, indexFile))
	if err != nil {
		if os.IsNotExist(err) {
			err = cas.ErrInvalid
		}
		return ispec.Index{}, errors.Wrap(err, "read index")
	}

	var index ispec.Index
	if err := json.Unmarshal(content, &index); err != nil {
		return ispec.Index{}, errors.Wrap(err, "parse index")
	}

	return index, nil
}

// DeleteBlob removes a blob from the image. This is idempotent; a nil
// error means "the content is not in the store" without implying "because
// of this DeleteBlob() call".
func (e *dirEngine) DeleteBlob(ctx context.Context, digest digest.Digest) error {
	path, err := blobPath(digest)
	if err != nil {
		return errors.Wrap(err, "compute blob path")
	}

	err = os.Remove(filepath.Join(e.path, path))
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "remove blob")
	}
	return nil
}

// ListBlobs returns the set of blob digests stored in the image.
func (e *dirEngine) ListBlobs(ctx context.Context) ([]digest.Digest, error) {
	digests := []digest.Digest{}
	blobDir := filepath.Join(e.path, blobDirectory, cas.BlobAlgorithm.String())

	if err := filepath.Walk(blobDir, func(path string, _ os.FileInfo, _ error) error {
		// Skip the actual directory.
		if path == blobDir {
			return nil
		}

		// XXX: Do we need to handle multiple-directory-deep cases?
		digest := digest.NewDigestFromHex(cas.BlobAlgorithm.String(), filepath.Base(path))
		digests = append(digests, digest)
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "walk blobdir")
	}

	return digests, nil
}

// Clean executes a garbage collection of any non-blob garbage in the store
// (this includes temporary files and directories not reachable from the CAS
// interface). This MUST NOT remove any blobs or references in the store.
func (e *dirEngine) Clean(ctx context.Context) error {
	// Remove every .umoci directory that isn't flocked.
	matches, err := filepath.Glob(filepath.Join(e.path, ".umoci-*"))
	if err != nil {
		return errors.Wrap(err, "glob .umoci-*")
	}
	for _, path := range matches {
		err = e.cleanPath(ctx, path)
		if err != nil && err != filepath.SkipDir {
			return err
		}
	}

	return nil
}

func (e *dirEngine) cleanPath(ctx context.Context, path string) error {
	cfh, err := os.Open(path)
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "open for locking")
	}
	defer cfh.Close()

	if err := unix.Flock(int(cfh.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		// If we fail to get a flock(2) then it's probably already locked,
		// so we shouldn't touch it.
		return filepath.SkipDir
	}
	defer unix.Flock(int(cfh.Fd()), unix.LOCK_UN)

	if err := os.RemoveAll(path); os.IsNotExist(err) {
		return nil // somebody else beat us to it
	} else if err != nil {
		log.Warnf("failed to remove %s: %v", path, err)
		return filepath.SkipDir
	}
	log.Debugf("cleaned %s", path)

	return nil
}

// Close releases all references held by the e. Subsequent operations may
// fail.
func (e *dirEngine) Close() error {
	if e.temp != "" {
		if err := unix.Flock(int(e.tempFile.Fd()), unix.LOCK_UN); err != nil {
			return errors.Wrap(err, "unlock tempdir")
		}
		if err := e.tempFile.Close(); err != nil {
			return errors.Wrap(err, "close tempdir")
		}
		if err := os.RemoveAll(e.temp); err != nil {
			return errors.Wrap(err, "remove tempdir")
		}
	}
	return nil
}

// Open opens a new reference to the directory-backed OCI image referenced by
// the provided path.
func Open(path string) (cas.Engine, error) {
	engine := &dirEngine{
		path: path,
		temp: "",
	}

	if err := engine.validate(); err != nil {
		return nil, errors.Wrap(err, "validate")
	}

	return engine, nil
}

// Create creates a new OCI image layout at the given path. If the path already
// exists, os.ErrExist is returned. However, all of the parent components of
// the path will be created if necessary.
func Create(path string) error {
	// We need to fail if path already exists, but we first create all of the
	// parent paths.
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return errors.Wrap(err, "mkdir parent")
		}
	}
	if err := os.Mkdir(path, 0755); err != nil {
		return errors.Wrap(err, "mkdir")
	}

	// Create the necessary directories and "oci-layout" file.
	if err := os.Mkdir(filepath.Join(path, blobDirectory), 0755); err != nil {
		return errors.Wrap(err, "mkdir blobdir")
	}
	if err := os.Mkdir(filepath.Join(path, blobDirectory, cas.BlobAlgorithm.String()), 0755); err != nil {
		return errors.Wrap(err, "mkdir algorithm")
	}

	indexFh, err := os.Create(filepath.Join(path, indexFile))
	if err != nil {
		return errors.Wrap(err, "create index.json")
	}
	defer indexFh.Close()

	defaultIndex := ispec.Index{
		Versioned: imeta.Versioned{
			SchemaVersion: 2, // FIXME: This is hardcoded at the moment.
		},
	}
	if err := json.NewEncoder(indexFh).Encode(defaultIndex); err != nil {
		return errors.Wrap(err, "encode index.json")
	}

	layoutFh, err := os.Create(filepath.Join(path, layoutFile))
	if err != nil {
		return errors.Wrap(err, "create oci-layout")
	}
	defer layoutFh.Close()

	ociLayout := ispec.ImageLayout{
		Version: ImageLayoutVersion,
	}
	if err := json.NewEncoder(layoutFh).Encode(ociLayout); err != nil {
		return errors.Wrap(err, "encode oci-layout")
	}

	// Everything is now set up.
	return nil
}
