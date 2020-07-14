// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the LICENSE.md file
// distributed with the sources of this project regarding your rights to use or distribute this
// software.

package singularity

import (
	"github.com/sylabs/sif/pkg/integrity"
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/pkg/sypgp"
)

type signer struct {
	opts []integrity.SignerOpt
}

// SignOpt are used to configure s.
type SignOpt func(s *signer) error

// OptSignEntitySelector specifies f be used to select (and decrypt, if necessary) the entity to
// use to generate signature(s).
func OptSignEntitySelector(f sypgp.EntitySelector) SignOpt {
	return func(s *signer) error {
		e, err := sypgp.GetPrivateEntity(f)
		if err != nil {
			return err
		}

		s.opts = append(s.opts, integrity.OptSignWithEntity(e))

		return nil
	}
}

// OptSignGroup specifies that a signature be applied to cover all objects in the group with the
// specified groupID. This may be called multiple times to add multiple group signatures.
func OptSignGroup(groupID uint32) SignOpt {
	return func(s *signer) error {
		s.opts = append(s.opts, integrity.OptSignGroup(groupID))
		return nil
	}
}

// OptSignObjects specifies that one or more signature(s) be applied to cover objects with the
// specified ids. One signature will be applied for each group ID associated with the object(s).
// This may be called multiple times to add multiple signatures.
func OptSignObjects(ids ...uint32) SignOpt {
	return func(s *signer) error {
		s.opts = append(s.opts, integrity.OptSignObjects(ids...))
		return nil
	}
}

// Sign adds one or more digital signatures to the SIF image found at path, according to opts. Key
// material must be provided via OptSignEntitySelector.
//
// By default, one digital signature is added per object group in f. To override this behavior,
// consider using OptSignGroup and/or OptSignObject.
func Sign(path string, opts ...SignOpt) error {
	// Apply options to signer.
	s := signer{}
	for _, opt := range opts {
		if err := opt(&s); err != nil {
			return err
		}
	}

	// Load container.
	f, err := sif.LoadContainer(path, false)
	if err != nil {
		return err
	}
	defer f.UnloadContainer()

	// Apply signature(s).
	is, err := integrity.NewSigner(&f, s.opts...)
	if err != nil {
		return err
	}
	return is.Sign()
}
