package fs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"syscall"

	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/user"
)

var overlayDir syscall.Stat_t
var finalDir syscall.Stat_t
var sessionDir syscall.Stat_t

// ContainerDirUpdate updates stat structure to match mountpoints where file/directory creation are authorized
func ContainerDirUpdate(sessionOnly bool) error {
	if sessionOnly == false {
		if info, err := os.Stat(buildcfg.CONTAINER_OVERLAY); err != nil {
			return err
		} else {
			overlayDir = *info.Sys().(*syscall.Stat_t)
		}
		if info, err := os.Stat(buildcfg.CONTAINER_FINALDIR); err != nil {
			return err
		} else {
			finalDir = *info.Sys().(*syscall.Stat_t)
		}
	}
	if info, err := os.Stat(buildcfg.SESSIONDIR); err != nil {
		return err
	} else {
		sessionDir = *info.Sys().(*syscall.Stat_t)
	}
	return nil
}

// ContainerMkdir creates directory inside authorized container path defined by ContainerDirUpdate
func ContainerMkdir(dir string, mode os.FileMode) error {
	var err error
	var joinedPath = "/"

	euid := os.Geteuid()
	uid := os.Getuid()

	/* check if caller is RPC server */
	privileged := (euid != uid)

	current, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("can't obtain current working directory: %s", err)
	}

	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("can't go into / directory: %s", err)
	}
	defer os.Chdir(current)

	for _, p := range strings.Split(dir, "/")[1:] {
		joinedPath = path.Join(joinedPath, p)
		if err := os.Chdir(joinedPath); err != nil {
			info, err := os.Stat(".")
			if err != nil {
				return fmt.Errorf("can't retrieve stat information for %s directory: %s", joinedPath, err)
			}
			sys := info.Sys().(*syscall.Stat_t)
			if sys.Dev != overlayDir.Dev && sys.Dev != finalDir.Dev && sys.Dev != sessionDir.Dev {
				return fmt.Errorf("trying to create directory %s outside of container in %s", p, path.Join(joinedPath, ".."))
			}
			/* modify setfsuid for NFS root_squash issues */
			if privileged {
				syscall.Setfsuid(0)
			}
			if err := os.Mkdir(p, mode); err != nil {
				if privileged {
					syscall.Setfsuid(uid)
				}
				return fmt.Errorf("can't create directory %s: %s", p, err)
			}
			if privileged {
				syscall.Setfsuid(uid)
			}
		}
	}
	return nil
}

// ContainerFile creates file with provided content inside container
func ContainerFile(name string, content []byte) error {
	euid := os.Geteuid()
	uid := os.Getuid()
	dirname := path.Dir(name)
	basename := path.Base(name)

	privileged := (euid != uid)

	current, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("can't obtain current working directory: %s", err)
	}
	defer os.Chdir(current)

	if err := os.Chdir(dirname); err != nil {
		if os.IsNotExist(err) {
			if err := ContainerMkdir(dirname, 0755); err != nil {
				return fmt.Errorf("failed to create parent directory %s: %s", dirname, err)
			}
			if err := os.Chdir(dirname); err != nil {
				return fmt.Errorf("failed to go into parent directory %s: %s", dirname, err)
			}
		} else {
			return fmt.Errorf("failed to go into parent directory %s: %s", dirname, err)
		}
	}

	info, err := os.Stat(".")
	if err != nil {
		return fmt.Errorf("can't retrieve stat information for %s directory: %s", dirname, err)
	}

	sys := info.Sys().(*syscall.Stat_t)
	if sys.Dev != overlayDir.Dev && sys.Dev != finalDir.Dev && sys.Dev != sessionDir.Dev {
		return fmt.Errorf("trying to create file %s outside of container in %s", basename, dirname)
	}

	flags := syscall.O_CREAT | syscall.O_WRONLY | syscall.O_TRUNC | syscall.O_NOFOLLOW

	if privileged {
		syscall.Setfsuid(0)
	}

	file, err := os.OpenFile(basename, flags, 0644)
	if err != nil {
		if privileged {
			syscall.Setfsuid(uid)
		}
		return fmt.Errorf("failed to create file %s: %s", name, err)
	}

	if len(content) > 0 {
		file.Write(content)
	}
	file.Close()

	if privileged {
		syscall.Setfsuid(uid)
	}

	return nil
}

