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
	"github.com/sylabs/singularity/pkg/util/fs/lock"
	"github.com/sylabs/singularity/pkg/util/loop"
	"golang.org/x/crypto/ssh/terminal"
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
		return "", fmt.Errorf("failed to attach image %s: %s", file.Name(), err)
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
		sylog.Debugf("Unable to delete the crypt device %s", err)
		return err
	}

	return nil
}

// ReadKeyFromStdin reads key from terminal and returns it
func (crypt *Device) ReadKeyFromStdin(confirm bool) (string, error) {

	fmt.Print("Enter the Key: ")
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		sylog.Fatalf("Error parsing the key: %s", err)
	}

	input := string(password)
	fmt.Println()
	if confirm {
		fmt.Print("Confirm the Key: ")
		password2, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			sylog.Fatalf("Error parsing the key: %s", err)
		}
		input2 := string(password2)
		fmt.Println()
		if input != input2 {
			return "", errors.New("Keys don't match")
		}
	}

	return input, nil
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
		sylog.Debugf("Error creating temporary crypt file")
		return "", "", err
	}
	defer cryptF.Close()

	// Truncate the file taking the squashfs size and crypt header into account
	// Crypt header is around 2MB in size. Slightly over-allocate to be safe
	devSize := fSize + 4*1024*1024 // 4MB for LUKS header

	err = os.Truncate(cryptF.Name(), devSize)
	if err != nil {
		sylog.Debugf("Unable to truncate crypt file to size %d", devSize)
		return "", "", err
	}

	// NOTE: This routine runs with root privileges. It's not necessary
	// to explicitly set cmd's uid or gid here
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
		return "", "", fmt.Errorf("unable to format crypt device: %s", cryptF.Name())
	}

	// Associate the temporary crypt file with a loop device
	loop, err := createLoop(cryptF, 0, uint64(devSize))
	if err != nil {
		return "", "", err
	}

	loopdev := filepath.Base(loop)

	fd, err := lock.Exclusive("/dev/mapper")
	if err != nil {
		return "", "", fmt.Errorf("unable to acquire lock on /dev/mapper")
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
		sylog.Verbosef("Unable to open crypt device: %s", nextCrypt)
		return "", "", err
	}

	copyDeviceContents(path, "/dev/mapper/"+nextCrypt, fSize)

	// Open a new Temp file to stash the loop contents
	loopF, err := ioutil.TempFile("", "loop-")
	if err != nil {
		sylog.Debugf("Error creating temporary crypt file")
		return "", "", err
	}

	copyDeviceContents("/dev/"+loopdev, loopF.Name(), devSize)

	return loopF.Name(), nextCrypt, err
}

// copyDeviceContents copies the contents of source to destination.
// source and dest can either be a file or a block device
func copyDeviceContents(source, dest string, size int64) error {
	sourceFd, err := syscall.Open(source, syscall.O_RDWR, 0777)
	if err != nil {
		return fmt.Errorf("Unable to open the file %s", source)
	}
	defer syscall.Close(sourceFd)

	destFd, err := syscall.Open(dest, syscall.O_RDWR, 0777)
	if err != nil {
		return fmt.Errorf("Unable to open the file: %s", dest)
	}
	defer syscall.Close(destFd)

	var writtenSoFar int64

	buffer := make([]byte, 1024)
	for writtenSoFar < size {
		numRead, err := syscall.Read(sourceFd, buffer)
		if err != nil {
			return fmt.Errorf("Unable to read the the file %s", source)
		}
		numWritten, err := syscall.Write(destFd, buffer)
		if err != nil {
			return fmt.Errorf("Unable to write to destination %s", dest)
		}

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
		sylog.Debugf("Unable to acquire lock on /dev/mapper while decrypting")
		return "", err
	}
	defer lock.Release(fd)

	maxRetries := 3 // Arbitrary number of retries.

retry:
	numRetries := 0
	nextCrypt := getNextAvailableCryptDevice()
	if nextCrypt == "" {
		return "", errors.New("Crypt Device not available")
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
