package singularity

import (
	"context"
	"fmt"

	"github.com/sylabs/scs-library-client/client"
	scs "github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/library"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/internal/pkg/util/uri"
	"github.com/sylabs/singularity/pkg/signing"
)

var errNotInCache = fmt.Errorf("image was not found in cache")

// Library is a Registry implementation for Sylabs Cloud Library.
type Library struct {
	keystoreURI string

	client *scs.Client
	cache  *cache.Handle
}

// NewLibrary initializes and returns new Library ready to  be used.
func NewLibrary(scsConfig *scs.Config, cache *cache.Handle, keystoreURI string) (*Library, error) {
	libraryClient, err := client.NewClient(scsConfig)
	if err != nil {
		return nil, fmt.Errorf("could not initialize library client: %v", err)
	}

	return &Library{
		keystoreURI: keystoreURI,
		client:      libraryClient,
		cache:       cache,
	}, nil
}

// Pull will download the image from the library.
// After downloading, the image will be checked for a valid signature.
func (l *Library) Pull(ctx context.Context, from, to, arch string) error {
	// strip leading "library://" and append default tag, as necessary
	libraryPath := library.NormalizeLibraryRef(from)

	// check if image exists in library
	imageMeta, err := l.client.GetImage(ctx, arch, libraryPath)
	if err == scs.ErrNotFound {
		return fmt.Errorf("image %s (%s) does not exist in the library", libraryPath, arch)
	}
	if err != nil {
		return fmt.Errorf("could not get image info: %v", err)
	}

	if l.cache.IsDisabled() {
		// don't use cached image
		err := l.pullAndVerify(ctx, imageMeta, libraryPath, to, arch)
		if err != nil {
			return fmt.Errorf("unable to download image: %v", err)
		}
	} else {
		// check and use cached image
		imageName := uri.GetName("library://" + libraryPath)
		err := l.copyFromCache(imageMeta.Hash, imageName, to)
		if err != nil {
			if err != errNotInCache {
				return fmt.Errorf("could not copy image from cache: %v", err)
			}
			imagePath := l.cache.LibraryImage(imageMeta.Hash, imageName)
			err := l.pullAndVerify(ctx, imageMeta, libraryPath, imagePath, arch)
			if err != nil {
				return fmt.Errorf("could not pull image: %v", err)
			}
			err = l.copyFromCache(imageMeta.Hash, imageName, to)
			if err != nil {
				return fmt.Errorf("could not copy image from cache: %v", err)
			}
		}
	}

	_, err = signing.IsSigned(to, l.keystoreURI, 0, false, l.client.AuthToken)
	if err != nil {
		sylog.Warningf("%v", err)
		return ErrLibraryPullUnsigned
	}

	sylog.Infof("Download complete: %s\n", to)
	return nil
}

// pullAndVerify downloads library image and verifies it by comparing checksum
// in imgMeta with actual checksum of the downloaded file. The resulting image
// will be saved to the location provided.
func (l *Library) pullAndVerify(ctx context.Context, imgMeta *scs.Image, from, to, arch string) error {
	sylog.Infof("Downloading library image")
	go interruptCleanup(to)

	err := library.DownloadImage(ctx, l.client, to, arch, from, printProgress)
	if err != nil {
		return fmt.Errorf("unable to download image: %v", err)
	}

	fileHash, err := scs.ImageHash(to)
	if err != nil {
		return fmt.Errorf("error getting image hash: %v", err)
	}
	if fileHash != imgMeta.Hash {
		return fmt.Errorf("file hash(%s) and expected hash(%s) does not match", fileHash, imgMeta.Hash)
	}
	return nil
}

// copyFromCache checks whether an image with the given name and checksum exists in the cache.
// If so, the image will be copied to the location provided.
func (l *Library) copyFromCache(hash, name, to string) error {
	exists, err := l.cache.LibraryImageExists(hash, name)
	if err != nil {
		return fmt.Errorf("unable to check if %s exists: %v", name, err)
	}
	if !exists {
		return errNotInCache
	}

	from := l.cache.LibraryImage(hash, name)
	// Perms are 777 *prior* to umask in order to allow image to be
	// executed with its leading shebang like a script
	err = fs.CopyFile(from, to, 0777)
	if err != nil {
		return fmt.Errorf("while copying image from cache: %v", err)
	}
	return nil
}
