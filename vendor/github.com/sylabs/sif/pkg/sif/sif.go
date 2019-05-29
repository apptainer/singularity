// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
// Copyright (c) 2017, Yannick Cote <yhcote@gmail.com> All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package sif implements data structures and routines to create
// and access SIF files.
// 	- sif.go contains the data definition the file format.
//	- create.go implements the core functionality for the creation of
//	  of new SIF files.
//	- load.go implements the core functionality for the loading of
//	  existing SIF files.
//	- lookup.go mostly implements search/lookup and printing routines
//	  and access to specific descriptor/data found in SIF container files.
//
// Layout of a SIF file (example):
//
//     .================================================.
//     | GLOBAL HEADER: Sifheader                       |
//     | - launch: "#!/usr/bin/env..."                  |
//     | - magic: "SIF_MAGIC"                           |
//     | - version: "1"                                 |
//     | - arch: "4"                                    |
//     | - uuid: b2659d4e-bd50-4ea5-bd17-eec5e54f918e   |
//     | - ctime: 1504657553                            |
//     | - mtime: 1504657653                            |
//     | - ndescr: 3                                    |
//     | - descroff: 120                                | --.
//     | - descrlen: 432                                |   |
//     | - dataoff: 4096                                |   |
//     | - datalen: 619362                              |   |
//     |------------------------------------------------| <-'
//     | DESCR[0]: Sifdeffile                           |
//     | - Sifcommon                                    |
//     |   - datatype: DATA_DEFFILE                     |
//     |   - id: 1                                      |
//     |   - groupid: 1                                 |
//     |   - link: NONE                                 |
//     |   - fileoff: 4096                              | --.
//     |   - filelen: 222                               |   |
//     |------------------------------------------------| <-----.
//     | DESCR[1]: Sifpartition                         |   |   |
//     | - Sifcommon                                    |   |   |
//     |   - datatype: DATA_PARTITION                   |   |   |
//     |   - id: 2                                      |   |   |
//     |   - groupid: 1                                 |   |   |
//     |   - link: NONE                                 |   |   |
//     |   - fileoff: 4318                              | ----. |
//     |   - filelen: 618496                            |   | | |
//     | - fstype: Squashfs                             |   | | |
//     | - parttype: System                             |   | | |
//     | - content: Linux                               |   | | |
//     |------------------------------------------------|   | | |
//     | DESCR[2]: Sifsignature                         |   | | |
//     | - Sifcommon                                    |   | | |
//     |   - datatype: DATA_SIGNATURE                   |   | | |
//     |   - id: 3                                      |   | | |
//     |   - groupid: NONE                              |   | | |
//     |   - link: 2                                    | ------'
//     |   - fileoff: 622814                            | ------.
//     |   - filelen: 644                               |   | | |
//     | - hashtype: SHA384                             |   | | |
//     | - entity: @                                    |   | | |
//     |------------------------------------------------| <-' | |
//     | Definition file data                           |     | |
//     | .                                              |     | |
//     | .                                              |     | |
//     | .                                              |     | |
//     |------------------------------------------------| <---' |
//     | File system partition image                    |       |
//     | .                                              |       |
//     | .                                              |       |
//     | .                                              |       |
//     |------------------------------------------------| <-----'
//     | Signed verification data                       |
//     | .                                              |
//     | .                                              |
//     | .                                              |
//     `================================================'
//
package sif

import (
	"bytes"
	"io"
	"os"

	uuid "github.com/satori/go.uuid"
)

// SIF header constants and quantities
const (
	HdrLaunch       = "#!/usr/bin/env run-singularity\n"
	HdrMagic        = "SIF_MAGIC" // SIF identification
	HdrVersion      = "01"        // SIF SPEC VERSION
	HdrArchUnknown  = "00"        // Undefined/Unsupported arch
	HdrArch386      = "01"        // 386 (i[3-6]86) arch code
	HdrArchAMD64    = "02"        // AMD64 arch code
	HdrArchARM      = "03"        // ARM arch code
	HdrArchARM64    = "04"        // AARCH64 arch code
	HdrArchPPC64    = "05"        // PowerPC 64 arch code
	HdrArchPPC64le  = "06"        // PowerPC 64 little-endian arch code
	HdrArchMIPS     = "07"        // MIPS arch code
	HdrArchMIPSle   = "08"        // MIPS little-endian arch code
	HdrArchMIPS64   = "09"        // MIPS64 arch code
	HdrArchMIPS64le = "10"        // MIPS64 little-endian arch code
	HdrArchS390x    = "11"        // IBM s390x arch code

	HdrLaunchLen  = 32 // len("#!/usr/bin/env... ")
	HdrMagicLen   = 10 // len("SIF_MAGIC")
	HdrVersionLen = 3  // len("99")
	HdrArchLen    = 3  // len("99")

	DescrNumEntries   = 48                 // the default total number of available descriptors
	DescrGroupMask    = 0xf0000000         // groups start at that offset
	DescrUnusedGroup  = DescrGroupMask     // descriptor without a group
	DescrDefaultGroup = DescrGroupMask | 1 // first groupid number created
	DescrUnusedLink   = 0                  // descriptor without link to other
	DescrEntityLen    = 256                // len("Joe Bloe <jbloe@gmail.com>...")
	DescrNameLen      = 128                // descriptor name (string identifier)
	DescrMaxPrivLen   = 384                // size reserved for descriptor specific data
	DescrStartOffset  = 4096               // where descriptors start after global header
	DataStartOffset   = 32768              // where data object start after descriptors
)

// Datatype represents the different SIF data object types stored in the image
type Datatype int32

