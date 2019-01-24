// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/libexec"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	"github.com/sylabs/singularity/pkg/build/types"
	client "github.com/sylabs/singularity/pkg/client/library"
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
		if !force {
			if _, err := os.Stat(name); err == nil {
				sylog.Fatalf("image file already exists - will not overwrite")
			}
		}

		libraryImage, err := client.GetImage(PullLibraryURI, authToken, args[i])
		if err != nil {
			sylog.Fatalf("While getting image info: %v", err)
		}

		imageName := uri.GetName(args[i])
		imagePath := cache.LibraryImage(libraryImage.Hash, imageName)
		if exists, err := cache.LibraryImageExists(libraryImage.Hash, imageName); err != nil {
			sylog.Fatalf("unable to check if %v exists: %v", imagePath, err)
		} else if !exists {
			sylog.Infof("Downloading library image")
			client.DownloadImage(imagePath, args[i], PullLibraryURI, false, authToken)
		}

		// Perms are 777 *prior* to umask
		dstFile, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
		if err != nil {
			sylog.Fatalf("%v\n", err)
		}
		defer dstFile.Close()

		srcFile, err := os.OpenFile(imagePath, os.O_RDONLY, 0444)
		if err != nil {
			sylog.Fatalf("%v\n", err)
		}
		defer srcFile.Close()

		// Copy SIF from cache
		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			sylog.Fatalf("%v\n", err)
		}
	case ShubProtocol:
		libexec.PullShubImage(name, args[i], force, noHTTPS)
	case HTTPProtocol, HTTPSProtocol:
		libexec.PullNetImage(name, args[i], force)
	default:
		authConf, err := makeDockerCredentials(cmd)
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
