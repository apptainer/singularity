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
	yumConf = "/etc/bootstrap-yum.conf"
)

// YumConveyor holds stuff that needs to be packed into the bundle
type YumConveyor struct {
	b         *types.Bundle
	rpmPath   string
	mirrorurl string
	updateurl string
	osversion string
	include   string
	gpg       string
	httpProxy string
}

// YumConveyorPacker only needs to hold the conveyor to have the needed data to pack
type YumConveyorPacker struct {
	YumConveyor
}

// Get downloads container information from the specified source
func (c *YumConveyor) Get(b *types.Bundle) (err error) {
	c.b = b

	// check for dnf or yum on system
	var installCommandPath string
	if installCommandPath, err = exec.LookPath("dnf"); err == nil {
		sylog.Debugf("Found dnf at: %v", installCommandPath)
	} else if installCommandPath, err = exec.LookPath("yum"); err == nil {
		sylog.Debugf("Found yum at: %v", installCommandPath)
	} else {
		return fmt.Errorf("Neither yum nor dnf in PATH")
	}

	// check for rpm on system
	err = c.getRPMPath()
	if err != nil {
		return fmt.Errorf("While checking rpm path: %v", err)
	}

	err = c.getBootstrapOptions()
	if err != nil {
		return fmt.Errorf("While getting bootstrap options: %v", err)
	}

	err = c.genYumConfig()
	if err != nil {
		return fmt.Errorf("While generating Yum config: %v", err)
	}

	err = c.copyPseudoDevices()
	if err != nil {
		return fmt.Errorf("While copying pseudo devices: %v", err)
	}

	args := []string{`--noplugins`, `-c`, filepath.Join(c.b.Rootfs(), yumConf), `--installroot`, c.b.Rootfs(), `--releasever=` + c.osversion, `-y`, `install`}
	args = append(args, strings.Fields(c.include)...)

	// Do the install
	sylog.Debugf("\n\tInstall Command Path: %s\n\tDetected Arch: %s\n\tOSVersion: %s\n\tMirrorURL: %s\n\tUpdateURL: %s\n\tIncludes: %s\n", installCommandPath, runtime.GOARCH, c.osversion, c.mirrorurl, c.updateurl, c.include)
	cmd := exec.Command(installCommandPath, args...)
	// cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("While bootstrapping: %v", err)
	}

	// clean up bootstrap packages
	os.RemoveAll(filepath.Join(c.b.Rootfs(), "/var/cache/yum-bootstrap"))

	return nil
}

