// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package crypt

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"

	uuid "github.com/satori/go.uuid"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/bin"
	"github.com/sylabs/singularity/pkg/util/fs/lock"
	"github.com/sylabs/singularity/pkg/util/loop"
)

// Device describes a crypt device
type Device struct{}

// Pre-defined error(s)
var (
	// ErrUnsupportedCryptsetupVersion is the error raised when the available version
	// of cryptsetup is not compatible with the Singularity encryption mechanism.
	ErrUnsupportedCryptsetupVersion = errors.New("available cryptsetup is not supported")
)

// createLoop attaches the specified file to the next available loop
// device and sets the sizelimit on it
func createLoop(path string, offset, size uint64) (string, error) {
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
	if err := loopDev.AttachFromPath(path, os.O_RDWR, &idx); err != nil {
		return "", fmt.Errorf("failed to attach image %s: %s", path, err)
	}
	return fmt.Sprintf("/dev/loop%d", idx), nil
}

// CloseCryptDevice closes the crypt device
func (crypt *Device) CloseCryptDevice(path string) error {
	cryptsetup, err := bin.Cryptsetup()
	if err != nil {
		return err
	}

	fd, err := lock.Exclusive("/dev/mapper")
	if err != nil {
		return err
	}
	defer lock.Release(fd)

	cmd := exec.Command(cryptsetup, "close", path)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: 0, Gid: 0},
	}
	sylog.Debugf("Running %s %s", cmd.Path, strings.Join(cmd.Args, " "))
	err = cmd.Run()
	if err != nil {
		sylog.Debugf("Unable to delete the crypt device %s", err)
		return err
	}

	return nil
}

func checkCryptsetupVersion(cryptsetup string) error {
	if cryptsetup == "" {
		return fmt.Errorf("binary path not defined")
	}

	cmd := exec.Command(cryptsetup, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run cryptsetup --version: %s", err)
	}

	if !strings.Contains(string(out), "cryptsetup 2.") {
		return ErrUnsupportedCryptsetupVersion
	}

	// We successfully ran cryptsetup --version and we know that the
	// version is compatible with our needs.
	return nil
}

// EncryptFilesystem takes the path to a file containing a non-encrypted
// filesystem, encrypts it using the provided key, and returns a path to
// a file that can be later used as an encrypted volume with cryptsetup.
// NOTE: it is the callers responsibility to remove the returned file that
// contains the crypt header.
func (crypt *Device) EncryptFilesystem(path string, key []byte) (string, error) {
	f, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("failed getting size of %s", path)
	}

	fSize := f.Size()

	// Create a temporary file to format with crypt header
	cryptF, err := ioutil.TempFile("", "crypt-")
	if err != nil {
		sylog.Debugf("Error creating temporary crypt file")
		return "", err
	}
	defer cryptF.Close()

	// Truncate the file taking the squashfs size and crypt header
	// into account. With the options specified below the LUKS header
	// is less than 16MB in size. Slightly over-allocate
	// to compensate for the encryption overhead itself.
	//
	// TODO(mem): the encryption overhead might depend on the size
	// of the data we are encrypting. For very large images, we
	// might not be overallocating enough. Figure out what's the
	// actual percentage we need to overallocate.
	devSize := fSize + 16*1024*1024

	sylog.Debugf("Total device size for encrypted image: %d", devSize)
	err = os.Truncate(cryptF.Name(), devSize)
	if err != nil {
		sylog.Debugf("Unable to truncate crypt file to size %d", devSize)
		return "", err
	}

	cryptF.Close()

	// Associate the temporary crypt file with a loop device
	loop, err := createLoop(cryptF.Name(), 0, uint64(devSize))
	if err != nil {
		return "", err
	}

	// NOTE: This routine runs with root privileges. It's not necessary
	// to explicitly set cmd's uid or gid here
	// TODO (schebro): Fix #3818, #3821
	// Currently we are relying on host's cryptsetup utility to encrypt and decrypt
	// the SIF. The possiblity to saving a version of cryptsetup inside the container should be
	// investigated. To do that, at least one additional partition is required, which is
	// not encrypted.

	cryptsetup, err := bin.Cryptsetup()
	if err != nil {
		return "", err
	}

	cmd := exec.Command(cryptsetup, "luksFormat", "--batch-mode", "--type", "luks2", "--key-file", "-", loop)
	stdin, err := cmd.StdinPipe()

	if err != nil {
		return "", err
	}

	go func() {
		stdin.Write(key)
		stdin.Close()
	}()

	sylog.Debugf("Running %s %s", cmd.Path, strings.Join(cmd.Args, " "))
	out, err := cmd.CombinedOutput()
	if err != nil {
		err = checkCryptsetupVersion(cryptsetup)
		if err == ErrUnsupportedCryptsetupVersion {
			// Special case of unsupported version of cryptsetup. We return the raw error
			// so it can propagate up and a user-friendly message be displayed. This error
			// should trigger an error at the CLI level.
			return "", err
		}
		return "", fmt.Errorf("unable to format crypt device: %s: %s", cryptF.Name(), string(out))
	}

	nextCrypt, err := crypt.Open(key, loop)
	if err != nil {
		sylog.Verbosef("Unable to open encrypted device %s: %s", loop, err)
		return "", err
	}

	copyDeviceContents(path, "/dev/mapper/"+nextCrypt, fSize)

	cmd = exec.Command(cryptsetup, "close", nextCrypt)
	sylog.Debugf("Running %s %s", cmd.Path, strings.Join(cmd.Args, " "))
	err = cmd.Run()
	if err != nil {
		return "", err
	}

	return cryptF.Name(), err
}

