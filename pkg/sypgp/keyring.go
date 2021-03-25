// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the LICENSE.md file
// distributed with the sources of this project regarding your rights to use or distribute this
// software.

package sypgp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/sylabs/scs-key-client/client"
	"github.com/sylabs/singularity/pkg/sylog"
	"golang.org/x/crypto/openpgp"
)

// PublicKeyRing retrieves the Singularity public KeyRing.
func PublicKeyRing() (openpgp.KeyRing, error) {
	return NewHandle("").LoadPubKeyring()
}

// hybridKeyRing is keyring made up of a local keyring as well as a keyserver. The type satisfies
// the openpgp.KeyRing interface.
type hybridKeyRing struct {
	local openpgp.KeyRing // Local keyring.
	ctx   context.Context // Context, for use when retrieving keys remotely.
	c     *client.Client  // Keyserver client.
}

// NewHybridKeyRing returns a keyring backed by both the local public keyring and the configured
// keyserver.
func NewHybridKeyRing(ctx context.Context, opts ...client.Option) (openpgp.KeyRing, error) {
	// Get local keyring.
	kr, err := PublicKeyRing()
	if err != nil {
		return nil, err
	}

	// Set up client to retrieve keys from keyserver.
	c, err := client.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	return &hybridKeyRing{
		local: kr,
		ctx:   ctx,
		c:     c,
	}, nil
}

// KeysById returns the set of keys that have the given key id.
//nolint:golint  // golang/x/crypto uses Id instead of ID so we have to too
func (kr *hybridKeyRing) KeysById(id uint64) []openpgp.Key {
	if keys := kr.local.KeysById(id); len(keys) > 0 {
		return keys
	}

	// No keys found in local keyring, check with keyserver.
	el, err := kr.remoteEntitiesByID(id)
	if err != nil {
		sylog.Warningf("failed to get key material: %v", err)
		return nil
	}
	return el.KeysById(id)
}

// KeysByIdUsage returns the set of keys with the given id that also meet the key usage given by
// requiredUsage. The requiredUsage is expressed as the bitwise-OR of packet.KeyFlag* values.
//nolint:golint  // golang/x/crypto uses Id instead of ID so we have to too
func (kr *hybridKeyRing) KeysByIdUsage(id uint64, requiredUsage byte) []openpgp.Key {
	if keys := kr.local.KeysByIdUsage(id, requiredUsage); len(keys) > 0 {
		return keys
	}

	// No keys found in local keyring, check with keyserver.
	el, err := kr.remoteEntitiesByID(id)
	if err != nil {
		sylog.Warningf("failed to get key material: %v", err)
		return nil
	}
	return el.KeysByIdUsage(id, requiredUsage)
}

// DecryptionKeys returns all private keys that are valid for decryption.
func (kr *hybridKeyRing) DecryptionKeys() []openpgp.Key {
	return kr.local.DecryptionKeys()
}

// remoteEntitiesByID returns the set of entities from the keyserver that have the given key id.
func (kr *hybridKeyRing) remoteEntitiesByID(id uint64) (openpgp.EntityList, error) {
	kt, err := kr.c.PKSLookup(kr.ctx, nil, fmt.Sprintf("%#x", id), client.OperationGet, false, true, nil)
	if err != nil {
		// If the request failed with HTTP status code unauthorized, guide the user to fix that.
		var httpError *client.HTTPError
		if errors.As(err, &httpError) && httpError.Code() == http.StatusUnauthorized {
			sylog.Infof(helpAuth)
		}
		return nil, err
	}

	return openpgp.ReadArmoredKeyRing(strings.NewReader(kt))
}

type multiKeyRing struct {
	keyrings []openpgp.KeyRing
}

// NewMultiKeyRing returns a keyring backed by different public keyring.
func NewMultiKeyRing(keyrings ...openpgp.KeyRing) openpgp.KeyRing {
	return &multiKeyRing{keyrings: keyrings}
}

// KeysById returns the set of keys that have the given key id.
//nolint:golint  // golang/x/crypto uses Id instead of ID so we have to too
func (mkr *multiKeyRing) KeysById(id uint64) []openpgp.Key {
	for _, kr := range mkr.keyrings {
		if keys := kr.KeysById(id); len(keys) > 0 {
			return keys
		}
	}
	return nil
}

// KeysByIdUsage returns the set of keys with the given id that also meet the key usage given by
// requiredUsage. The requiredUsage is expressed as the bitwise-OR of packet.KeyFlag* values.
//nolint:golint  // golang/x/crypto uses Id instead of ID so we have to too
func (mkr *multiKeyRing) KeysByIdUsage(id uint64, requiredUsage byte) []openpgp.Key {
	for _, kr := range mkr.keyrings {
		if keys := kr.KeysByIdUsage(id, requiredUsage); len(keys) > 0 {
			return keys
		}
	}
	return nil
}

// DecryptionKeys returns all private keys that are valid for decryption.
func (mkr *multiKeyRing) DecryptionKeys() []openpgp.Key {
	for _, kr := range mkr.keyrings {
		if keys := kr.DecryptionKeys(); len(keys) > 0 {
			return keys
		}
	}
	return nil
}
