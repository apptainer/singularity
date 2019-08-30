package singularity

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sylabs/scs-library-client/client"
	scs "github.com/sylabs/scs-library-client/client"
)

// DeleteImage deletes an image from a remote library.
func DeleteImage(ctx context.Context, scsConfig *scs.Config, imageRef, arch string) error {
	libraryClient, err := client.NewClient(scsConfig)
	if err != nil {
		return errors.Wrap(err, "couldn't create a new client")
	}

	err = libraryClient.DeleteImage(ctx, imageRef, arch)
	if err != nil {
		return errors.Wrap(err, "couldn't delete requested image")
	}

	return nil
}
