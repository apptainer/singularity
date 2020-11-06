// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the LICENSE.md file
// distributed with the sources of this project regarding your rights to use or distribute this
// software.

package singularity

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/sylabs/scs-key-client/client"
	"github.com/sylabs/sif/pkg/integrity"
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/pkg/sypgp"
	"golang.org/x/crypto/openpgp"
)

// TODO - error overlaps with ECL - should probably become part of a common errors package at some point.
var errNotSignedByRequired = errors.New("image not signed by required entities")

type VerifyCallback func(*sif.FileImage, integrity.VerifyResult) bool

type verifier struct {
	opts      []client.Option
	groupIDs  []uint32
	objectIDs []uint32
	all       bool
	legacy    bool
	cb        VerifyCallback
}

// VerifyOpt are used to configure v.
type VerifyOpt func(v *verifier) error

// OptVerifyUseKeyServer specifies that the keyserver specified by opts be used as a source of key
// material, in addition to the local public keyring.
func OptVerifyUseKeyServer(opts ...client.Option) VerifyOpt {
	return func(v *verifier) error {
		v.opts = opts
		return nil
	}
}

// OptVerifyGroup adds a verification task for the group with the specified groupID. This may be
// called multliple times to request verification of more than one group.
func OptVerifyGroup(groupID uint32) VerifyOpt {
	return func(v *verifier) error {
		v.groupIDs = append(v.groupIDs, groupID)
		return nil
	}
}

// OptVerifyObject adds a verification task for the object with the specified id. This may be
// called multliple times to request verification of more than one object.
func OptVerifyObject(id uint32) VerifyOpt {
	return func(v *verifier) error {
		v.objectIDs = append(v.objectIDs, id)
		return nil
	}
}

// OptVerifyAll adds one verification task per non-signature object in the image when verification
// of legacy signatures is enabled. When verification of legacy signatures is disabled (the
// default), this option has no effect.
func OptVerifyAll() VerifyOpt {
	return func(v *verifier) error {
		v.all = true
		return nil
	}
}

// OptVerifyLegacy enables verification of legacy signatures.
func OptVerifyLegacy() VerifyOpt {
	return func(v *verifier) error {
		v.legacy = true
		return nil
	}
}

// OptVerifyCallback registers f as the verification callback.
func OptVerifyCallback(cb VerifyCallback) VerifyOpt {
	return func(v *verifier) error {
		v.cb = cb
		return nil
	}
}

// newVerifier constructs a new verifier based on opts.
func newVerifier(opts []VerifyOpt) (verifier, error) {
	v := verifier{}
	for _, opt := range opts {
		if err := opt(&v); err != nil {
			return verifier{}, err
		}
	}
	return v, nil
}

// getOpts returns integrity.VerifierOpt necessary to validate f.
func (v verifier) getOpts(ctx context.Context, f *sif.FileImage) ([]integrity.VerifierOpt, error) {
	var iopts []integrity.VerifierOpt

	// Add keyring.
	var kr openpgp.KeyRing
	if v.opts != nil {
		hkr, err := sypgp.NewHybridKeyRing(ctx, v.opts...)
		if err != nil {
			return nil, err
		}
		kr = hkr
	} else {
		pkr, err := sypgp.PublicKeyRing()
		if err != nil {
			return nil, err
		}
		kr = pkr
	}

	// wrap the global keyring around
	global := sypgp.NewHandle(buildcfg.SINGULARITY_CONFDIR, sypgp.GlobalHandleOpt())
	gkr, err := global.LoadPubKeyring()
	if err != nil {
		return nil, err
	}
	kr = sypgp.NewMultiKeyRing(gkr, kr)

	iopts = append(iopts, integrity.OptVerifyWithKeyRing(kr))

	// Add group IDs, if applicable.
	for _, groupID := range v.groupIDs {
		iopts = append(iopts, integrity.OptVerifyGroup(groupID))
	}

	// Add objectIDs, if applicable.
	for _, objectID := range v.objectIDs {
		iopts = append(iopts, integrity.OptVerifyObject(objectID))
	}

	// Set legacy options, if applicable.
	if v.legacy {
		if v.all {
			iopts = append(iopts, integrity.OptVerifyLegacyAll())
		} else {
			iopts = append(iopts, integrity.OptVerifyLegacy())

			// If no objects explicitly selected, select system partition.
			if len(v.groupIDs) == 0 && len(v.objectIDs) == 0 {
				od, _, err := f.GetPartPrimSys()
				if err != nil {
					return nil, err
				}
				iopts = append(iopts, integrity.OptVerifyObject(od.ID))
			}
		}
	}

	// Add callback, if applicable.
	if v.cb != nil {
		fn := func(r integrity.VerifyResult) bool {
			return v.cb(f, r)
		}
		iopts = append(iopts, integrity.OptVerifyCallback(fn))
	}

	return iopts, nil
}

// Verify verifies digital signature(s) in the SIF image found at path, according to opts.
//
// By default, the singularity public keyring provides key material. To supplement this with a
// keyserver, use OptVerifyUseKeyServer.
//
// By default, non-legacy signatures for all object groups are verified. To override the default
// behavior, consider using OptVerifyGroup, OptVerifyObject, OptVerifyAll, and/or OptVerifyLegacy.
func Verify(ctx context.Context, path string, opts ...VerifyOpt) error {
	v, err := newVerifier(opts)
	if err != nil {
		return err
	}

	// Load container.
	f, err := sif.LoadContainer(path, true)
	if err != nil {
		return err
	}
	defer f.UnloadContainer()

	// Get options to validate f.
	vopts, err := v.getOpts(ctx, &f)
	if err != nil {
		return err
	}

	// Verify signature(s).
	iv, err := integrity.NewVerifier(&f, vopts...)
	if err != nil {
		return err
	}
	return iv.Verify()
}

// VerifyFingerprints verifies an image and checks it was signed by *all* of the provided fingerprints
//
// By default, the singularity public keyring provides key material. To supplement this with a
// keyserver, use OptVerifyUseKeyServer.
//
// By default, non-legacy signatures for all object groups are verified. To override the default
// behavior, consider using OptVerifyGroup, OptVerifyObject, OptVerifyAll, and/or OptVerifyLegacy.
func VerifyFingerprints(ctx context.Context, path string, fingerprints []string, opts ...VerifyOpt) error {
	v, err := newVerifier(opts)
	if err != nil {
		return err
	}

	// Load container.
	f, err := sif.LoadContainer(path, true)
	if err != nil {
		return err
	}
	defer f.UnloadContainer()

	// Get options to validate f.
	vopts, err := v.getOpts(ctx, &f)
	if err != nil {
		return err
	}

	// Verify signature(s).
	iv, err := integrity.NewVerifier(&f, vopts...)
	if err != nil {
		return err
	}
	err = iv.Verify()
	if err != nil {
		return err
	}

	// get signing entities fingerprints that have signed all selected objects
	keyfps, err := iv.AllSignedBy()
	if err != nil {
		return err
	}
	// were the selected objects signed by the provided fingerprints?

	m := map[string]bool{}
	for _, v := range fingerprints {
		m[v] = false
		for _, u := range keyfps {
			if strings.EqualFold(v, hex.EncodeToString(u[:])) {
				m[v] = true
			}
		}
	}
	for _, v := range m {
		if !v {
			return errNotSignedByRequired
		}
	}
	return nil
}
