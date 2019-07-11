package mtree

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vbatts/go-mtree/pkg/govis"
)

// UpdateKeywordFunc is the signature for a function that will restore a file's
// attributes. Where path is relative path to the file, and value to be
// restored to.
type UpdateKeywordFunc func(path string, kv KeyVal) (os.FileInfo, error)

// UpdateKeywordFuncs is the registered list of functions to update file attributes.
// Keyed by the keyword as it would show up in the manifest
var UpdateKeywordFuncs = map[Keyword]UpdateKeywordFunc{
	"mode":     modeUpdateKeywordFunc,
	"time":     timeUpdateKeywordFunc,
	"tar_time": tartimeUpdateKeywordFunc,
	"uid":      uidUpdateKeywordFunc,
	"gid":      gidUpdateKeywordFunc,
	"xattr":    xattrUpdateKeywordFunc,
	"link":     linkUpdateKeywordFunc,
}

func uidUpdateKeywordFunc(path string, kv KeyVal) (os.FileInfo, error) {
	uid, err := strconv.Atoi(kv.Value())
	if err != nil {
		return nil, err
	}

	stat, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if statIsUID(stat, uid) {
		return stat, nil
	}

	if err := os.Lchown(path, uid, -1); err != nil {
		return nil, err
	}
	return os.Lstat(path)
}

func gidUpdateKeywordFunc(path string, kv KeyVal) (os.FileInfo, error) {
	gid, err := strconv.Atoi(kv.Value())
	if err != nil {
		return nil, err
	}

	stat, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if statIsGID(stat, gid) {
		return stat, nil
	}

	if err := os.Lchown(path, -1, gid); err != nil {
		return nil, err
	}
	return os.Lstat(path)
}

func modeUpdateKeywordFunc(path string, kv KeyVal) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}

	// don't set mode on symlinks, as it passes through to the backing file
	if info.Mode()&os.ModeSymlink != 0 {
		return info, nil
	}
	vmode, err := strconv.ParseInt(kv.Value(), 8, 32)
	if err != nil {
		return nil, err
	}

	stat, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if stat.Mode() == os.FileMode(vmode) {
		return stat, nil
	}

	logrus.Debugf("path: %q, kv.Value(): %q, vmode: %o", path, kv.Value(), vmode)
	if err := os.Chmod(path, os.FileMode(vmode)); err != nil {
		return nil, err
	}
	return os.Lstat(path)
}

// since tar_time will only be second level precision, then when restoring the
// filepath from a tar_time, then compare the seconds first and only Chtimes if
// the seconds value is different.
func tartimeUpdateKeywordFunc(path string, kv KeyVal) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}

	v := strings.SplitN(kv.Value(), ".", 2)
	if len(v) != 2 {
		return nil, fmt.Errorf("expected a number like 1469104727.000000000")
	}
	sec, err := strconv.ParseInt(v[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("expected seconds, but got %q", v[0])
	}

	// if the seconds are the same, don't do anything, because the file might
	// have nanosecond value, and if using tar_time it would zero it out.
	if info.ModTime().Unix() == sec {
		return info, nil
	}

	vtime := time.Unix(sec, 0)

	// if times are same then don't modify anything
	// comparing Unix, since it does not include Nano seconds
	if info.ModTime().Unix() == vtime.Unix() {
		return info, nil
	}

	// symlinks are strange and most of the time passes through to the backing file
	if info.Mode()&os.ModeSymlink != 0 {
		if err := lchtimes(path, vtime, vtime); err != nil {
			return nil, err
		}
	} else if err := os.Chtimes(path, vtime, vtime); err != nil {
		return nil, err
	}
	return os.Lstat(path)
}

// this is nano second precision
func timeUpdateKeywordFunc(path string, kv KeyVal) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}

	v := strings.SplitN(kv.Value(), ".", 2)
	if len(v) != 2 {
		return nil, fmt.Errorf("expected a number like 1469104727.871937272")
	}
	nsec, err := strconv.ParseInt(v[0]+v[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("expected nano seconds, but got %q", v[0]+v[1])
	}
	logrus.Debugf("arg: %q; nsec: %d", v[0]+v[1], nsec)

	vtime := time.Unix(0, nsec)

	// if times are same then don't modify anything
	if info.ModTime().Equal(vtime) {
		return info, nil
	}

	// symlinks are strange and most of the time passes through to the backing file
	if info.Mode()&os.ModeSymlink != 0 {
		if err := lchtimes(path, vtime, vtime); err != nil {
			return nil, err
		}
	} else if err := os.Chtimes(path, vtime, vtime); err != nil {
		return nil, err
	}
	return os.Lstat(path)
}

func linkUpdateKeywordFunc(path string, kv KeyVal) (os.FileInfo, error) {
	linkname, err := govis.Unvis(kv.Value(), DefaultVisFlags)
	if err != nil {
		return nil, err
	}
	got, err := os.Readlink(path)
	if err != nil {
		return nil, err
	}
	if got == linkname {
		return os.Lstat(path)
	}

	logrus.Debugf("linkUpdateKeywordFunc: removing %q to link to %q", path, linkname)
	if err := os.Remove(path); err != nil {
		return nil, err
	}
	if err := os.Symlink(linkname, path); err != nil {
		return nil, err
	}

	return os.Lstat(path)
}