// List of supported SIF data types
const (
	DataDeffile     Datatype = iota + 0x4001 // definition file data object
	DataEnvVar                               // environment variables data object
	DataLabels                               // JSON labels data object
	DataPartition                            // file system data object
	DataSignature                            // signing/verification data object
	DataGenericJSON                          // generic JSON meta-data
	DataGeneric                              // generic / raw data
)

// Fstype represents the different SIF file system types found in partition data objects
type Fstype int32

// List of supported file systems
const (
	FsSquash  Fstype = iota + 1 // Squashfs file system, RDONLY
	FsExt3                      // EXT3 file system, RDWR (deprecated)
	FsImmuObj                   // immutable data object archive
	FsRaw                       // raw data
	FsEncrypt                   // Encrypted File System
)

// Parttype represents the different SIF container partition types (system and data)
type Parttype int32

// List of supported partition types
const (
	PartSystem  Parttype = iota + 1 // partition hosts an operating system
	PartPrimSys                     // partition hosts the primary operating system
	PartData                        // partition hosts data only
	PartOverlay                     // partition hosts an overlay
)

// Hashtype represents the different SIF hashing function types used to fingerprint data objects
type Hashtype int32

// List of supported hash functions
const (
	HashSHA256 Hashtype = iota + 1
	HashSHA384
	HashSHA512
	HashBLAKE2S
	HashBLAKE2B
)

// SIF data object deletation strategies
const (
	DelZero    = iota + 1 // zero the data object bytes
	DelCompact            // free the space used by data object
)

// Descriptor represents the SIF descriptor type
type Descriptor struct {
	Datatype Datatype // informs of descriptor type
	Used     bool     // is the descriptor in use
	ID       uint32   // a unique id for this data object
	Groupid  uint32   // object group this data object is related to
	Link     uint32   // special link or relation to an id or group
	Fileoff  int64    // offset from start of image file
	Filelen  int64    // length of data in file
	Storelen int64    // length of data + alignment to store data in file

	Ctime int64                 // image creation time
	Mtime int64                 // last modification time
	UID   int64                 // system user owning the file
	Gid   int64                 // system group owning the file
	Name  [DescrNameLen]byte    // descriptor name (string identifier)
	Extra [DescrMaxPrivLen]byte // big enough for extra data below
}

// Deffile represents the SIF definition-file data object descriptor
type Deffile struct {
}

// Labels represents the SIF JSON-labels data object descriptor
type Labels struct {
}

// Envvar represents the SIF envvar data object descriptor
type Envvar struct {
}

// Partition represents the SIF partition data object descriptor
type Partition struct {
	Fstype   Fstype
	Parttype Parttype
	Arch     [HdrArchLen]byte // arch the image is built for
}

// Signature represents the SIF signature data object descriptor
type Signature struct {
	Hashtype Hashtype
	Entity   [DescrEntityLen]byte
}

// GenericJSON represents the SIF generic JSON meta-data data object descriptor
type GenericJSON struct {
}

// Generic represents the SIF generic data object descriptor
type Generic struct {
}

// Header describes a loaded SIF file
type Header struct {
	Launch [HdrLaunchLen]byte // #! shell execution line

	Magic   [HdrMagicLen]byte   // look for "SIF_MAGIC"
	Version [HdrVersionLen]byte // SIF version
	Arch    [HdrArchLen]byte    // arch the primary partition is built for
	ID      uuid.UUID           // image unique identifier

	Ctime int64 // image creation time
	Mtime int64 // last modification time

	Dfree    int64 // # of unused data object descr.
	Dtotal   int64 // # of total available data object descr.
	Descroff int64 // bytes into file where descs start
	Descrlen int64 // bytes used by all current descriptors
	Dataoff  int64 // bytes into file where data starts
	Datalen  int64 // bytes used by all data objects
}

//
// This section describes SIF creation/loading data structures used when
// building or opening a SIF file. Transient data not found in the final
// SIF file. Those data structures are internal.
//

// ReadWriter describes the operations needed to support reading and
// writing SIF files
type ReadWriter interface {
	Name() string
	Close() error
	Fd() uintptr
	Read(b []byte) (n int, err error)
	Seek(offset int64, whence int) (ret int64, err error)
	Stat() (os.FileInfo, error)
	Sync() error
	Truncate(size int64) error
	Write(b []byte) (n int, err error)
}

// FileImage describes the representation of a SIF file in memory
type FileImage struct {
	Header     Header        // the loaded SIF global header
	Fp         ReadWriter    // file pointer of opened SIF file
	Filesize   int64         // file size of the opened SIF file
	Filedata   []byte        // the content of the opened file
	Amodebuf   bool          // access mode: mmap = false, buffered = true
	Reader     *bytes.Reader // reader on top of Mapdata
	DescrArr   []Descriptor  // slice of loaded descriptors from SIF file
	PrimPartID uint32        // ID of primary system partition if present
}

// CreateInfo wraps all SIF file creation info needed
type CreateInfo struct {
	Pathname   string            // the end result output filename
	Launchstr  string            // the shell run command
	Sifversion string            // the SIF specification version used
	ID         uuid.UUID         // image unique identifier
	InputDescr []DescriptorInput // slice of input info for descriptor creation
}

// DescriptorInput describes the common info needed to create a data object descriptor
type DescriptorInput struct {
	Datatype  Datatype // datatype being harvested for new descriptor
	Groupid   uint32   // group to be set for new descriptor
	Link      uint32   // link to be set for new descriptor
	Size      int64    // size of the data object for the new descriptor
	Alignment int      // Align requirement for data object
	Encrypt   bool

	Fname string    // file containing data associated with the new descriptor
	Fp    io.Reader // file pointer to opened 'fname'
	Data  []byte    // loaded data from file

	Image *FileImage  // loaded SIF file in memory
	Descr *Descriptor // created end result descriptor

	Extra bytes.Buffer // where specific input type store their data
}
