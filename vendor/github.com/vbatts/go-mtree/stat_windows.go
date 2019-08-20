// +build windows

package mtree

import "os"

func statIsUID(stat os.FileInfo, uid int) bool {
	return false
}
func statIsGID(stat os.FileInfo, uid int) bool {
	return false
}
