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

	jsonresp "github.com/sylabs/json-resp"
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
	local   openpgp.KeyRing  // Local keyring.
	ctx     context.Context  // Context, for use when retrieving keys remotely.
	clients []*client.Client // Keyserver client.
}

// NewHybridKeyRing returns a keyring backed by both the local public keyring and the configured
// keyserver.
func NewHybridKeyRing(ctx context.Context, cfg []*client.Config) (openpgp.KeyRing, error) {
	// Get local keyring.
	kr, err := PublicKeyRing()
	if err != nil {
		return nil, err
	}

	var clients []*client.Client

	// Set up clients to retrieve keys from keyserver.
	for _, clientConfig := range cfg {
		c, err := client.NewClient(clientConfig)
		if err != nil {
			return nil, err
		}
		clients = append(clients, c)
	}

	return &hybridKeyRing{
		local:   kr,
		ctx:     ctx,
		clients: clients,
	}, nil
}

// KeysById returns the set of keys that have the given key id.
//nolint:golint  // golang/x/crypto uses Id instead of ID so we have to too
func (kr *hybridKeyRing) KeysById(id uint64) []openpgp.Key {
	if keys := kr.local.KeysById(id); len(keys) > 0 {
		return keys
	}

	// No keys found in local keyring, check with keyservers.
	for _, c := range kr.clients {
		el, err := kr.remoteEntitiesByID(c, id)
		if err != nil {
			sylog.Warningf("failed to get key material from %s: %v", c.BaseURL.String(), err)
			continue
		}
		keys := el.KeysById(id)
		if len(keys) > 0 {
			return keys
		}
	}

	return nil
}

// KeysByIdUsage returns the set of keys with the given id that also meet the key usage given by
// requiredUsage. The requiredUsage is expressed as the bitwise-OR of packet.KeyFlag* values.
//nolint:golint  // golang/x/crypto uses Id instead of ID so we have to too
func (kr *hybridKeyRing) KeysByIdUsage(id uint64, requiredUsage byte) []openpgp.Key {
	if keys := kr.local.KeysByIdUsage(id, requiredUsage); len(keys) > 0 {
		return keys
	}

	// No keys found in local keyring, check with keyservers.
	for _, c := range kr.clients {
		el, err := kr.remoteEntitiesByID(c, id)
		if err != nil {
			sylog.Warningf("failed to get key material from %s: %v", c.BaseURL.String(), err)
			continue
		}
		keys := el.KeysByIdUsage(id, requiredUsage)
		if len(keys) > 0 {
			return keys
		}
	}

	return nil
}

// DecryptionKeys returns all private keys that are valid for decryption.
func (kr *hybridKeyRing) DecryptionKeys() []openpgp.Key {
	return kr.local.DecryptionKeys()
}

// remoteEntitiesByID returns the set of entities from the keyserver that have the given key id.
func (kr *hybridKeyRing) remoteEntitiesByID(c *client.Client, id uint64) (openpgp.EntityList, error) {
	kt, err := c.PKSLookup(kr.ctx, nil, fmt.Sprintf("%#x", id), client.OperationGet, false, true, nil)
	if err != nil {
		// If the request failed with HTTP status code unauthorized, guide the user to fix that.
		var jerr *jsonresp.Error
		if errors.As(err, &jerr) {
			if jerr.Code == http.StatusUnauthorized {
				sylog.Infof(helpAuth)
			}
		}

		return nil, err
	}

	return openpgp.ReadArmoredKeyRing(strings.NewReader(kt))
}
