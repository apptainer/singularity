/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package user

/*
#include <stdlib.h>
#include <sys/types.h>
#include <pwd.h>
#include <grp.h>
*/
import "C"
import "fmt"
import "unsafe"
import "sync"

// Passwd correspond to Go structure mapping C passwd structure
type Passwd struct {
	Name   string
	Passwd string
	UID    uint32
	GID    uint32
	Gecos  string
	Dir    string
	Shell  string
}

// Group correspond to Go structure mapping C group structure
type Group struct {
	Name   string
	Passwd string
	GID    uint32
}

func convertCPasswd(cPasswd *C.struct_passwd) *Passwd {
	return &Passwd{
		Name:   C.GoString(cPasswd.pw_name),
		Passwd: C.GoString(cPasswd.pw_passwd),
		UID:    uint32(cPasswd.pw_uid),
		GID:    uint32(cPasswd.pw_gid),
		Gecos:  C.GoString(cPasswd.pw_gecos),
		Dir:    C.GoString(cPasswd.pw_dir),
		Shell:  C.GoString(cPasswd.pw_shell),
	}
}

func convertCGroup(cGroup *C.struct_group) *Group {
	return &Group{
		Name:   C.GoString(cGroup.gr_name),
		Passwd: C.GoString(cGroup.gr_passwd),
		GID:    uint32(cGroup.gr_gid),
	}
}

var pwUIDMux sync.Mutex
var pwNamMux sync.Mutex
var grGIDMux sync.Mutex
var grNamMux sync.Mutex

// GetPwUID wraps C library getpwuid call and returns a pointer to Passwd structure
// associated with user uid
func GetPwUID(uid uint32) (*Passwd, error) {
	pwUIDMux.Lock()
	defer pwUIDMux.Unlock()

	cPasswd, _ := C.getpwuid(C.__uid_t(uid))
	if unsafe.Pointer(cPasswd) == nil {
		return nil, fmt.Errorf("can't retrieve password information for uid %d", uid)
	}

	return convertCPasswd(cPasswd), nil
}

// GetPwNam wraps C library getpwnam call and returns a pointer to Passwd structure
// associated with user name
func GetPwNam(name string) (*Passwd, error) {
	cName := C.CString(name)

	if unsafe.Pointer(cName) == nil {
		return nil, fmt.Errorf("failed to allocate memory")
	}
	defer C.free(unsafe.Pointer(cName))

	pwNamMux.Lock()
	defer pwNamMux.Unlock()

	cPasswd, _ := C.getpwnam(cName)
	if unsafe.Pointer(cPasswd) == nil {
		return nil, fmt.Errorf("can't retrieve password information for user %s", name)
	}

	return convertCPasswd(cPasswd), nil
}

// GetGrGID wraps C library getgrgid call and returns a pointer to Group structure
// associated with groud gid
func GetGrGID(gid uint32) (*Group, error) {
	grGIDMux.Lock()
	defer grGIDMux.Unlock()

	cGroup, _ := C.getgrgid(C.__gid_t(gid))
	if unsafe.Pointer(cGroup) == nil {
		return nil, fmt.Errorf("can't retrieve password information for uid %d", gid)
	}

	return convertCGroup(cGroup), nil
}

// GetGrNam wraps C library getgrnam call and returns a pointer to Group structure
// associated with group name
func GetGrNam(name string) (*Group, error) {
	cName := C.CString(name)

	if unsafe.Pointer(cName) == nil {
		return nil, fmt.Errorf("failed to allocate memory")
	}
	defer C.free(unsafe.Pointer(cName))

	grNamMux.Lock()
	defer grNamMux.Unlock()

	cGroup, _ := C.getgrnam(cName)
	if unsafe.Pointer(cGroup) == nil {
		return nil, fmt.Errorf("can't retrieve group information for group %s", name)
	}

	return convertCGroup(cGroup), nil
}
