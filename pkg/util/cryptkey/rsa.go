// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cryptkey

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

const (
	// DefaultKeySize is the default size of the key that is used when a
	// size is not explicitly specified
	DefaultKeySize = 2048
)

// GenerateRSAKey creates a new RSA key of length keySize.
func GenerateRSAKey(keySize int) (*rsa.PrivateKey, error) {
	reader := rand.Reader

	if keySize == 0 {
		keySize = DefaultKeySize
	}

	key, err := rsa.GenerateKey(reader, keySize)
	if err != nil {
		return nil, fmt.Errorf("unable to generate RSA key: %v", err)
	}

	return key, nil
}

// publicPEM generates a new PEM public key based on a RSA key
func publicPEM(key *rsa.PrivateKey) (string, error) {
	var buf bytes.Buffer

	if key == nil {
		return "", errors.New("cannot encode nil key")
	}

	err := key.Validate()
	if err != nil {
		return "", fmt.Errorf("cannot encode invalid key: %v", err)
	}

	asn1Bytes, err := asn1.Marshal(key.PublicKey)
	if err != nil {
		return "", fmt.Errorf("unable to encode public key: %v", err)
	}

	pemkey := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: asn1Bytes,
	}

	err = pem.Encode(&buf, pemkey)
	if err != nil {
		return "", fmt.Errorf("error encoding key: %v", err)
	}

	return buf.String(), nil
}

// SavePublicPEM saves a public PEM key into a file.
func SavePublicPEM(fileName string, key *rsa.PrivateKey) error {
	pem, err := publicPEM(key)
	if err != nil {
		return err
	}

	pemfile, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("unable to create key file: %v", err)
	}
	defer pemfile.Close()

	_, err = pemfile.WriteString(pem)
	if err != nil {
		return fmt.Errorf("error writing key to file: %v", err)
	}

	return nil
}

// SavePrivatePEM saves a private PEM key into a file.
func SavePrivatePEM(fileName string, key *rsa.PrivateKey) error {
	if key == nil {
		return errors.New("cannot save nil key")
	}

	err := key.Validate()
	if err != nil {
		return fmt.Errorf("cannot save invalid key: %v", err)
	}

	outFile, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("unable to create key file: %v", err)
	}

	defer outFile.Close()

	privateKey := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	err = pem.Encode(outFile, privateKey)
	if err != nil {
		return fmt.Errorf("error writing key to file: %v", err)
	}

	return nil
}
