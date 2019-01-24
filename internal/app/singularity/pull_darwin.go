// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	library "github.com/sylabs/singularity/pkg/client/library"
	shub "github.com/sylabs/singularity/pkg/client/shub"
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
		// would use libexec.PullLibraryImage but it pulls in build.X
		err := library.DownloadImage(name, args[i], PullLibraryURI, force, authToken)
		if err != nil {
			sylog.Fatalf("%v\n", err)
		}
	case ShubProtocol:
		// would use libexec.PullShubImage but it pulls in build.X
		err := shub.DownloadImage(name, args[i], force, noHTTPS)
		if err != nil {
			sylog.Fatalf("%v\n", err)
		}
	default:
		sylog.Fatalf("%s unsupported on this platform", transport)
	}
}