// Pack puts relevant objects in a Bundle!
func (cp *YumConveyorPacker) Pack() (b *types.Bundle, err error) {
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

func (c *YumConveyor) getRPMPath() (err error) {
	c.rpmPath, err = exec.LookPath("rpm")
	if err != nil {
		return fmt.Errorf("RPM is not in PATH: %v", err)
	}

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

	if rpmDBPath == "" {
		return fmt.Errorf("Could find dbpath")
	} else if rpmDBPath != `%{_var}/lib/rpm` {
		return fmt.Errorf("RPM database is using a weird path: %s\n"+
			"You are probably running this bootstrap on Debian or Ubuntu.\n"+
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

func (c *YumConveyor) getBootstrapOptions() (err error) {
	var ok bool

	// look for http_proxy and gpg environment vars
	c.gpg = os.Getenv("GPG")
	c.httpProxy = os.Getenv("http_proxy")

	// get mirrorURL, updateURL, OSVerison, and Includes components to definition
	c.mirrorurl, ok = c.b.Recipe.Header["mirrorurl"]
	if !ok {
		return fmt.Errorf("Invalid yum header, no MirrorURL specified")
	}

	c.updateurl, _ = c.b.Recipe.Header["updateurl"]

	// look for an OS version if a mirror specifies it
	regex := regexp.MustCompile(`(?i)%{OSVERSION}`)
	if regex.MatchString(c.mirrorurl) || regex.MatchString(c.updateurl) {
		c.osversion, ok = c.b.Recipe.Header["osversion"]
		if !ok {
			return fmt.Errorf("Invalid yum header, OSVersion referenced in mirror but no OSVersion specified")
		}
		c.mirrorurl = regex.ReplaceAllString(c.mirrorurl, c.osversion)
		c.updateurl = regex.ReplaceAllString(c.updateurl, c.osversion)
	}

	include, _ := c.b.Recipe.Header["include"]

	// check for include environment variable and add it to requires string
	include += ` ` + os.Getenv("INCLUDE")

	// trim leading and trailing whitespace
	include = strings.TrimSpace(include)

	// add aa_base to start of include list by default
	include = `/etc/redhat-release coreutils ` + include

	c.include = include

	return nil
}

func (c *YumConveyor) genYumConfig() (err error) {
	fileContent := "[main]\n"
	// http proxy
	if c.httpProxy != "" {
		fileContent += "proxy=" + c.httpProxy + "\n"
	}
	fileContent += "cachedir=/var/cache/yum-bootstrap\n"
	fileContent += "keepcache=0\n"
	fileContent += "debuglevel=2\n"
	fileContent += "logfile=/var/log/yum.log\n"
	fileContent += "syslog_device=/dev/null\n"
	fileContent += "exactarch=1\n"
	fileContent += "obsoletes=1\n"
	// gpg
	if c.gpg != "" {
		fileContent += "gpgcheck=1\n"
	} else {
		fileContent += "gpgcheck=0\n"
	}
	fileContent += "plugins=1\n"
	fileContent += "reposdir=0\n"
	fileContent += "deltarpm=0\n"
	fileContent += "\n"
	fileContent += "[base]\n"
	fileContent += "name=Linux $releasever - $basearch\n"
	// mirror
	if c.mirrorurl != "" {
		fileContent += "baseurl=" + c.mirrorurl + "\n"
	}
	fileContent += "enabled=1\n"
	// gpg
	if c.gpg != "" {
		fileContent += "gpgcheck=1\n"
	} else {
		fileContent += "gpgcheck=0\n"
	}

	// add update section if updateurl is specified
	if c.updateurl != "" {
		fileContent += "[updates]\n"
		fileContent += "name=Linux $releasever - $basearch updates\n"
		fileContent += "baseurl=" + c.updateurl + "\n"
		fileContent += "enabled=1\n"
		// gpg
		if c.gpg != "" {
			fileContent += "gpgcheck=1\n"
		} else {
			fileContent += "gpgcheck=0\n"
		}
		fileContent += "\n"
	}

	err = os.Mkdir(filepath.Join(c.b.Rootfs(), "/etc"), 0775)
	if err != nil {
		return fmt.Errorf("While creating %v: %v", filepath.Join(c.b.Rootfs(), "/etc"), err)
	}

	err = ioutil.WriteFile(filepath.Join(c.b.Rootfs(), yumConf), []byte(fileContent), 0664)
	if err != nil {
		return fmt.Errorf("While creating %v: %v", filepath.Join(c.b.Rootfs(), yumConf), err)
	}

	// if gpg key is specified, import it
	if c.gpg != "" {
		err = c.importGPGKey()
		if err != nil {
			return fmt.Errorf("While importing GPG key: %v", err)
		}
	} else {
		sylog.Infof("Skipping GPG Key Import")
	}

	return nil
}

func (c *YumConveyor) importGPGKey() (err error) {
	sylog.Infof("We have a GPG key!  Preparing RPM database.")

	// make sure gpg is being imported over https
	if strings.HasPrefix(c.gpg, "https://") == false {
		return fmt.Errorf("GPG key must be fetched with https")
	}

	// make sure curl is installed so rpm can import gpg key
	if _, err = exec.LookPath("curl"); err != nil {
		return fmt.Errorf("Neither yum nor dnf in PATH")
	}

	cmd := exec.Command(c.rpmPath, "--root", c.b.Rootfs(), "--initdb")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("While initializing new rpm db: %v", err)
	}

	cmd = exec.Command(c.rpmPath, "--root", c.b.Rootfs(), "--import", c.gpg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("While importing GPG key with rpm: %v", err)
	}

	sylog.Infof("GPG key import complete!")

	return nil
}

func (c *YumConveyor) copyPseudoDevices() (err error) {
	err = os.Mkdir(filepath.Join(c.b.Rootfs(), "/dev"), 0775)
	if err != nil {
		return fmt.Errorf("While creating %v: %v", filepath.Join(c.b.Rootfs(), "/dev"), err)
	}

	devs := []string{"/dev/null", "/dev/zero", "/dev/random", "/dev/urandom"}

	for _, dev := range devs {
		cmd := exec.Command("cp", "-a", dev, filepath.Join(c.b.Rootfs(), "/dev"))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err = cmd.Run(); err != nil {
			f, err := os.Create(c.b.Rootfs() + "/.singularity.d/runscript")
			if err != nil {
				return fmt.Errorf("While creating %v: %v", filepath.Join(c.b.Rootfs(), dev), err)
			}

			defer f.Close()
		}
	}

	return nil
}

func (cp *YumConveyorPacker) insertBaseEnv() (err error) {
	if err = makeBaseEnv(cp.b.Rootfs()); err != nil {
		return
	}
	return nil
}

func (cp *YumConveyorPacker) insertRunScript() (err error) {
	ioutil.WriteFile(filepath.Join(cp.b.Rootfs(), "/.singularity.d/runscript"), []byte("#!/bin/sh\n"), 0755)
	if err != nil {
		return
	}

	return nil
}
