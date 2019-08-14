// +build linux

package mtree

import (
	"encoding/base64"
	"os"

	"github.com/vbatts/go-mtree/xattr"
)

func xattrUpdateKeywordFunc(path string, kv KeyVal) (os.FileInfo, error) {
	buf, err := base64.StdEncoding.DecodeString(kv.Value())
	if err != nil {
		return nil, err
	}
	if err := xattr.Set(path, kv.Keyword().Suffix(), buf); err != nil {
		return nil, err
	}
	return os.Lstat(path)
}
