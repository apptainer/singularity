// +build !linux,!darwin,!freebsd,!netbsd,!openbsd

package mtree

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
)

var (
	// this is bsd specific https://www.freebsd.org/cgi/man.cgi?query=chflags&sektion=2
	flagsKeywordFunc = func(path string, info os.FileInfo, r io.Reader) ([]KeyVal, error) {
		return nil, nil
	}
	unameKeywordFunc = func(path string, info os.FileInfo, r io.Reader) ([]KeyVal, error) {
		if hdr, ok := info.Sys().(*tar.Header); ok {
			return []KeyVal{KeyVal(fmt.Sprintf("uname=%s", hdr.Uname))}, nil
		}
		return nil, nil
	}
	gnameKeywordFunc = func(path string, info os.FileInfo, r io.Reader) ([]KeyVal, error) {
		if hdr, ok := info.Sys().(*tar.Header); ok {
			return []KeyVal{KeyVal(fmt.Sprintf("gname=%s", hdr.Gname))}, nil
		}
		return nil, nil
	}
	uidKeywordFunc = func(path string, info os.FileInfo, r io.Reader) ([]KeyVal, error) {
		if hdr, ok := info.Sys().(*tar.Header); ok {
			return []KeyVal{KeyVal(fmt.Sprintf("uid=%d", hdr.Uid))}, nil
		}
		return nil, nil
	}
	gidKeywordFunc = func(path string, info os.FileInfo, r io.Reader) ([]KeyVal, error) {
		if hdr, ok := info.Sys().(*tar.Header); ok {
			return []KeyVal{KeyVal(fmt.Sprintf("gid=%d", hdr.Gid))}, nil
		}
		return nil, nil
	}
	nlinkKeywordFunc = func(path string, info os.FileInfo, r io.Reader) ([]KeyVal, error) {
		return nil, nil
	}
	xattrKeywordFunc = func(path string, info os.FileInfo, r io.Reader) ([]KeyVal, error) {
		return nil, nil
	}
)
