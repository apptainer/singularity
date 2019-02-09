// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package loop

// Device describes a loop device
type Device struct {
	MaxLoopDevices int
	Shared         bool
	Info           *Info64
}

// Loop device flags values
const (
	FlagsReadOnly  = 1
	FlagsAutoClear = 4
	FlagsPartScan  = 8
	FlagsDirectIO  = 16
)

// Loop device encryption types
const (
	CryptNone      = 0
	CryptXor       = 1
	CryptDes       = 2
	CryptFish2     = 3
	CryptBlow      = 4
	CryptCast128   = 5
	CryptIdea      = 6
	CryptDummy     = 9
	CryptSkipJack  = 10
	CryptCryptoAPI = 18
	CryptMax       = 20
)

// Loop device IOCTL commands
const (
	CmdSetFd       = 0x4C00
	CmdClrFd       = 0x4C01
	CmdSetStatus   = 0x4C02
	CmdGetStatus   = 0x4C03
	CmdSetStatus64 = 0x4C04
	CmdGetStatus64 = 0x4C05
	CmdChangeFd    = 0x4C06
	CmdSetCapacity = 0x4C07
	CmdSetDirectIO = 0x4C08
)

// Info64 contains information about a loop device.
type Info64 struct {
	Device         uint64
	Inode          uint64
	Rdevice        uint64
	Offset         uint64
	SizeLimit      uint64
	Number         uint32
	EncryptType    uint32
	EncryptKeySize uint32
	Flags          uint32
	FileName       [64]byte
	CryptName      [64]byte
	EncryptKey     [32]byte
	Init           [2]uint64
}
