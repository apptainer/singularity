// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package crypt

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"io"
	"io/ioutil"
	"net/url"

	"github.com/pkg/errors"
)

var (
	ErrUnsupportedKeyURI = errors.New("unsupported key URI")
	ErrNoPEMData         = errors.New("No PEM data")
)

func getRandomBytes(size int) ([]byte, error) {
	buf := make([]byte, size)
	_, err := rand.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func NewPlaintextKey(keyURI string) ([]byte, error) {
	u, err := url.Parse(keyURI)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot parse URI %s", keyURI)
	}

	switch u.Scheme {
	case "pem":
		// in this case we will generate a random secret and
		// encrypt it using the PEM key.use the PEM key to
		// encrypt a secret
		return getRandomBytes(64)

	case "":
		// return the original value unmodified
		return []byte(keyURI), nil

	default:
		return nil, ErrUnsupportedKeyURI
	}
}

func EncryptKey(keyURI string, plaintext []byte) ([]byte, error) {
	u, err := url.Parse(keyURI)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot parse URI %s", keyURI)
	}

	switch u.Scheme {
	case "pem":
		pubKey, err := loadPEMPublicKey(u.Path)
		if err != nil {
			return nil, errors.Wrap(err, "loading public key for key encryption")
		}

		ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, plaintext, nil)
		if err != nil {
			return nil, errors.Wrap(err, "encrypting key")
		}

		var buf bytes.Buffer

		if err := savePEMMessage(&buf, ciphertext); err != nil {
			return nil, errors.Wrap(err, "serializing encrypted key")
		}

		return buf.Bytes(), nil

	case "":
		return nil, nil

	default:
		return nil, ErrUnsupportedKeyURI
	}
}

func loadPEMPublicKey(fn string) (*rsa.PublicKey, error) {
	b, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.Wrapf(ErrNoPEMData, "reading %s", fn)
	}

	return x509.ParsePKCS1PublicKey(block.Bytes)
}

func savePEMMessage(w io.Writer, msg []byte) error {
	asn1Bytes, err := asn1.Marshal(msg)
	if err != nil {
		return err
	}

	var b = &pem.Block{
		Type:  "MESSAGE",
		Bytes: asn1Bytes,
	}

	return pem.Encode(w, b)
}
