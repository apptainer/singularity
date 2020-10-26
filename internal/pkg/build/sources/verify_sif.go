// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"context"

	scskeyclient "github.com/sylabs/scs-key-client/client"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/pkg/sylog"
)

// checkSIFFingerprint checks whether a bootstrap SIF image verifies, and was signed with a specified fingerprint
func checkSIFFingerprint(imagePath string, fingerprints []string, kc *scskeyclient.Config) error {
	sylog.Infof("Checking bootstrap image verifies with fingerprint(s): %v", fingerprints)
	opts := []singularity.VerifyOpt{}
	if kc != nil {
		opts = append(opts, singularity.OptVerifyUseKeyServer(kc))
	}
	// TODO - we should attempt to pass context down to this level properly.
	ctx := context.TODO()
	return singularity.VerifyFingerprints(ctx, imagePath, fingerprints, opts...)
}

// verifySIF checks whether a bootstrap SIF image verifies
func verifySIF(imagePath string, kc *scskeyclient.Config) error {
	sylog.Infof("Verifying bootstrap image %s", imagePath)
	opts := []singularity.VerifyOpt{}
	if kc != nil {
		opts = append(opts, singularity.OptVerifyUseKeyServer(kc))
	}
	// TODO - we should attempt to pass context down to this level properly.
	ctx := context.TODO()
	return singularity.Verify(ctx, imagePath, opts...)
}
