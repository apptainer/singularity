// Copyright (c) 2020-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the LICENSE.md file
// distributed with the sources of this project regarding your rights to use or distribute this
// software.

package sypgp

import "github.com/ProtonMail/go-crypto/openpgp"

// EntitySelector selects an Entity given an EntityList.
type EntitySelector func(el openpgp.EntityList) (*openpgp.Entity, error)

// getPrivateEntity retrieves the entity selected by f from keyring.
func (keyring *Handle) getPrivateEntity(f EntitySelector) (*openpgp.Entity, error) {
	el, err := keyring.LoadPrivKeyring()
	if err != nil {
		return nil, err
	}
	return f(el)
}

// GetPrivateEntity retrieves the entity selected by f from the Singularity private keyring.
func GetPrivateEntity(f EntitySelector) (*openpgp.Entity, error) {
	return NewHandle("").getPrivateEntity(f)
}
