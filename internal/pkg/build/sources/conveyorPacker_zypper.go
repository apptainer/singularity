// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sources

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
)

const (
	zypperConf = "/etc/zypp/zypp.conf"
)

// ZypperConveyorPacker only needs to hold the bundle for the container
type ZypperConveyorPacker struct {
	b *types.Bundle
}

// Get downloads container information from the specified source
func (cp *ZypperConveyorPacker) Get(b *types.Bundle) (err error) {
	cp.b = b

	// check for zypper on system
	zypperPath, err := exec.LookPath("zypper")
	if err != nil {
		return fmt.Errorf("zypper is not in PATH: %v", err)
	}

	// check for rpm on system
	err = rpmPathCheck()
	if err != nil {
		return
	}

	// get mirrorURL, OSVerison, and Includes components to definition
	mirrorurl, ok := cp.b.Recipe.Header["mirrorurl"]
	if !ok {
		return fmt.Errorf("Invalid zypper header, no MirrorURL specified")
	}

	// look for an OS version if the mirror specifies it
	osversion := ""
	regex := regexp.MustCompile(`(?i)%{OSVERSION}`)
	if regex.MatchString(mirrorurl) {
		osversion, ok = cp.b.Recipe.Header["osversion"]
		if !ok {
			return fmt.Errorf("Invalid zypper header, OSVersion referenced in mirror but no OSVersion specified")
		}
		mirrorurl = regex.ReplaceAllString(mirrorurl, osversion)
	}

	include, _ := cp.b.Recipe.Header["include"]

	// check for include environment variable and add it to requires string
	include += ` ` + os.Getenv("INCLUDE")

	// trim leading and trailing whitespace
	include = strings.TrimSpace(include)

	// add aaa_base to start of include list by default
	include = `aaa_base ` + include

	// Create the main portion of zypper config
	err = cp.genZypperConfig()
	if err != nil {
		return fmt.Errorf("While generating Zypper config: %v", err)
	}

	err = cp.copyPseudoDevices()
	if err != nil {
		return fmt.Errorf("While copying pseudo devices: %v", err)
	}

	// Add mirrorURL as repo
	cmd := exec.Command(zypperPath, `--root`, cp.b.Rootfs(), `ar`, mirrorurl, `repo-oss`)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("While adding zypper mirror: %v", err)
	}

	// Refreshing gpg keys
	cmd = exec.Command(zypperPath, `--root`, cp.b.Rootfs(), `--gpg-auto-import-keys`, `refresh`)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("While refreshing gpg keys: %v", err)
	}

	args := []string{`--non-interactive`, `-c`, filepath.Join(cp.b.Rootfs(), zypperConf), `--root`, cp.b.Rootfs(), `--releasever=` + osversion, `-n`, `install`, `--auto-agree-with-licenses`, `--download-in-advance`}
	args = append(args, strings.Fields(include)...)

	// Zypper install command
	cmd = exec.Command(zypperPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	sylog.Debugf("\n\tZypper Path: %s\n\tDetected Arch: %s\n\tOSVersion: %s\n\tMirrorURL: %s\n\tIncludes: %s\n", zypperPath, runtime.GOARCH, osversion, mirrorurl, include)

	// run zypper
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("While bootstrapping from zypper: %v", err)
	}

	return nil
}

// Pack puts relevant objects in a Bundle!
func (cp *ZypperConveyorPacker) Pack() (b *types.Bundle, err error) {
	err = cp.insertBaseEnv()
	if err != nil {
		return nil, fmt.Errorf("While inserting base environment: %v", err)
	}

	err = cp.insertRunScript()
	if err != nil {
		return nil, fmt.Errorf("While inserting runscript: %v", err)
	}

	return cp.b, nil
}

func (cp *ZypperConveyorPacker) insertBaseEnv() (err error) {
	if err = makeBaseEnv(cp.b.Rootfs()); err != nil {
		return
	}
	return nil
}

func (cp *ZypperConveyorPacker) insertRunScript() (err error) {
	f, err := os.Create(cp.b.Rootfs() + "/.singularity.d/runscript")
	if err != nil {
		return
	}

	defer f.Close()

	_, err = f.WriteString("#!/bin/sh\n")
	if err != nil {
		return
	}

	if err != nil {
		return
	}

	f.Sync()

	err = os.Chmod(cp.b.Rootfs()+"/.singularity.d/runscript", 0755)
	if err != nil {
		return
	}

	return nil
}

func (cp *ZypperConveyorPacker) genZypperConfig() (err error) {
	err = os.MkdirAll(filepath.Join(cp.b.Rootfs(), "/etc/zypp"), 0775)
	if err != nil {
		return fmt.Errorf("While creating %v: %v", filepath.Join(cp.b.Rootfs(), "/etc/zypp"), err)
	}

	err = ioutil.WriteFile(filepath.Join(cp.b.Rootfs(), zypperConf), []byte("[main]\ncachedir=/val/cache/zypp-bootstrap\n\n"), 0664)
	if err != nil {
		return
	}

	return nil
}

func (cp *ZypperConveyorPacker) copyPseudoDevices() (err error) {
	err = os.Mkdir(filepath.Join(cp.b.Rootfs(), "/dev"), 0775)
	if err != nil {
		return fmt.Errorf("While creating %v: %v", filepath.Join(cp.b.Rootfs(), "/dev"), err)
	}

	devs := []string{"/dev/null", "/dev/zero", "/dev/random", "/dev/urandom"}

	for _, dev := range devs {
		cmd := exec.Command("cp", "-a", dev, filepath.Join(cp.b.Rootfs(), "/dev"))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err = cmd.Run(); err != nil {
			f, err := os.Create(cp.b.Rootfs() + "/.singularity.d/runscript")
			if err != nil {
				return fmt.Errorf("While creating %v: %v", filepath.Join(cp.b.Rootfs(), dev), err)
			}

			defer f.Close()
		}
	}

	return nil
}

func rpmPathCheck() (err error) {
	output := &bytes.Buffer{}
	cmd := exec.Command("rpm", "--showrc")
	cmd.Stdout = output

	if err = cmd.Run(); err != nil {
		return
	}

	rpmDBPath := ""
	scanner := bufio.NewScanner(output)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		// search for dbpath from showrc output
		if strings.Contains(scanner.Text(), "_dbpath\t") {
			// second field in the string is the path
			rpmDBPath = strings.Fields(scanner.Text())[2]
		}
	}

	if rpmDBPath != `%{_var}/lib/rpm` {
		return fmt.Errorf("RPM database is using a non-standard path: %s\n"+
			"There is a way to work around this problem:\n"+
			"Create a file at path %s/.rpmmacros.\n"+
			"Place the following lines into the '.rpmmacros' file:\n"+
			"%s\n"+
			"%s\n"+
			"After creating the file, re-run the bootstrap.\n"+
			"More info: https://github.com/sylabs/singularity/issues/241\n",
			rpmDBPath, os.Getenv("HOME"), `%_var /var`, `%_dbpath %{_var}/lib/rpm`)
	}

	return nil
}