// copyDeviceContents copies the contents of source to destination.
// source and dest can either be a file or a block device
func copyDeviceContents(source, dest string, size int64) error {
	sylog.Debugf("Copying %s to %s, size %d", source, dest, size)

	sourceFd, err := syscall.Open(source, syscall.O_RDONLY, 0000)
	if err != nil {
		return fmt.Errorf("unable to open the file %s", source)
	}
	defer syscall.Close(sourceFd)

	destFd, err := syscall.Open(dest, syscall.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("unable to open the file: %s", dest)
	}
	defer syscall.Close(destFd)

	var writtenSoFar int64

	buffer := make([]byte, 10240)
	for writtenSoFar < size {
		buffer = buffer[:cap(buffer)]
		numRead, err := syscall.Read(sourceFd, buffer)
		if err != nil {
			return fmt.Errorf("unable to read the the file %s", source)
		}
		buffer = buffer[:numRead]
		for n := 0; n < numRead; {
			numWritten, err := syscall.Write(destFd, buffer[n:])
			if err != nil {
				return fmt.Errorf("unable to write to destination %s", dest)
			}
			n += numWritten
			writtenSoFar += int64(numWritten)
		}
	}

	return nil
}

func getNextAvailableCryptDevice() string {
	return (uuid.NewV4()).String()
}

// Open opens the encrypted filesystem specified by path (usually a loop
// device, but any encrypted block device will do) using the given key
// and returns the name assigned to it that can be later used to close
// the device.
func (crypt *Device) Open(key []byte, path string) (string, error) {
	fd, err := lock.Exclusive("/dev/mapper")
	if err != nil {
		return "", fmt.Errorf("unable to acquire lock on /dev/mapper")
	}
	defer lock.Release(fd)

	maxRetries := 3 // Arbitrary number of retries.

	cryptsetup, err := bin.Cryptsetup()
	if err != nil {
		return "", err
	}

	for i := 0; i < maxRetries; i++ {
		nextCrypt := getNextAvailableCryptDevice()
		if nextCrypt == "" {
			return "", errors.New("Crypt device not available")
		}

		cmd := exec.Command(cryptsetup, "open", "--batch-mode", "--type", "luks2", "--key-file", "-", path, nextCrypt)
		cmd.SysProcAttr = &syscall.SysProcAttr{}
		cmd.SysProcAttr.Credential = &syscall.Credential{Uid: 0, Gid: 0}
		sylog.Debugf("Running %s %s", cmd.Path, strings.Join(cmd.Args, " "))
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return "", err
		}

		go func() {
			stdin.Write(key)
			stdin.Close()
		}()

		out, err := cmd.CombinedOutput()
		if err != nil {
			if strings.Contains(string(out), "No key available") {
				sylog.Debugf("Invalid password")
			}
			if strings.Contains(string(out), "Device already exists") {
				continue
			}
			err = checkCryptsetupVersion(cryptsetup)
			if err == ErrUnsupportedCryptsetupVersion {
				// Special case of unsupported version of cryptsetup. We return the raw error
				// so it can propagate up and a user-friendly message be displayed. This error
				// should trigger an error at the CLI level.
				return "", err
			}

			return "", fmt.Errorf("cryptsetup open failed: %s: %v", string(out), err)
		}
		sylog.Debugf("Successfully opened encrypted device %s", path)
		return nextCrypt, nil
	}

	return "", errors.New("Unable to open crypt device")
}
