package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func deleteDir(dir string, err error) {
	if err != nil {
		os.RemoveAll(dir)
	}
}

// CreateOverlay creates a writable overlay
func CreateOverlay(bundlePath string) (err error) {
	oldumask := syscall.Umask(0)
	defer syscall.Umask(oldumask)

	overlayDir := filepath.Join(bundlePath, "overlay")
	if err = os.Mkdir(overlayDir, 0700); err != nil {
		return
	}
	// delete overlay directory in case of error
	defer deleteDir(overlayDir, err)

	if syscall.Mount(overlayDir, overlayDir, "", syscall.MS_BIND, ""); err != nil {
		return err
	}
	// best effort to cleanup mount
	defer func() {
		if err != nil {
			syscall.Unmount(overlayDir, syscall.MNT_DETACH)
		}
	}()

	if syscall.Mount("", overlayDir, "", syscall.MS_REMOUNT|syscall.MS_BIND, ""); err != nil {
		return err
	}

	upperDir := filepath.Join(overlayDir, "upper")
	if err = os.Mkdir(upperDir, 0700); err != nil {
		return
	}
	workDir := filepath.Join(overlayDir, "work")
	if err = os.Mkdir(workDir, 0700); err != nil {
		return
	}
	rootFsDir := RootFs(bundlePath).Path()

	options := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", rootFsDir, upperDir, workDir)
	if err = syscall.Mount("overlay", rootFsDir, "overlay", 0, options); err != nil {
		return
	}
	return
}

// DeleteOverlay deletes overlay
func DeleteOverlay(bundlePath string) error {
	overlayDir := filepath.Join(bundlePath, "overlay")
	rootFsDir := RootFs(bundlePath).Path()

	if err := syscall.Unmount(rootFsDir, syscall.MNT_DETACH); err != nil {
		return err
	}
	if err := syscall.Unmount(overlayDir, syscall.MNT_DETACH); err != nil {
		return err
	}
	if err := os.RemoveAll(overlayDir); err != nil {
		return err
	}
	return nil
}
