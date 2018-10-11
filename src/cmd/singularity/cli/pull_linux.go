package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/src/pkg/libexec"
	"github.com/sylabs/singularity/src/pkg/sylog"
	"github.com/sylabs/singularity/src/pkg/util/uri"
)

func pullRun(cmd *cobra.Command, args []string) {
	i := len(args) - 1 // uri is stored in args[len(args)-1]
	transport, ref := uri.SplitURI(args[i])
	if ref == "" {
		sylog.Fatalf("bad uri %s", args[i])
	}

	name := args[0]
	if len(args) == 1 {
		name = uri.NameFromURI(args[i]) // TODO: If not library/shub & no name specified, simply put to cache
	}

	switch transport {
	case LibraryProtocol, "":
		libexec.PullLibraryImage(name, args[i], PullLibraryURI, force, authToken)
	case ShubProtocol:
		libexec.PullShubImage(name, args[i], force)
	default:
		libexec.PullOciImage(name, args[i], force)
	}
}
