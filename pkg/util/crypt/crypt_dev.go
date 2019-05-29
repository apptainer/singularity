// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package crypt

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/util/fs/lock"
	"github.com/sylabs/singularity/pkg/util/loop"
	"golang.org/x/crypto/ssh/terminal"
)

// Device is
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
	err = cmd.Run()
	if err != nil {
		sylog.Debugf("Unable to delete the crypt device %s", err)
		return err
	}
	err = lock.Release(fd)
	if err != nil {
		sylog.Debugf("Unable to release the lock on /dev/mapper")
		return err
	}

	return nil
}

// FormatCryptDevice allocates a loop device, encrypts, and returns the loop device name, and encrypted device name
func (crypt *Device) FormatCryptDevice(path string) (string, string, error) {

	// Read the password from terminal
	fmt.Print("Enter the password to encrypt the filesystem: ")
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		sylog.Fatalf("Error parsing the password: %s", err)
	}
	input := string(password)

	fmt.Print("\nConfirm the password: ")
	password2, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		sylog.Fatalf("Error parsing the password: %s", err)
	}
	input2 := string(password2)
	fmt.Println()

	if input != input2 {
		return "", "", errors.New("Passwords don't match")
	}

	squashF, err := os.Stat(path)
	if err != nil {
		sylog.Debugf("Error acquiring size of %s", path)
		return "", "", err
	}

	squashFsSize := squashF.Size()

	// Create a temporary file to format with crypt header
	cryptF, err := ioutil.TempFile("", "crypt-")
	if err != nil {
		sylog.Debugf("Error creating temporary crypt file")
		return "", "", err
	}
	defer cryptF.Close()

	// Truncate the file taking the squashfs size and crypt header into account
	// Crypt header is around 2MB in size. Slightly over-allocate to be safe
	size := squashFsSize + 4*1024*1024 // 4MB for LUKS header

	err = os.Truncate(cryptF.Name(), size)
	if err != nil {
		sylog.Debugf("Unable to truncate crypt file to size %d", size)
		return "", "", err
	}

	// NOTE: This routine runs with root privileges. It's not necessary
	// to explicitly set cmd's uid or gid here
	cmd := exec.Command("/sbin/cryptsetup", "luksFormat", cryptF.Name())
	stdin, err := cmd.StdinPipe()

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, input)
	}()

	err = cmd.Run()
	if err != nil {
		sylog.Verbosef("Unable to format crypt device")
		return "", "", err
	}

	// Associate the temporary crypt file with a loop device
	loop, err := createLoop(cryptF, 0, uint64(size))

	sp := strings.Split(loop, "/")
	loopdev := sp[len(sp)-1]

	fd, err := lock.Exclusive("/dev/mapper")
	if err != nil {
		sylog.Debugf("Unable to acquire lock on /dev/mapper")
		return "", "", err
	}
	nextCrypt := getNextAvailableCryptDevice()
	cmd = exec.Command("/sbin/cryptsetup", "luksOpen", loopdev, nextCrypt)
	cmd.Dir = "/dev"
	stdin, err = cmd.StdinPipe()

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, input)
	}()

	err = cmd.Run()
	if err != nil {
		sylog.Verbosef("Unable to open crypt device: %s", nextCrypt)
		return "", "", err
	}

	err = lock.Release(fd)
	if err != nil {
		sylog.Debugf("Unable to release lock on /dev/mapper")
		return "", "", err
	}

	copyDeviceContents(path, "/dev/mapper/"+nextCrypt, squashFsSize)

	// Open a new Temp file to stash the loop contents
	loopF, err := ioutil.TempFile("", "loop-")
	if err != nil {
		sylog.Debugf("Error creating temporary crypt file")
		return "", "", err
	}

	copyDeviceContents("/dev/"+loopdev, loopF.Name(), size)

	return loopF.Name(), nextCrypt, err
}

// copyDeviceContents copies the contents of source to destination.
// source and dest can either be a file or a block device
func copyDeviceContents(source string, dest string, size int64) error {

	squashFd, err := syscall.Open(source, syscall.O_RDWR, 0777)
	if err != nil {
		sylog.Debugf("Unable to open the file %s", source)
		return err
	}
	defer syscall.Close(squashFd)

	cryptFd, err := syscall.Open(dest, syscall.O_RDWR, 0777)

	if err != nil {
		sylog.Debugf("Unable to open the file: %s", dest)
		return err
	}

	defer syscall.Close(cryptFd)

	var writtenSoFar int64

	buffer := make([]byte, 1024)
	for writtenSoFar < size {
		numRead, err := syscall.Read(squashFd, buffer)
		if err != nil {
			sylog.Debugf("Unable to read the the file %s", source)
			return err
		}
		numWritten, err := syscall.Write(cryptFd, buffer)
		if err != nil {
			sylog.Debugf("Unable to write to destination %s", dest)
			return err
		}

		if numRead != numWritten {
			sylog.Debugf("numRead != numWritten")
		}
		writtenSoFar += int64(numWritten)
	}

	return nil
}

func getRandomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		sylog.Debugf("Something went wrong while generating random string")
		return ""
	}
	return fmt.Sprintf("%x", b)
}

func getNextAvailableCryptDevice() string {
	return getRandomString(15)
}

// GetCryptDevice returns the next available device in /dev/mapper for encryption/decryption
func (crypt *Device) GetCryptDevice(loopDev string) (string, error) {

	fmt.Print("Enter the password to decrypt the File System: ")
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		sylog.Fatalf("Error parsing input: %s", err)
	}
	fmt.Println()

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

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, string(password))
	}()

	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "No key available") == true {
			sylog.Debugf("Invalid password")
		}
		if strings.Contains(string(out), "Device already exists") == true {
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
