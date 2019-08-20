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
	"github.com/sylabs/sif/pkg/sif"
)

var (
	ErrEncryptedKeyNotFound = errors.New("encrypted key not found")
	ErrUnsupportedKeyURI    = errors.New("unsupported key URI")
	ErrNoEncryptedKeyData   = errors.New("no encrypted key data")
	ErrNoPEMData            = errors.New("No PEM data")
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

func PlaintextKey(keyURI, image string) ([]byte, error) {
	u, err := url.Parse(keyURI)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot parse URI %s", keyURI)
	}

	switch u.Scheme {
	case "pem":
		privateKey, err := loadPEMPrivateKey(u.Path)
		if err != nil {
			return nil, errors.Wrap(err, "loading private key for key decryption")
		}

		pemKey, err := getEncryptionKeyFromImage(image)
		if err != nil {
			return nil, errors.Wrapf(err, "loading encrypted key SIF image %s", image)
		}

		pemBuf := bytes.NewReader(pemKey)

		encKey, err := loadPEMMessage(pemBuf)
		if err != nil {
			return nil, errors.Wrapf(err, "unpacking PEM message from SIF image %s", image)
		}

		plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, encKey, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "decrypting key from image %s", image)
		}

		return plaintext, nil

	case "":
		return []byte(u.Path), nil

	default:
		return nil, ErrUnsupportedKeyURI
	}
}

func loadPEMPrivateKey(fn string) (*rsa.PrivateKey, error) {
	b, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.Wrapf(ErrNoPEMData, "reading %s", fn)
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
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

func loadPEMMessage(r io.Reader) ([]byte, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.Wrapf(ErrNoPEMData, "reading PEM block")
	}

	var buf []byte
	if _, err := asn1.Unmarshal(block.Bytes, &buf); err != nil {
		return nil, errors.Wrapf(err, "unmarshalling ASN.1 data")
	}

	return buf, nil
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

func getEncryptionKeyFromImage(fn string) ([]byte, error) {
	img, err := sif.LoadContainer(fn, true)
	if err != nil {
		return nil, errors.Wrapf(err, "loading container image from %s", fn)
	}
	defer img.UnloadContainer()

	primDescr, _, err := img.GetPartPrimSys()
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving primary system partition from %s", fn)
	}

	descr, _, err := img.GetLinkedDescrsByType(primDescr.ID, sif.DataCryptoMessage)
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving linked descriptors for primary system partition from %s", fn)
	}

	for _, d := range descr {
		format, err := d.GetFormatType()
		if err != nil {
			return nil, errors.Wrapf(err, "while retrieving cryptographic message format")
		}

		message, err := d.GetMessageType()
		if err != nil {
			return nil, errors.Wrapf(err, "while retrieving cryptographic message type")
		}

		// currently only support one type of message
		if format != sif.FormatPEM || message != sif.MessageRSAOAEP {
			continue
		}

		// TODO(ian): For now, assume the first linked message is what we
		// are looking for. We should consider what we want to do in the
		// case of multiple linked messages
		data := d.GetData(&img)
		if data == nil {
			return nil, errors.Wrapf(ErrNoEncryptedKeyData, "retrieving encrypted key data from %s", fn)
		}

		key := make([]byte, len(data))
		copy(key, data)

		return key, nil
	}

	return nil, errors.Wrapf(ErrEncryptedKeyNotFound, "reading from %s", fn)
}
