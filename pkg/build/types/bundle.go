// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package types

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	ocitypes "github.com/containers/image/types"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/util/loop"
	"golang.org/x/crypto/ssh/terminal"
)

// Bundle is the temporary build environment used during the image
// building process. A Bundle is the programmatic representation of
// the directory structure which will constitute this environmenb.
// /tmp/...:
//     fs/ - A chroot filesystem
//     .singularity.d/ - Container metadata (from 2.x image format)
//     config.json (optional) - Contain information for OCI image bundle
//     etc... - The Bundle dir can theoretically contain arbitrary directories,
//              files, etc... which can be interpreted by the Chef
type Bundle struct {
	// FSObjects is a map of the filesystem objects contained in the Bundle. An object
	// will be built as one section of a SIF file.
	//
	// Known FSObjects labels:
	//   * rootfs -> root file system
	//   * .singularity.d -> .singularity.d directory (includes image exec scripts)
	//   * data -> directory containing data files
	FSObjects   map[string]string `json:"fsObjects"`
	JSONObjects map[string][]byte `json:"jsonObjects"`
	Recipe      Definition        `json:"rawDeffile"`
	BindPath    []string          `json:"bindPath"`
	Path        string            `json:"bundlePath"`
	Opts        Options           `json:"opts"`
	LoopPath    string            `json:"loopPath"`
}

// Options defines build time behavior to be executed on the bundle
type Options struct {
	// Encrypt specifies if the filesystem needs to be encrypted
	Encrypt bool `json:"encrypt"`
	// sections are the parts of the definition to run during the build
	Sections []string `json:"sections"`
	// TmpDir specifies a non-standard temporary location to perform a build
	TmpDir string
	// LibraryURL contains URL to library where base images can be pulled
	LibraryURL string `json:"libraryURL"`
	// LibraryAuthToken contains authentication token to access specified library
	LibraryAuthToken string `json:"libraryAuthToken"`
	// contains docker credentials if specified
	DockerAuthConfig *ocitypes.DockerAuthConfig
	// noTest indicates if build should skip running the test script
	NoTest bool `json:"noTest"`
	// force automatically deletes an existing container at build destination while performing build
	Force bool `json:"force"`
	// update detects and builds using an existing sandbox container at build destination
	Update bool `json:"update"`
	// noHTTPS
	NoHTTPS bool `json:"noHTTPS"`
	// NoCleanUp allows a user to prevent a bundle from being cleaned up after a failed build
	// useful for debugging
	NoCleanUp bool `json:"noCleanUp"`
}

func createLoop(file *os.File, offset, size uint64) (string, error) {
	loopDev := &loop.Device{
		MaxLoopDevices: 256,
		Shared:         true,
		Info: &loop.Info64{
			SizeLimit: size,
			Offset:    offset,
			Flags:     loop.FlagsAutoClear,
		},
	}
	idx := 0
	if err := loopDev.AttachFromFile(file, os.O_RDWR, &idx); err != nil {
		return "", fmt.Errorf("failed to attach image %s: %s", file.Name(), err)
	}
	return fmt.Sprintf("/dev/loop%d", idx), nil
}

// NewBundle creates a Bundle environment
func NewBundle(encrypt bool, bundleDir, bundlePrefix string) (b *Bundle, err error) {
	b = &Bundle{}
	b.JSONObjects = make(map[string][]byte)

	if bundlePrefix == "" {
		bundlePrefix = "sbuild-"
	}

	b.Path, err = ioutil.TempDir(bundleDir, bundlePrefix+"-")
	if err != nil {
		return nil, err
	}
	sylog.Debugf("Created temporary directory for bundle %v\n", b.Path)

	b.FSObjects = map[string]string{
		"rootfs": "fs",
	}

	var input = "Default"
	if encrypt == true {
		// Read the password from terminal
		fmt.Print("Enter a password to encrypt the filesystem: ")
		password, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			sylog.Fatalf("Error parsing input: %s", err)
		}
		input = string(password)

		// Create a sparse file in tmp dir
		f, err := os.Create(b.Path + "/sparse_fs.loop")
		if err != nil {
			return nil, err
		}

		// Create a 500MB sparse file
		err = f.Truncate(5 * 1e8)
		if err != nil {
			return nil, err
		}

		file, err := os.OpenFile(b.Path+"/sparse_fs.loop", os.O_RDWR, 0755)
		defer file.Close()

		// Associate the above created file with a loop device
		loop, err := createLoop(file, 0, 5*1e8)

		b.LoopPath = loop
		sp := strings.Split(loop, "/")
		loopdev := sp[len(sp)-1]
		cmd := exec.Command("cryptsetup", "luksFormat", loopdev)
		cmd.Dir = "/dev"
		stdin, err := cmd.StdinPipe()

		go func() {
			defer stdin.Close()
			io.WriteString(stdin, input)
		}()

		out, err := cmd.CombinedOutput()
		if err != nil {
			sylog.Verbosef("Out is %s, err is %s", out, err)
			return nil, err
		}

		cmd = exec.Command("cryptsetup", "--disable-locks", "luksOpen", loopdev, "sycrypt")
		cmd.Dir = "/dev"
		stdin, err = cmd.StdinPipe()

		go func() {
			defer stdin.Close()
			io.WriteString(stdin, input)
		}()

		out, err = cmd.CombinedOutput()
		if err != nil {
			sylog.Verbosef("Out is %s, err is %s", out, err)
			return nil, err
		}

		// Create an EXT3 FS in the mapped device
		cmd = exec.Command("mkfs.ext3", "sycrypt")
		cmd.Dir = "/dev/mapper"
		out, err = cmd.CombinedOutput()
		if err != nil {
			sylog.Verbosef("Out is %s, err is %s", out, err)
			return nil, err
		}
	}

	for _, fso := range b.FSObjects {
		if err = os.MkdirAll(filepath.Join(b.Path, fso), 0755); err != nil {
			return
		}
		if encrypt {
			err = syscall.Mount("/dev/mapper/sycrypt", b.Rootfs(), "ext3", syscall.MS_NOSUID, "")
			if err != nil {
				sylog.Debugf("Unable to mount err: %s", err)
				time.Sleep(time.Minute)
			}
			err = syscall.Rmdir(b.Rootfs() + "/lost+found")
			if err != nil {
				sylog.Debugf("Unable to mount err: %s", err)
				time.Sleep(time.Minute)
			}
		}
	}

	return b, nil
}

// Rootfs give the path to the root filesystem in the Bundle
func (b *Bundle) Rootfs() string {
	sylog.Debugf("Rootfs is %s", filepath.Join(b.Path, b.FSObjects["rootfs"]))
	return filepath.Join(b.Path, b.FSObjects["rootfs"])
}

// RunSection iterates through the sections specified in a bundle
// and returns true if the given string, s, is a section of the
// definition that should be executed during the build process
func (b Bundle) RunSection(s string) bool {
	for _, section := range b.Opts.Sections {
		if section == "none" {
			return false
		}
		if section == "all" || section == s {
			return true
		}
	}
	return false
}
