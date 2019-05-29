package image

import (
	"fmt"
	"os"

	"github.com/sylabs/singularity/internal/pkg/sylog"
)

type encryptFSFormat struct{}

func checkEncryptfsHeader(b []byte) (uint64, error) {
	return 0, nil
}

func (f *encryptFSFormat) initializer(img *Image, fileinfo os.FileInfo) error {

	sylog.Debugf("EncryptFS Initializer")

	if fileinfo.IsDir() {
		return fmt.Errorf("not an encryptfs image")
	}
	b := make([]byte, bufferSize)
	if n, err := img.File.Read(b); err != nil || n != bufferSize {
		return fmt.Errorf("can't read first %d bytes: %s", bufferSize, err)
	}
	offset, err := checkEncryptfsHeader(b)
	if err != nil {
		return err
	}
	img.Type = ENCRYPTFS
	img.Partitions = []Section{
		{
			Offset: offset,
			Size:   uint64(fileinfo.Size()) - offset,
			Type:   ENCRYPTFS,
			Name:   RootFs,
		},
	}

	return nil
}

func (f *encryptFSFormat) openMode(writable bool) int {
	if writable {
		return os.O_RDWR
	}
	return os.O_RDONLY
}
