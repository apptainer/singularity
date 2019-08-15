// +build darwin dragonfly freebsd openbsd linux netbsd solaris

package mtree

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func lchtimes(name string, atime time.Time, mtime time.Time) error {
	utimes := []unix.Timespec{
		unix.NsecToTimespec(atime.UnixNano()),
		unix.NsecToTimespec(mtime.UnixNano()),
	}
	if e := unix.UtimesNanoAt(unix.AT_FDCWD, name, utimes, unix.AT_SYMLINK_NOFOLLOW); e != nil {
		return &os.PathError{Op: "chtimes", Path: name, Err: e}
	}
	return nil

}
