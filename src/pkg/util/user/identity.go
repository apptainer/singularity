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

// GetPwUID wraps C library getpwuid call and returns a pointer to Passwd structure
// associated with user uid
func GetPwUID(uid uint32) (*Passwd, error) {
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

	cPasswd, _ := C.getpwnam(cName)
	if unsafe.Pointer(cPasswd) == nil {
		return nil, fmt.Errorf("can't retrieve password information for user %s", name)
	}

	return convertCPasswd(cPasswd), nil
}

// GetGrGID wraps C library getgrgid call and returns a pointer to Group structure
// associated with groud gid
func GetGrGID(gid uint32) (*Group, error) {
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

	cGroup, _ := C.getgrnam(cName)
	if unsafe.Pointer(cGroup) == nil {
		return nil, fmt.Errorf("can't retrieve group information for group %s", name)
	}

	return convertCGroup(cGroup), nil
}
