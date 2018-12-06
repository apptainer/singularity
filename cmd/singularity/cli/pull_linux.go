// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/build/types"
	"github.com/sylabs/singularity/internal/pkg/libexec"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
)

func pullRun(cmd *cobra.Command, args []string) {
	i := len(args) - 1 // uri is stored in args[len(args)-1]
	transport, ref := uri.Split(args[i])
	if ref == "" {
		sylog.Fatalf("bad uri %s", args[i])
	}

	var name string
	if PullImageName == "" {
		name = args[0]
		if len(args) == 1 {
			name = uri.GetName(args[i]) // TODO: If not library/shub & no name specified, simply put to cache
		}
	} else {
		name = PullImageName
	}

	switch transport {
	case LibraryProtocol, "":
		libexec.PullLibraryImage(name, args[i], PullLibraryURI, force, authToken)
	case ShubProtocol:
		libexec.PullShubImage(name, args[i], force, noHTTPS)
	case HTTPProtocol, HTTPSProtocol:
		libexec.PullNetImage(name, args[i], force)
	default:
		authConf, err := makeDockerCredentials(dockerLogin)
		if err != nil {
			sylog.Fatalf("While creating Docker credentials: %v", err)
		}

		libexec.PullOciImage(name, args[i], types.Options{
			TmpDir:           tmpDir,
			Force:            force,
			NoHTTPS:          noHTTPS,
			DockerAuthConfig: authConf,
		})
	}
}
