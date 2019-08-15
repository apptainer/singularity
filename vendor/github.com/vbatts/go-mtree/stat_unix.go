// +build !windows

package mtree

import (
	"os"
	"syscall"
)

func statIsUID(stat os.FileInfo, uid int) bool {
	statT := stat.Sys().(*syscall.Stat_t)
	return statT.Uid == uint32(uid)
}

func statIsGID(stat os.FileInfo, gid int) bool {
	statT := stat.Sys().(*syscall.Stat_t)
	return statT.Gid == uint32(gid)
}
