package build

import (
	"context"
	"fmt"

	ocitypes "github.com/containers/image/v5/types"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	buildtypes "github.com/sylabs/singularity/pkg/build/types"
)

// ConvertOciToSIf will convert an OCI source into a SIF using the build routines
func ConvertOciToSIF(ctx context.Context, imgCache *cache.Handle, image, cachedImgPath, tmpDir string, noHTTPS, noCleanUp bool, authConf *ocitypes.DockerAuthConfig) error {
	if imgCache == nil {
		return fmt.Errorf("image cache is undefined")
	}

	b, err := NewBuild(
		image,
		Config{
			Dest:      cachedImgPath,
			Format:    "sif",
			NoCleanUp: noCleanUp,
			Opts: buildtypes.Options{
				TmpDir:           tmpDir,
				NoCache:          imgCache.IsDisabled(),
				NoTest:           true,
				NoHTTPS:          noHTTPS,
				DockerAuthConfig: authConf,
				ImgCache:         imgCache,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("unable to create new build: %v", err)
	}

	return b.Full(ctx)
}
