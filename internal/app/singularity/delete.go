package singularity

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sylabs/scs-library-client/client"
	scs "github.com/sylabs/scs-library-client/client"
)

func DeleteImage(scsConfig *scs.Config, imageRef, arch string) error {
	libraryClient, err := client.NewClient(scsConfig)
	if err != nil {
		return errors.Wrap(err, "couldn't create a new client")
	}

	ctx, _ := context.WithTimeout(context.Background(), 15*time.Second)

	return libraryClient.DeleteImage(ctx, imageRef, arch)
}
