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

	"github.com/hpcng/sif/pkg/sif"
	"github.com/hpcng/singularity/pkg/image"
	"golang.org/x/sys/unix"
)

const (
	mkfsBinary = "mkfs.ext3"
	ddBinary   = "dd"
)

func sifInfo(img *os.File) (string, bool, error) {
	fimg, err := sif.LoadContainerFp(img, true)
	if err != nil {
		return "", false, err
	}

	arch := string(fimg.Header.Arch[:sif.HdrArchLen-1])
	if arch == sif.HdrArchUnknown {
		arch = sif.GetSIFArch(runtime.GOARCH)
	}

	signed := false
	for _, desc := range fimg.DescrArr {
		if desc.Datatype == sif.DataSignature && desc.Link == sif.DescrDefaultGroup {
			signed = true
			break
		}
	}

	return arch, signed, fimg.UnloadContainer()
}

func OverlayCreate(size int, imgPath string, overlayDirs ...string) error {
	if size < 64 {
		return fmt.Errorf("image size must be equal or greater than 64 MiB")
	}

	mkfs, err := exec.LookPath(mkfsBinary)
	if err != nil {
		return fmt.Errorf("%s not found in $PATH", mkfsBinary)
	}
	dd, err := exec.LookPath(ddBinary)
	if err != nil {
		return fmt.Errorf("%s not found in $PATH", ddBinary)
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
	sifArch := ""

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
			arch, signed, err := sifInfo(img.File)
			if err != nil {
				return fmt.Errorf("while getting SIF info: %s", err)
			} else if signed {
				return fmt.Errorf("SIF image %s is signed: could not add writable overlay", imgPath)
			}
			sifArch = arch

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

	if err := os.Chmod(tmpFile, 0600); err != nil {
		return fmt.Errorf("while setting 0600 permission on %s: %s", tmpFile, err)
	}

	tmpDir, err := ioutil.TempDir("", "overlay-")
	if err != nil {
		return fmt.Errorf("while creating temporary overlay directory: %s", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	perm := os.FileMode(0755)

	if os.Getuid() > 65535 || os.Getgid() > 65535 {
		perm = 0777
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
		self, err := os.Executable()
		if err != nil {
			return fmt.Errorf("while determining current executable path: %s", err)
		}

		args := []string{
			"sif", "add",
			"--datatype", "4", "--partfs", "2",
			"--parttype", "4", "--partarch", sifArch,
			"--groupid", "1",
			imgPath, tmpFile,
		}
		cmd = exec.Command(self, args...)
		cmd.Stderr = errBuf
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("while adding ext3 overlay partition to %s: %s\nCommand error: %s", imgPath, err, errBuf)
		}
	} else {
		if err := os.Rename(tmpFile, imgPath); err != nil {
			return fmt.Errorf("while renaming %s to %s: %s", tmpFile, imgPath, err)
		}
	}

	return nil
}
