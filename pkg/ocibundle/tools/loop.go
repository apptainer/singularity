package tools

import (
	"fmt"
	"os"

	"github.com/sylabs/singularity/pkg/util/loop"
)

// CreateLoop ...
func CreateLoop(file *os.File, offset, size uint64) (string, error) {
	loopDev := &loop.Device{
		MaxLoopDevices: 256,
		Shared:         true,
		Info: &loop.Info64{
			SizeLimit: size,
			Offset:    offset,
			Flags:     loop.FlagsAutoClear,
		},
	}
	idx := 0
	if err := loopDev.AttachFromFile(file, os.O_RDONLY, &idx); err != nil {
		return "", err
	}
	return fmt.Sprintf("/dev/loop%d", idx), nil
}
