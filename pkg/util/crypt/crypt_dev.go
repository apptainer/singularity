// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package crypt

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	uuid "github.com/satori/go.uuid"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/sypgp"
	"github.com/sylabs/singularity/pkg/util/fs/lock"
	"github.com/sylabs/singularity/pkg/util/loop"
)

// Device describes a crypt device
type Device struct{}

// createLoop attaches the file to the next available loop device and
// sets the sizelimit on it
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
		return "", fmt.Errorf("failed to attach image: %s: %s", file.Name(), err)
	}
	return fmt.Sprintf("/dev/loop%d", idx), nil
}

// CloseCryptDevice closes the crypt device
func (crypt *Device) CloseCryptDevice(path string) error {

	cmd := exec.Command("/sbin/cryptsetup", "luksClose", path)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: 0, Gid: 0}
	fd, err := lock.Exclusive("/dev/mapper")
	if err != nil {
		return err
	}
	defer lock.Release(fd)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("unable to delete the crypt device: %s", err)
	}

	return nil
}

// ReadKeyFromStdin reads key from terminal and returns it
// TODO (schebro): Fix #3816, #3851
// Currently keys are being read interactively from the terminal.
// Keys should be non-interactive, preferably in a keyfile that can
// be passed to cryptsetup utility
func (crypt *Device) ReadKeyFromStdin(confirm bool) (string, error) {
	pass1, err := sypgp.AskQuestionNoEcho("Enter a passphrase: ")
	if err != nil {
		return "", fmt.Errorf("unable getting input: %s", err)
	}

	if confirm {
		pass2, err := sypgp.AskQuestionNoEcho("Confirm the passphrase: ")
		if err != nil {
			return "", fmt.Errorf("unable parsing input: %s", err)
		}
		if pass1 != pass2 {
			return "", fmt.Errorf("passphrases don't match")
		}
	}

	return pass1, nil
}

// FormatCryptDevice allocates a loop device, encrypts, and returns the loop device name, and encrypted device name
func (crypt *Device) FormatCryptDevice(path, key string) (string, string, error) {

	f, err := os.Stat(path)
	if err != nil {
		return "", "", fmt.Errorf("failed getting size of %s", path)
	}

	fSize := f.Size()

	// Create a temporary file to format with crypt header
	cryptF, err := ioutil.TempFile("", "crypt-")
	if err != nil {
		return "", "", fmt.Errorf("error creating temporary crypt file: ", err)
	}
	defer cryptF.Close()

	// Truncate the file taking the squashfs size and crypt header into account
	// Crypt header is around 2MB in size. Slightly over-allocate to be safe
	devSize := fSize + 4*1024*1024 // 4MB for LUKS header

	err = os.Truncate(cryptF.Name(), devSize)
	if err != nil {
		return "", "", fmt.Errorf("unable to truncate crypt file to size: %d: %s", devSize, err)
	}

	// NOTE: This routine runs with root privileges. It's not necessary
	// to explicitly set cmd's uid or gid here
	// TODO (schebro): Fix #3818, #3821
	// Currently we are relying on host's cryptsetup utility to encrypt and decrypt
	// the SIF. The possiblity to saving a version of cryptsetup inside the container should be
	// investigated. To do that, at least one additional partition is required, which is
	// not encrypted.

	// TODO (schebro): Fix #3819
	// If we choose not to save a version of cryptsetup in container, host's cryptsetup utility's
	// paty should be saved in a configuration file at build time (similar to mksquashfs) for
	// security reasons

	cmd := exec.Command("/sbin/cryptsetup", "luksFormat", cryptF.Name())
	stdin, err := cmd.StdinPipe()

	if err != nil {
		return "", "", err
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, key)
	}()

	err = cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf("unable to format crypt device: %s: %s", cryptF.Name(), err)
	}

	// Associate the temporary crypt file with a loop device
	loop, err := createLoop(cryptF, 0, uint64(devSize))
	if err != nil {
		return "", "", err
	}

	loopdev := filepath.Base(loop)

	fd, err := lock.Exclusive("/dev/mapper")
	if err != nil {
		return "", "", fmt.Errorf("unable to acquire lock on /dev/mapper: %s", err)
	}
	defer lock.Release(fd)

	nextCrypt := getNextAvailableCryptDevice()
	cmd = exec.Command("/sbin/cryptsetup", "luksOpen", loopdev, nextCrypt)
	cmd.Dir = "/dev"
	stdin, err = cmd.StdinPipe()
	if err != nil {
		return "", "", err
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, key)
	}()

	err = cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf("unable to open crypt device: %s", nextCrypt, err)
	}

	copyDeviceContents(path, "/dev/mapper/"+nextCrypt, fSize)

	// Open a new Temp file to stash the loop contents
	loopF, err := ioutil.TempFile("", "loop-")
	if err != nil {
		return "", "", fmt.Errorf("error creating temporary crypt file: %s", err)
	}

	copyDeviceContents("/dev/"+loopdev, loopF.Name(), devSize)

	return loopF.Name(), nextCrypt, err
}

// copyDeviceContents copies the contents of source to destination.
// source and dest can either be a file or a block device
func copyDeviceContents(source, dest string, size int64) error {
	sourceFd, err := syscall.Open(source, syscall.O_RDWR, 0777)
	if err != nil {
		return fmt.Errorf("unable to open the file: %s: %s", source, err)
	}
	defer syscall.Close(sourceFd)

	destFd, err := syscall.Open(dest, syscall.O_RDWR, 0777)
	if err != nil {
		return fmt.Errorf("unable to open the file: %s: %s", dest, err)
	}
	defer syscall.Close(destFd)

	var writtenSoFar int64

	buffer := make([]byte, 1024)
	for writtenSoFar < size {
		numRead, err := syscall.Read(sourceFd, buffer)
		if err != nil {
			return fmt.Errorf("unable to read the the file: %s: %s", source, err)
		}
		numWritten, err := syscall.Write(destFd, buffer)
		if err != nil {
			return fmt.Errorf("unable to write to destination: %s: %s", dest, err)
		}

		// TODO: is this really necessary???
		if numRead != numWritten {
			sylog.Debugf("numRead != numWritten")
		}
		writtenSoFar += int64(numWritten)
	}

	return nil
}

func getNextAvailableCryptDevice() string {
	return (uuid.NewV4()).String()
}

// GetCryptDevice returns the next available device in /dev/mapper for encryption/decryption
func (crypt *Device) GetCryptDevice(key, loopDev string) (string, error) {
	fd, err := lock.Exclusive("/dev/mapper")
	if err != nil {
		return "", fmt.Errorf("unable to acquire lock on /dev/mapper while decrypting: %s", err)
	}
	defer lock.Release(fd)

	maxRetries := 3 // Arbitrary number of retries.

retry:
	numRetries := 0
	nextCrypt := getNextAvailableCryptDevice()
	if nextCrypt == "" {
		return "", errors.New("crypt device not available")
	}

	cmd := exec.Command("/sbin/cryptsetup", "luksOpen", loopDev, nextCrypt)
	cmd.Dir = "/dev"
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: 0, Gid: 0}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, key)
	}()

	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "No key available") {
			sylog.Debugf("Invalid password")
		}
		if strings.Contains(string(out), "Device already exists") {
			numRetries++
			if numRetries < maxRetries {
				goto retry
			}
		}
		return "", err
	}
	sylog.Debugf("Decrypted the FS successfully")

	return nextCrypt, nil
}
