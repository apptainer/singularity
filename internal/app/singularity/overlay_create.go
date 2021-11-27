// Copyright (c) 2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.package singularity

package singularity

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hpcng/sif/v2/pkg/sif"
	"github.com/hpcng/singularity/internal/pkg/util/bin"
	"github.com/hpcng/singularity/pkg/image"
	"golang.org/x/sys/unix"
)

const (
	mkfsBinary = "mkfs.ext3"
	ddBinary   = "dd"
)

// isSigned returns true if the SIF in rw contains one or more signature objects.
func isSigned(rw sif.ReadWriter) (bool, error) {
	f, err := sif.LoadContainer(rw,
		sif.OptLoadWithFlag(os.O_RDONLY),
		sif.OptLoadWithCloseOnUnload(false),
	)
	if err != nil {
		return false, err
	}
	defer f.UnloadContainer()

	sigs, err := f.GetDescriptors(sif.WithDataType(sif.DataSignature))
	return len(sigs) > 0, err
}

// addOverlayToImage adds the EXT3 overlay at overlayPath to the SIF image at imagePath.
func addOverlayToImage(imagePath, overlayPath string) error {
	f, err := sif.LoadContainerFromPath(imagePath)
	if err != nil {
		return err
	}
	defer f.UnloadContainer()

	tf, err := os.Open(overlayPath)
	if err != nil {
		return err
	}
	defer tf.Close()

	arch := f.PrimaryArch()
	if arch == "unknown" {
		arch = runtime.GOARCH
	}

	di, err := sif.NewDescriptorInput(sif.DataPartition, tf,
		sif.OptPartitionMetadata(sif.FsExt3, sif.PartOverlay, arch),
	)
	if err != nil {
		return err
	}

	return f.AddObject(di)
}

func OverlayCreate(size int, imgPath string, overlayDirs ...string) error {
	if size < 64 {
		return fmt.Errorf("image size must be equal or greater than 64 MiB")
	}

	mkfs, err := bin.FindBin(mkfsBinary)
	if err != nil {
		return err
	}
	dd, err := bin.FindBin(ddBinary)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)

	// check if -d option is available
	cmd := exec.Command(mkfs, "--help")
	cmd.Stderr = buf
	// ignore error because the command always returns with exit code 1
	_ = cmd.Run()

	if !strings.Contains(buf.String(), "[-d ") {
		return fmt.Errorf("%s seems too old as it doesn't support -d, this is required to create the overlay layout", mkfsBinary)
	}

	sifImage := false

	if err := unix.Access(imgPath, unix.W_OK); err == nil {
		img, err := image.Init(imgPath, false)
		if err != nil {
			return fmt.Errorf("while opening image file %s: %s", imgPath, err)
		}
		switch img.Type {
		case image.SIF:
			sysPart, err := img.GetRootFsPartition()
			if err != nil {
				return fmt.Errorf("while getting root FS partition: %s", err)
			} else if sysPart.Type == image.ENCRYPTSQUASHFS {
				return fmt.Errorf("encrypted root FS partition in %s: could not add writable overlay", imgPath)
			}

			overlays, err := img.GetOverlayPartitions()
			if err != nil {
				return fmt.Errorf("while getting SIF overlay partitions: %s", err)
			}
			signed, err := isSigned(img.File)
			if err != nil {
				return fmt.Errorf("while getting SIF info: %s", err)
			} else if signed {
				return fmt.Errorf("SIF image %s is signed: could not add writable overlay", imgPath)
			}

			img.File.Close()

			for _, overlay := range overlays {
				if overlay.Type != image.EXT3 {
					continue
				}
				delCmd := fmt.Sprintf("singularity sif del %d %s", overlay.ID, imgPath)
				return fmt.Errorf("a writable overlay partition already exists in %s (ID: %d), delete it first with %q", imgPath, overlay.ID, delCmd)
			}

			sifImage = true
		case image.EXT3:
			return fmt.Errorf("EXT3 overlay image %s already exists", imgPath)
		default:
			return fmt.Errorf("destination image must be SIF image")
		}
	}

	tmpFile := imgPath + ".ext3"
	defer func() {
		_ = os.Remove(tmpFile)
	}()

	errBuf := new(bytes.Buffer)

	cmd = exec.Command(dd, "if=/dev/zero", "of="+tmpFile, "bs=1M", fmt.Sprintf("count=%d", size))
	cmd.Stderr = errBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("while zero'ing overlay image %s: %s\nCommand error: %s", tmpFile, err, errBuf)
	}
	errBuf.Reset()

	if err := os.Chmod(tmpFile, 0o600); err != nil {
		return fmt.Errorf("while setting 0600 permission on %s: %s", tmpFile, err)
	}

	tmpDir, err := ioutil.TempDir("", "overlay-")
	if err != nil {
		return fmt.Errorf("while creating temporary overlay directory: %s", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	perm := os.FileMode(0o755)

	if os.Getuid() > 65535 || os.Getgid() > 65535 {
		perm = 0o777
	}

	upperDir := filepath.Join(tmpDir, "upper")
	workDir := filepath.Join(tmpDir, "work")

	oldumask := unix.Umask(0)
	defer unix.Umask(oldumask)

	if err := os.Mkdir(upperDir, perm); err != nil {
		return fmt.Errorf("while creating %s: %s", upperDir, err)
	}
	if err := os.Mkdir(workDir, perm); err != nil {
		return fmt.Errorf("while creating %s: %s", workDir, err)
	}

	for _, dir := range overlayDirs {
		od := filepath.Join(upperDir, dir)
		if !strings.HasPrefix(od, upperDir) {
			return fmt.Errorf("overlay directory created outside of overlay layout %s", upperDir)
		}
		if err := os.MkdirAll(od, perm); err != nil {
			return fmt.Errorf("while creating %s: %s", od, err)
		}
	}

	cmd = exec.Command(mkfs, "-d", tmpDir, tmpFile)
	cmd.Stderr = errBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("while creating ext3 partition in %s: %s\nCommand error: %s", tmpFile, err, errBuf)
	}
	errBuf.Reset()

	if sifImage {
		if err := addOverlayToImage(imgPath, tmpFile); err != nil {
			return fmt.Errorf("while adding ext3 overlay partition to %s: %w", imgPath, err)
		}
	} else {
		if err := os.Rename(tmpFile, imgPath); err != nil {
			return fmt.Errorf("while renaming %s to %s: %s", tmpFile, imgPath, err)
		}
	}

	return nil
}
