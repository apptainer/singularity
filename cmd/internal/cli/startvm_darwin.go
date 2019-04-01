// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

func getHypervisorArgs(sifImage, bzImage, initramfs, singAction, cliExtra string) []string {
	// Setup some needed variables
	hdString := fmt.Sprintf("2:0,ahci-hd,%s", sifImage)

	// Default xhyve Arguments
	args := []string{
		filepath.Join(buildcfg.LIBEXECDIR, "/singularity/vm/xhyve"),
		"-A",
		"-m", VMRAM,
		"-c", VMCPU,
		"-s", "0:0,hostbridge",
		"-s", hdString,
		"-s", "31,lpc",
		"-l", "com1,stdio",
	}

	// Bind mounts
	singBinds := []string{""}

	// Set slot to 26. slot has a max value of 31, so this will give us a max of 48 bind mounts from the Mac host.
	slot := 26
	function := 0

	for _, bindpath := range BindPaths {
		// If we somehow surpass slot 31 ... error. We can't do anything at this point.
		if slot > 31 {
			sylog.Fatalf("Surpassed max amount of binds we can pass to virtual machine")
		}

		splitted := strings.Split(bindpath, ":")
		src := splitted[0]
		dst := ""
		if len(splitted) > 1 {
			dst = splitted[1]
		} else {
			dst = src
		}

		sylog.Debugf("Bind path: " + src + " -> " + dst)
		// TODO: Figure out if src is a directory or not
		mntTag := filepath.Base(src)

		pciArgs := fmt.Sprintf("%s:%s,virtio-9p,%s=%s", strconv.Itoa(slot), strconv.Itoa(function), mntTag, src)
		args = append(args, "-s", pciArgs)

		localBind := fmt.Sprintf("%s:%s", mntTag, dst)
		singBinds = append(singBinds, localBind)

		sylog.Debugf("PCI: %s", pciArgs)

		// The PCI function can be a value from 0-7 per slot. If we have more than 8 binds, increase the slot, 
		// and reset the function value back to 0
		function++
		if function > 7 {
			slot++
			function = 0
		}
	}

	usr, err := user.Current()
	if err != nil {
		sylog.Fatalf("Failed to get current user")
	}

	// Force $HOME to be mounted
	// TODO: engineConfig.GetHomeSource() / GetHomeDest() -- should probably be used
	homeSrc := usr.HomeDir
	pciArgs := fmt.Sprintf("4:0,virtio-9p,home=%s", homeSrc)
	homeBind := fmt.Sprintf("home:%s", homeSrc)
	singBinds = append(singBinds, homeBind)

	sylog.Debugf("PCI: %s", pciArgs)
	args = append(args, "-s", pciArgs)

	userInfo := fmt.Sprintf("%s:%s:%s", usr.Username, usr.Uid, usr.Gid)

	if IsSyOS {
		// We're ignoring anything passed since we want a SyOS
		// shell ... We aren't going into the image
		// automatically here.

		cliExtra = "syos"
	}

	// Get our CWD and pass it along
	cwdDir, err := os.Getwd()
	if err != nil {
		sylog.Fatalf("Error getting working directory: %s", err)
	}

	kexecArgs := fmt.Sprintf("kexec,%s,%s,console=ttyS0 quiet root=/dev/ram0 loglevel=0 sing_img_name=%s sing_user=%s sing_cwd=%s singularity_action=%s singularity_arguments=\"%s\" singularity_binds=\"%v\"", bzImage, initramfs, filepath.Base(sifImage), userInfo, cwdDir, singAction, cliExtra, strings.Join(singBinds, "|"))

	// Add our actual kexec entry
	args = append(args, "-f", kexecArgs)

	return args
}