// ContainerPasswd creates a staging passwd file with container /etc/passwd and current user information
// inside session directory and returns file path
func ContainerPasswd() (string, error) {
	var content []byte

	passwdContainer := path.Join(buildcfg.CONTAINER_MOUNTDIR, "etc/passwd")
	passwdSession := path.Join(buildcfg.SESSIONDIR, "passwd")

	sylog.Verbosef("Checking for template passwd file: %s\n", passwdContainer)
	if IsFile(passwdContainer) == false {
		return "", fmt.Errorf("passwd file doesn't exist in container, not updating")
	}

	sylog.Verbosef("Creating template of /etc/passwd\n")
	passwdFile, err := os.Open(passwdContainer)
	if err != nil {
		return "", fmt.Errorf("failed to open passwd file in container: %s", err)
	}
	defer passwdFile.Close()

	content, err = ioutil.ReadAll(passwdFile)
	if err != nil {
		return "", fmt.Errorf("failed to read passwd file content in container: %s", err)
	}

	pwInfo, err := user.GetPwUID(uint32(os.Getuid()))
	if err != nil {
		return "", err
	}

	// TODO: consider using value from SINGULARITY_HOME once we get something ala singularity_registry
	userInfo := fmt.Sprintf("%s:x:%d:%d:%s:%s:%s\n", pwInfo.Name, pwInfo.UID, pwInfo.GID, pwInfo.Gecos, pwInfo.Dir, pwInfo.Shell)

	if content[len(content)-1] != '\n' {
		content = append(content, byte('\n'))
	}

	sylog.Verbosef("Creating template passwd file and appending user data: %s\n", passwdSession)
	content = append(content, []byte(userInfo)...)

	// TODO: move ContainerDirUpdate into overlay mount code when implemented
	ContainerDirUpdate(false)
	if err := ContainerFile(passwdSession, content); err != nil {
		return "", fmt.Errorf("failed to append user group information to %s: %s", passwdSession, err)
	}
	return passwdSession, nil
}

// ContainerGroup creates a staging group file with container /etc/group and current user group information
// inside session directory and returns file path
func ContainerGroup() (string, error) {
	var content []byte
	duplicate := false

	groupContainer := path.Join(buildcfg.CONTAINER_MOUNTDIR, "etc/group")
	groupSession := path.Join(buildcfg.SESSIONDIR, "group")

	if IsFile(groupContainer) == false {
		return "", fmt.Errorf("group file doesn't exist in container, not updating")
	}
	groupFile, err := os.Open(groupContainer)
	if err != nil {
		return "", fmt.Errorf("failed to open group file in container: %s", err)
	}
	defer groupFile.Close()

	pwInfo, err := user.GetPwUID(uint32(os.Getuid()))
	if err != nil || pwInfo == nil {
		return "", err
	}
	grInfo, err := user.GetGrGID(pwInfo.GID)
	if err != nil || grInfo == nil {
		return "", err
	}
	groups, err := os.Getgroups()
	if err != nil {
		return "", err
	}
	for _, gid := range groups {
		if gid == int(pwInfo.GID) {
			duplicate = true
			break
		}
	}
	if duplicate == false {
		groups = append(groups, int(pwInfo.GID))
	}
	content, err = ioutil.ReadAll(groupFile)
	if err != nil {
		return "", fmt.Errorf("failed to read group file content in container: %s", err)
	}

	if content[len(content)-1] != '\n' {
		content = append(content, byte('\n'))
	}

	for _, gid := range groups {
		grInfo, err := user.GetGrGID(uint32(gid))
		if err != nil || grInfo == nil {
			sylog.Verbosef("Skipping GID %d as group entry doesn't exist.\n")
			continue
		}
		groupLine := fmt.Sprintf("%s:x:%d:%s\n", grInfo.Name, grInfo.GID, pwInfo.Name)
		content = append(content, []byte(groupLine)...)
	}

	if err := ContainerFile(groupSession, content); err != nil {
		return "", fmt.Errorf("failed to append user group information to %s: %s", groupSession, err)
	}
	return groupSession, nil
}

// ContainerHostname creates a staging hostname file with provided hostname inside session directory and
// returns file path
func ContainerHostname(hostname string) (string, error) {
	hostnameSession := path.Join(buildcfg.SESSIONDIR, "hostname")

	content := []byte(hostname + "\n")

	if err := ContainerFile(hostnameSession, content); err != nil {
		return "", fmt.Errorf("can't create staging hostname file: %s", err)
	}
	return hostnameSession, nil
}

// ContainerResolvConf creates a staging hostname file with provided hostname inside session directory and
// returns file path
func ContainerResolvConf(content []byte) (string, error) {
	resolvConfSession := path.Join(buildcfg.SESSIONDIR, "resolv.conf")

	if err := ContainerFile(resolvConfSession, content); err != nil {
		return "", fmt.Errorf("can't create staging resolv.conf file: %s", err)
	}
	return resolvConfSession, nil
}
