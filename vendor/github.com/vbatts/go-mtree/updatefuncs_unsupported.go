// +build !linux

package mtree

import (
	"os"
)

func xattrUpdateKeywordFunc(path string, kv KeyVal) (os.FileInfo, error) {
	return os.Lstat(path)
}
