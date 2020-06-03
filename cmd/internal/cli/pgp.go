// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the LICENSE.md file
// distributed with the sources of this project regarding your rights to use or distribute this
// software.

package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/sylabs/singularity/internal/pkg/util/interactive"
	"github.com/sylabs/singularity/pkg/sypgp"
	"golang.org/x/crypto/openpgp"
)

var (
	errEmptyKeyring    = errors.New("keyring is empty")
	errIndexOutOfRange = errors.New("index out of range")
)

// printEntityAtIndex prints entity e, associated with index i, to w.
func printEntityAtIndex(w io.Writer, i int, e *openpgp.Entity) {
	for _, v := range e.Identities {
		fmt.Fprintf(w, "%d) U: %s (%s) <%s>\n", i, v.UserId.Name, v.UserId.Comment, v.UserId.Email)
	}
	fmt.Fprintf(w, "   C: %s\n", e.PrimaryKey.CreationTime)
	fmt.Fprintf(w, "   F: %0X\n", e.PrimaryKey.Fingerprint)
	bits, _ := e.PrimaryKey.BitLength()
	fmt.Fprintf(w, "   L: %d\n", bits)
	fmt.Fprint(os.Stdout, "   --------\n")
}

// selectEntityInteractive returns an EntitySelector that selects an entity from el, prompting the
// user for a selection if there is more than one entity in el.
func selectEntityInteractive() sypgp.EntitySelector {
	return func(el openpgp.EntityList) (*openpgp.Entity, error) {
		switch len(el) {
		case 0:
			return nil, errEmptyKeyring
		case 1:
			return el[0], nil
		default:
			for i, e := range el {
				printEntityAtIndex(os.Stdout, i, e)
			}

			n, err := interactive.AskNumberInRange(0, len(el)-1, "Enter # of private key to use : ")
			if err != nil {
				return nil, err
			}
			return el[n], nil
		}
	}
}

// selectEntityAtIndex returns an EntitySelector that selects the entity at index i.
func selectEntityAtIndex(i int) sypgp.EntitySelector {
	return func(el openpgp.EntityList) (*openpgp.Entity, error) {
		if i >= len(el) {
			return nil, errIndexOutOfRange
		}
		return el[i], nil
	}
}

// decryptSelectedEntityInteractive wraps f, attempting to decrypt the private key in the selected
// entity with a passpharse provided interactively by the user.
func decryptSelectedEntityInteractive(f sypgp.EntitySelector) sypgp.EntitySelector {
	return func(el openpgp.EntityList) (*openpgp.Entity, error) {
		e, err := f(el)
		if err != nil {
			return nil, err
		}

		if e.PrivateKey.Encrypted {
			if err := decryptPrivateKeyInteractive(e); err != nil {
				return nil, err
			}
		}

		return e, nil
	}
}

// decryptPrivateKeyInteractive decrypts the private key in e, prompting the user for a passphrase.
func decryptPrivateKeyInteractive(e *openpgp.Entity) error {
	passphrase, err := interactive.AskQuestionNoEcho("Enter key passphrase : ")
	if err != nil {
		return err
	}

	return e.PrivateKey.Decrypt([]byte(passphrase))
}
