package mtree

import (
	"archive/tar"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"os"

	"github.com/vbatts/go-mtree/pkg/govis"
	"golang.org/x/crypto/ripemd160"
)

// KeywordFunc is the type of a function called on each file to be included in
// a DirectoryHierarchy, that will produce the string output of the keyword to
// be included for the file entry. Otherwise, empty string.
// io.Reader `r` is to the file stream for the file payload. While this
// function takes an io.Reader, the caller needs to reset it to the beginning
// for each new KeywordFunc
type KeywordFunc func(path string, info os.FileInfo, r io.Reader) ([]KeyVal, error)

var (
	// KeywordFuncs is the map of all keywords (and the functions to produce them)
	KeywordFuncs = map[Keyword]KeywordFunc{
		"size":            sizeKeywordFunc,                                      // The size, in bytes, of the file
		"type":            typeKeywordFunc,                                      // The type of the file
		"time":            timeKeywordFunc,                                      // The last modification time of the file
		"link":            linkKeywordFunc,                                      // The target of the symbolic link when type=link
		"uid":             uidKeywordFunc,                                       // The file owner as a numeric value
		"gid":             gidKeywordFunc,                                       // The file group as a numeric value
		"nlink":           nlinkKeywordFunc,                                     // The number of hard links the file is expected to have
		"uname":           unameKeywordFunc,                                     // The file owner as a symbolic name
		"gname":           gnameKeywordFunc,                                     // The file group as a symbolic name
		"mode":            modeKeywordFunc,                                      // The current file's permissions as a numeric (octal) or symbolic value
		"cksum":           cksumKeywordFunc,                                     // The checksum of the file using the default algorithm specified by the cksum(1) utility
		"md5":             hasherKeywordFunc("md5digest", md5.New),              // The MD5 message digest of the file
		"md5digest":       hasherKeywordFunc("md5digest", md5.New),              // A synonym for `md5`
		"rmd160":          hasherKeywordFunc("ripemd160digest", ripemd160.New),  // The RIPEMD160 message digest of the file
		"rmd160digest":    hasherKeywordFunc("ripemd160digest", ripemd160.New),  // A synonym for `rmd160`
		"ripemd160digest": hasherKeywordFunc("ripemd160digest", ripemd160.New),  // A synonym for `rmd160`
		"sha1":            hasherKeywordFunc("sha1digest", sha1.New),            // The SHA1 message digest of the file
		"sha1digest":      hasherKeywordFunc("sha1digest", sha1.New),            // A synonym for `sha1`
		"sha256":          hasherKeywordFunc("sha256digest", sha256.New),        // The SHA256 message digest of the file
		"sha256digest":    hasherKeywordFunc("sha256digest", sha256.New),        // A synonym for `sha256`
		"sha384":          hasherKeywordFunc("sha384digest", sha512.New384),     // The SHA384 message digest of the file
		"sha384digest":    hasherKeywordFunc("sha384digest", sha512.New384),     // A synonym for `sha384`
		"sha512":          hasherKeywordFunc("sha512digest", sha512.New),        // The SHA512 message digest of the file
		"sha512digest":    hasherKeywordFunc("sha512digest", sha512.New),        // A synonym for `sha512`
		"sha512256":       hasherKeywordFunc("sha512digest", sha512.New512_256), // The SHA512/256 message digest of the file
		"sha512256digest": hasherKeywordFunc("sha512digest", sha512.New512_256), // A synonym for `sha512256`

		"flags": flagsKeywordFunc, // NOTE: this is a noop, but here to support the presence of the "flags" keyword.

		// This is not an upstreamed keyword, but used to vary from "time", as tar
		// archives do not store nanosecond precision. So comparing on "time" will
		// be only seconds level accurate.
		"tar_time": tartimeKeywordFunc, // The last modification time of the file, from a tar archive mtime

		// This is not an upstreamed keyword, but a needed attribute for file validation.
		// The pattern for this keyword key is prefixed by "xattr." followed by the extended attribute "namespace.key".
		// The keyword value is the SHA1 digest of the extended attribute's value.
		// In this way, the order of the keys does not matter, and the contents of the value is not revealed.
		"xattr":  xattrKeywordFunc,
		"xattrs": xattrKeywordFunc,
	}
)
var (
	modeKeywordFunc = func(path string, info os.FileInfo, r io.Reader) ([]KeyVal, error) {
		permissions := info.Mode().Perm()
		if os.ModeSetuid&info.Mode() > 0 {
			permissions |= (1 << 11)
		}
		if os.ModeSetgid&info.Mode() > 0 {
			permissions |= (1 << 10)
		}
		if os.ModeSticky&info.Mode() > 0 {
			permissions |= (1 << 9)
		}
		return []KeyVal{KeyVal(fmt.Sprintf("mode=%#o", permissions))}, nil
	}
	sizeKeywordFunc = func(path string, info os.FileInfo, r io.Reader) ([]KeyVal, error) {
		if sys, ok := info.Sys().(*tar.Header); ok {
			if sys.Typeflag == tar.TypeSymlink {
				return []KeyVal{KeyVal(fmt.Sprintf("size=%d", len(sys.Linkname)))}, nil
			}
		}
		return []KeyVal{KeyVal(fmt.Sprintf("size=%d", info.Size()))}, nil
	}
	cksumKeywordFunc = func(path string, info os.FileInfo, r io.Reader) ([]KeyVal, error) {
		if !info.Mode().IsRegular() {
			return nil, nil
		}
		sum, _, err := cksum(r)
		if err != nil {
			return nil, err
		}
		return []KeyVal{KeyVal(fmt.Sprintf("cksum=%d", sum))}, nil
	}
	hasherKeywordFunc = func(name string, newHash func() hash.Hash) KeywordFunc {
		return func(path string, info os.FileInfo, r io.Reader) ([]KeyVal, error) {
			if !info.Mode().IsRegular() {
				return nil, nil
			}
			h := newHash()
			if _, err := io.Copy(h, r); err != nil {
				return nil, err
			}
			return []KeyVal{KeyVal(fmt.Sprintf("%s=%x", KeywordSynonym(name), h.Sum(nil)))}, nil
		}
	}
	tartimeKeywordFunc = func(path string, info os.FileInfo, r io.Reader) ([]KeyVal, error) {
		return []KeyVal{KeyVal(fmt.Sprintf("tar_time=%d.%9.9d", info.ModTime().Unix(), 0))}, nil
	}
	timeKeywordFunc = func(path string, info os.FileInfo, r io.Reader) ([]KeyVal, error) {
		tSec := info.ModTime().Unix()
		tNano := info.ModTime().Nanosecond()
		return []KeyVal{KeyVal(fmt.Sprintf("time=%d.%9.9d", tSec, tNano))}, nil
	}
	linkKeywordFunc = func(path string, info os.FileInfo, r io.Reader) ([]KeyVal, error) {
		if sys, ok := info.Sys().(*tar.Header); ok {
			if sys.Linkname != "" {
				linkname, err := govis.Vis(sys.Linkname, DefaultVisFlags)
				if err != nil {
					return nil, nil
				}
				return []KeyVal{KeyVal(fmt.Sprintf("link=%s", linkname))}, nil
			}
			return nil, nil
		}

		if info.Mode()&os.ModeSymlink != 0 {
			str, err := os.Readlink(path)
			if err != nil {
				return nil, nil
			}
			linkname, err := govis.Vis(str, DefaultVisFlags)
			if err != nil {
				return nil, nil
			}
			return []KeyVal{KeyVal(fmt.Sprintf("link=%s", linkname))}, nil
		}
		return nil, nil
	}
	typeKeywordFunc = func(path string, info os.FileInfo, r io.Reader) ([]KeyVal, error) {
		if info.Mode().IsDir() {
			return []KeyVal{"type=dir"}, nil
		}
		if info.Mode().IsRegular() {
			return []KeyVal{"type=file"}, nil
		}
		if info.Mode()&os.ModeSocket != 0 {
			return []KeyVal{"type=socket"}, nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return []KeyVal{"type=link"}, nil
		}
		if info.Mode()&os.ModeNamedPipe != 0 {
			return []KeyVal{"type=fifo"}, nil
		}
		if info.Mode()&os.ModeDevice != 0 {
			if info.Mode()&os.ModeCharDevice != 0 {
				return []KeyVal{"type=char"}, nil
			}
			return []KeyVal{"type=block"}, nil
		}
		return nil, nil
	}
)
