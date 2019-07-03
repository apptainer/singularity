package singularity

import (
	"errors"
	"fmt"

	"github.com/sylabs/singularity/internal/pkg/client/cache"
)

var (
	errInvalidCacheHandle = errors.New("invalid cache handle")
	errInvalidCacheType   = errors.New("invalid cache type")
)

func normalizeCacheList(cacheList []string) ([]string, error) {
	all := false
	list := []string{}

	for _, e := range cacheList {
		switch e {
		case "library", "oci", "shub", "blob", "net", "oras":
			list = append(list, e)

		case "blobs":
			list = append(list, "blob")

		case "all":
			// cacheList contains "all", fall back to all
			// entries, but continue validating entries just
			// to be on the safe side
			all = true

		default:
			return nil, fmt.Errorf("cache value %s: %+v", e, errInvalidCacheType)
		}
	}

	if all {
		// cleanAll overrides all the specified names
		list = []string{"library", "oci", "shub", "blob", "net", "oras"}
	}

	return list, nil
}

func cacheTypeToDir(imgCache *cache.Handle, cacheType string) (string, error) {
	switch cacheType {
	case "library":
		return imgCache.Library, nil
	case "oci":
		return imgCache.OciTemp, nil
	case "shub":
		return imgCache.Shub, nil
	case "blob":
		return imgCache.OciBlob, nil
	case "net":
		return imgCache.Net, nil
	case "oras":
		return imgCache.Oras, nil
	}

	return "", errInvalidCacheType
}
