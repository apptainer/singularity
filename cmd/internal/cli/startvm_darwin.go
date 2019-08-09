// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"math/rand"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

// Lots of love from:
// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
func genString(n int) string {
	validChar := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	s := make([]rune, n)
	for i := range s {
		s[i] = validChar[rand.Intn(len(validChar))]
	}
	return string(s)
}

func getHypervisorArgs(sifImage, bzImage, initramfs, singAction, cliExtra string) []string {
	// Seed on call to getHypervisorArgs()
	rand.Seed(time.Now().UnixNano())

	// Setup some needed variables
	hdString := fmt.Sprintf("2:0,ahci-hd,%s", sifImage)

	// Check xhyve permissions
	fi, err := os.Stat(filepath.Join(buildcfg.LIBEXECDIR, "/singularity/vm/xhyve"))
	if err != nil {
		sylog.Fatalf("Filed to Stat() xhyve binary: %v", err)
	}
	setuid := false
	if fi.Mode() & os.ModeSetuid == os.ModeSetuid {
		setuid = true
	}
	// Default xhyve Arguments
	args := []string{
		filepath.Join(buildcfg.LIBEXECDIR, "/singularity/vm/xhyve"),
		"-A",
		"-m", VMRAM,
		"-c", VMCPU,
		"-s", "0:0,hostbridge",
		"-s", "31,lpc",
		"-l", "com1,stdio",
	}
	if !NoNet && setuid {
		args = append(args, "-s 3,virtio-net")
	}

	// Bind mounts
	singBinds := []string{""}

	// If we surpass 48 bind mounts ... error. We can't do anything at this point.
	if len(BindPaths) > 48 {
		sylog.Fatalf("Surpassed max amount of binds we can pass to virtual machine")
	}

	// Set slot to 26. slot has a max value of 31, so this will give us a max of 48 bind mounts from the Mac host.
	slot := 26
	function := 0

	for _, bindpath := range BindPaths {
		splitted := strings.Split(bindpath, ":")
		src := splitted[0]
		dst := ""
		if len(splitted) > 1 {
			dst = splitted[1]
		} else {
			dst = src
		}

		sylog.Debugf("Bind path: " + src + " -> " + dst)
		// 6 char is the limit for a usable mount tag...
		mntTag := genString(6)

		// TODO: Figure out if src is a directory or not
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

	// NOTE: The 0:4:x PCI slot is to be used for static mounts. (BUS:SLOT:FUNCTION)
	// Force $HOME to be mounted
	// TODO: engineConfig.GetHomeSource() / GetHomeDest() -- should probably be used
	homeSrc := usr.HomeDir
	pciArgs := fmt.Sprintf("4:0,virtio-9p,home=%s", homeSrc)
	homeBind := fmt.Sprintf("home:%s", homeSrc)
	singBinds = append(singBinds, homeBind)
	sylog.Debugf("PCI: %s", pciArgs)
	args = append(args, "-s", pciArgs)

	// Check for Sandbox Image
	sylog.Debugf("Check for sandbox image")
	if f, err := os.Stat(sifImage); err == nil {
		if f.IsDir() {
			sylog.Debugf("Image is sandbox. Setting up share.")
			pciArgs = fmt.Sprintf("4:1,virtio-9p,runimg=%s", sifImage)
			args = append(args, "-s", pciArgs)
			sboxImgBind := fmt.Sprintf("runimg:/runImage")
			singBinds = append(singBinds, sboxImgBind)
		} else {
			// We are not a sandbox
			args = append(args, "-s", hdString)
		}
	}

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

	// If we disable networking, set "none" for our network
	netip := VMIP
	if NoNet || !setuid {
		netip = "none"
	}

	kexecArgs := fmt.Sprintf("kexec,%s,%s,console=ttyS0 quiet root=/dev/ram0 loglevel=0 sing_img_name=%s sing_user=%s sing_cwd=%s singularity_action=%s ipv4=%s singularity_arguments=\"%s\" singularity_binds=\"%v\"", bzImage, initramfs, filepath.Base(sifImage), userInfo, cwdDir, singAction, netip, cliExtra, strings.Join(singBinds, "|"))

	// Add our actual kexec entry
	args = append(args, "-f", kexecArgs)

	return args
}
