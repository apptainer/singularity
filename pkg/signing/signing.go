// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package signing

import (
	"fmt"
	"os"

	"github.com/sylabs/sif/pkg/sif"
)

// return all signatures for the primary partition
func getSigsPrimPart(fimg *sif.FileImage) (sigs []*sif.Descriptor, descr []*sif.Descriptor, err error) {
	descr = make([]*sif.Descriptor, 1)

	descr[0], _, err = fimg.GetPartPrimSys()
	if err != nil {
		return nil, nil, fmt.Errorf("no primary partition found")
	}

	sigs, _, err = fimg.GetLinkedDescrsByType(descr[0].ID, sif.DataSignature)
	if err != nil {
		return nil, nil, fmt.Errorf("no signatures found for system partition")
	}

	return
}

func getSignEntities(fimg *sif.FileImage) ([]string, error) {
	// get all signature blocks (signatures) for ID/GroupID selected (descr) from SIF file
	signatures, _, err := getSigsPrimPart(fimg)
	if err != nil {
		return nil, err
	}

	entities := make([]string, 0, len(signatures))
	for _, v := range signatures {
		fingerprint, err := v.GetEntityString()
		if err != nil {
			return nil, err
		}
		entities = append(entities, fingerprint)
	}

	return entities, nil
}

// GetSignEntities returns all signing entities for an ID/Groupid
func GetSignEntities(cpath string) ([]string, error) {
	fimg, err := sif.LoadContainer(cpath, true)
	if err != nil {
		fimg.UnloadContainer()
		return nil, err
	}
	defer fimg.UnloadContainer()

	return getSignEntities(&fimg)
}

// GetSignEntitiesFp returns all signing entities for an ID/Groupid
func GetSignEntitiesFp(fp *os.File) ([]string, error) {
	fimg, err := sif.LoadContainerFp(fp, true)
	if err != nil {
		return nil, err
	}

	return getSignEntities(&fimg)
}
