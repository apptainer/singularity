package mtree

import "os"

// FsEval is a mock-friendly method of specifying to go-mtree how to carry out
// filesystem operations such as opening files and the like. The semantics of
// all of these wrappers MUST be identical to the semantics described here.
type FsEval interface {
	// Open must have the same semantics as os.Open.
	Open(path string) (*os.File, error)

	// Lstat must have the same semantics as os.Lstat.
	Lstat(path string) (os.FileInfo, error)

	// Readdir must have the same semantics as calling os.Open on the given
	// path and then returning the result of (*os.File).Readdir(-1).
	Readdir(path string) ([]os.FileInfo, error)

	// KeywordFunc must return a wrapper around the provided function (in other
	// words, the returned function must refer to the same keyword).
	KeywordFunc(fn KeywordFunc) KeywordFunc
}

// DefaultFsEval is the default implementation of FsEval (and is the default
// used if a nil interface is passed to any mtree function). It does not modify
// or wrap any of the methods (they all just call out to os.*).
type DefaultFsEval struct{}

// Open must have the same semantics as os.Open.
func (fs DefaultFsEval) Open(path string) (*os.File, error) {
	return os.Open(path)
}

// Lstat must have the same semantics as os.Lstat.
func (fs DefaultFsEval) Lstat(path string) (os.FileInfo, error) {
	return os.Lstat(path)
}

// Readdir must have the same semantics as calling os.Open on the given
// path and then returning the result of (*os.File).Readdir(-1).
func (fs DefaultFsEval) Readdir(path string) ([]os.FileInfo, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	return fh.Readdir(-1)
}

// KeywordFunc must return a wrapper around the provided function (in other
// words, the returned function must refer to the same keyword).
func (fs DefaultFsEval) KeywordFunc(fn KeywordFunc) KeywordFunc {
	return fn
}
