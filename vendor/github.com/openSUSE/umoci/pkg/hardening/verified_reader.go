/*
 * umoci: Umoci Modifies Open Containers' Images
 * Copyright (C) 2018 SUSE LLC.
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

package hardening

import (
	"io"

	"github.com/apex/log"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

// Exported errors for verification issues that occur during processing within
// VerifiedReadCloser. Note that you will need to use
// "github.com/pkg/errors".Cause to get these exported errors in most cases.
var (
	ErrDigestMismatch = errors.Errorf("verified reader digest mismatch")
	ErrSizeMismatch   = errors.Errorf("verified reader size mismatch")
)

// VerifiedReadCloser is a basic io.ReadCloser which allows for simple
// verification that a stream matches an expected hash. The entire stream is
// hashed while being passed through this reader, and on EOF it will verify
// that the hash matches the expected hash. If not, an error is returned. Note
// that this means you need to read all input to EOF in order to find
// verification errors.
//
// If Reader is a VerifiedReadCloser (with the same ExpectedDigest), all of the
// methods are just piped to the underlying methods (with no verification in
// the upper layer).
type VerifiedReadCloser struct {
	// Reader is the underlying reader.
	Reader io.ReadCloser

	// ExpectedDigest is the expected digest. When the underlying reader
	// returns an EOF, the entire stream's sum will be compared to this hash
	// and an error will be returned if they don't match.
	ExpectedDigest digest.Digest

	// ExpectedSize is the expected amount of data to be read overall. If the
	// underlying reader hasn't returned an EOF by the time this value is
	// exceeded, an error is returned and no further reads will occur.
	ExpectedSize int64

	// digester stores the current state of the stream's hash.
	digester digest.Digester

	// currentSize is the number of bytes that have been read so far.
	currentSize int64
}

func (v *VerifiedReadCloser) init() {
	// Define digester if not already set.
	if v.digester == nil {
		alg := v.ExpectedDigest.Algorithm()
		if !alg.Available() {
			log.Fatalf("verified reader: unsupported hash algorithm %s", alg)
			panic("verified reader: unreachable section") // should never be hit
		}
		v.digester = alg.Digester()
	}
}

func (v *VerifiedReadCloser) isNoop() bool {
	innerV, ok := v.Reader.(*VerifiedReadCloser)
	return ok &&
		innerV.ExpectedDigest == v.ExpectedDigest &&
		innerV.ExpectedSize == v.ExpectedSize
}

func (v *VerifiedReadCloser) verify(nilErr error) error {
	// Digest mismatch (always takes precedence)?
	if actualDigest := v.digester.Digest(); actualDigest != v.ExpectedDigest {
		return errors.Wrapf(ErrDigestMismatch, "expected %s not %s", v.ExpectedDigest, actualDigest)
	}
	// Do we need to check the size for mismatches?
	if v.ExpectedSize >= 0 {
		switch {
		// Not enough bytes in the stream.
		case v.currentSize < v.ExpectedSize:
			return errors.Wrapf(ErrSizeMismatch, "expected %d bytes (only %d bytes in stream)", v.ExpectedSize, v.currentSize)

		// We don't read the entire blob, so the message needs to be slightly adjusted.
		case v.currentSize > v.ExpectedSize:
			return errors.Wrapf(ErrSizeMismatch, "expected %d bytes (extra bytes in stream)", v.ExpectedSize)

		}
	}
	// Forward the provided error.
	return nilErr
}

// Read is a wrapper around VerifiedReadCloser.Reader, with a digest check on
// EOF.  Make sure that you always check for EOF and read-to-the-end for all
// files.
func (v *VerifiedReadCloser) Read(p []byte) (n int, err error) {
	// Make sure we don't read after v.ExpectedSize has been passed.
	err = io.EOF
	left := v.ExpectedSize - v.currentSize
	switch {
	// ExpectedSize has been disabled.
	case v.ExpectedSize < 0:
		n, err = v.Reader.Read(p)

	// We still have something left to read.
	case left > 0:
		if int64(len(p)) > left {
			p = p[:left]
		}
		// Piped to the underling read.
		n, err = v.Reader.Read(p)
		v.currentSize += int64(n)

	// We have either read everything, or just happened to land on a boundary
	// (with potentially more things afterwards). So we must check if there is
	// anything left by doing a 1-byte read (Go doesn't allow for zero-length
	// Read()s to give EOFs).
	case left == 0:
		// We just want to know whether we read something (n>0). #nosec G104
		nTmp, _ := v.Reader.Read(make([]byte, 1))
		v.currentSize += int64(nTmp)
	}
	// Are we going to be a noop?
	if v.isNoop() {
		return n, err
	}
	// Make sure we're ready.
	v.init()
	// Forward it to the digester.
	if n > 0 {
		// hash.Hash guarantees Write() never fails and is never short.
		nWrite, err := v.digester.Hash().Write(p[:n])
		if nWrite != n || err != nil {
			log.Fatalf("verified reader: short write to %s Digester (err=%v)", v.ExpectedDigest.Algorithm(), err)
			panic("verified reader: unreachable section") // should never be hit
		}
	}
	// We have finished reading -- let's verify the state!
	if errors.Cause(err) == io.EOF {
		err = v.verify(err)
	}
	return n, err
}

// Close is a wrapper around VerifiedReadCloser.Reader, but with a digest check
// which will return an error if the underlying Close() didn't.
func (v *VerifiedReadCloser) Close() error {
	// Piped to underlying close.
	err := v.Reader.Close()
	if err != nil {
		return err
	}
	// Are we going to be a noop?
	if v.isNoop() {
		return err
	}
	// Make sure we're ready.
	v.init()
	// Verify the state.
	return v.verify(nil)
}
