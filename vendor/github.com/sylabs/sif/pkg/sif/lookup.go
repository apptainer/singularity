// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
// Copyright (c) 2017, Yannick Cote <yhcote@gmail.com> All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sif

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// ErrNotFound is the code for when no search key is not found
var ErrNotFound = errors.New("no match found")

// ErrMultValues is the code for when search key is not unique
var ErrMultValues = errors.New("lookup would return more than one match")

//
// Methods on (fimg *FIleImage)
//

// GetSIFArch returns the SIF arch code from go runtime arch code
func GetSIFArch(goarch string) (sifarch string) {
	var ok bool

	archMap := map[string]string{
		"386":      HdrArch386,
		"amd64":    HdrArchAMD64,
		"arm":      HdrArchARM,
		"arm64":    HdrArchARM64,
		"ppc64":    HdrArchPPC64,
		"ppc64le":  HdrArchPPC64le,
		"mips":     HdrArchMIPS,
		"mipsle":   HdrArchMIPSle,
		"mips64":   HdrArchMIPS64,
		"mips64le": HdrArchMIPS64le,
		"s390x":    HdrArchS390x,
	}

	if sifarch, ok = archMap[goarch]; !ok {
		sifarch = HdrArchUnknown
	}
	return sifarch
}

// GetGoArch returns the go runtime arch code from the SIF arch code
func GetGoArch(sifarch string) (goarch string) {
	var ok bool

	archMap := map[string]string{
		HdrArch386:      "386",
		HdrArchAMD64:    "amd64",
		HdrArchARM:      "arm",
		HdrArchARM64:    "arm64",
		HdrArchPPC64:    "ppc64",
		HdrArchPPC64le:  "ppc64le",
		HdrArchMIPS:     "mips",
		HdrArchMIPSle:   "mipsle",
		HdrArchMIPS64:   "mips64",
		HdrArchMIPS64le: "mips64le",
		HdrArchS390x:    "s390x",
	}

	if goarch, ok = archMap[sifarch]; !ok {
		goarch = "unknown"
	}
	return goarch
}

// GetHeader returns the loaded SIF global header
func (fimg *FileImage) GetHeader() *Header {
	return &fimg.Header
}

// GetFromDescrID searches for a descriptor with
func (fimg *FileImage) GetFromDescrID(id uint32) (*Descriptor, int, error) {
	var match = -1

	for i, v := range fimg.DescrArr {
		if !v.Used {
			continue
		} else {
			if v.ID == id {
				if match != -1 {
					return nil, -1, ErrMultValues
				}
				match = i
			}
		}
	}

	if match == -1 {
		return nil, -1, ErrNotFound
	}

	return &fimg.DescrArr[match], match, nil
}

// GetPartFromGroup searches for partition descriptors inside a specific group
func (fimg *FileImage) GetPartFromGroup(groupid uint32) ([]*Descriptor, []int, error) {
	var descrs []*Descriptor
	var indexes []int
	var count int

	for i, v := range fimg.DescrArr {
		if !v.Used {
			continue
		} else {
			if v.Datatype == DataPartition && v.Groupid == groupid {
				indexes = append(indexes, i)
				descrs = append(descrs, &fimg.DescrArr[i])
				count++
			}
		}
	}

	if count == 0 {
		return nil, nil, ErrNotFound
	}

	return descrs, indexes, nil
}

// GetSignFromGroup searches for signature descriptors inside a specific group
func (fimg *FileImage) GetSignFromGroup(groupid uint32) ([]*Descriptor, []int, error) {
	var descrs []*Descriptor
	var indexes []int
	var count int

	for i, v := range fimg.DescrArr {
		if !v.Used {
			continue
		} else {
			if v.Datatype == DataSignature && v.Groupid == groupid {
				indexes = append(indexes, i)
				descrs = append(descrs, &fimg.DescrArr[i])
				count++
			}
		}
	}

	if count == 0 {
		return nil, nil, ErrNotFound
	}

	return descrs, indexes, nil
}

// GetFromLinkedDescr searches for descriptors that point to "id"
func (fimg *FileImage) GetFromLinkedDescr(ID uint32) ([]*Descriptor, []int, error) {
	var descrs []*Descriptor
	var indexes []int
	var count int

	for i, v := range fimg.DescrArr {
		if !v.Used {
			continue
		} else {
			if v.Link == ID {
				indexes = append(indexes, i)
				descrs = append(descrs, &fimg.DescrArr[i])
				count++
			}
		}
	}

	if count == 0 {
		return nil, nil, ErrNotFound
	}

	return descrs, indexes, nil
}

// GetFromDescr searches for descriptors comparing all non-nil fields of a provided descriptor
func (fimg *FileImage) GetFromDescr(descr Descriptor) ([]*Descriptor, []int, error) {
	var descrs []*Descriptor
	var indexes []int
	var count int

	for i, v := range fimg.DescrArr {
		if !v.Used {
			continue
		} else {
			if descr.Datatype != 0 && descr.Datatype != v.Datatype {
				continue
			}
			if descr.ID != 0 && descr.ID != v.ID {
				continue
			}
			if descr.Groupid != 0 && descr.Groupid != v.Groupid {
				continue
			}
			if descr.Link != 0 && descr.Link != v.Link {
				continue
			}
			if descr.Fileoff != 0 && descr.Fileoff != v.Fileoff {
				continue
			}
			if descr.Filelen != 0 && descr.Filelen != v.Filelen {
				continue
			}
			if descr.Storelen != 0 && descr.Storelen != v.Storelen {
				continue
			}
			if descr.Ctime != 0 && descr.Ctime != v.Ctime {
				continue
			}
			if descr.Mtime != 0 && descr.Mtime != v.Mtime {
				continue
			}
			if descr.UID != 0 && descr.UID != v.UID {
				continue
			}
			if descr.Gid != 0 && descr.Gid != v.Gid {
				continue
			}
			if descr.Name[0] != 0 && !bytes.Equal(descr.Name[:], v.Name[:]) {
				continue
			}

			indexes = append(indexes, i)
			descrs = append(descrs, &fimg.DescrArr[i])
			count++
		}
	}

	if count == 0 {
		return nil, nil, ErrNotFound
	}

	return descrs, indexes, nil
}

//
// Methods on (descr *Descriptor)
//

// GetData return a memory mapped byte slice mirroring the data object in a SIF file.
func (descr *Descriptor) GetData(fimg *FileImage) []byte {
	if fimg.Amodebuf {
		if _, err := fimg.Fp.Seek(descr.Fileoff, 0); err != nil {
			return nil
		}
		data := make([]byte, descr.Filelen)
		if n, _ := fimg.Fp.Read(data); int64(n) != descr.Filelen {
			return nil
		}
		return data
	}

	return fimg.Filedata[descr.Fileoff : descr.Fileoff+descr.Filelen]
}

// GetName returns the name tag associated with the descriptor. Analogous to file name.
func (descr *Descriptor) GetName() string {
	return strings.TrimRight(string(descr.Name[:]), "\000")
}

// GetFsType extracts the Fstype field from the Extra field of a Partition Descriptor
func (descr *Descriptor) GetFsType() (Fstype, error) {
	if descr.Datatype != DataPartition {
		return -1, fmt.Errorf("expected DataPartition, got %v", descr.Datatype)
	}

	var pinfo Partition
	b := bytes.NewReader(descr.Extra[:])
	if err := binary.Read(b, binary.LittleEndian, &pinfo); err != nil {
		return -1, fmt.Errorf("while extracting Partition extra info: %s", err)
	}

	return pinfo.Fstype, nil
}

// GetPartType extracts the Parttype field from the Extra field of a Partition Descriptor
func (descr *Descriptor) GetPartType() (Parttype, error) {
	if descr.Datatype != DataPartition {
		return -1, fmt.Errorf("expected DataPartition, got %v", descr.Datatype)
	}

	var pinfo Partition
	b := bytes.NewReader(descr.Extra[:])
	if err := binary.Read(b, binary.LittleEndian, &pinfo); err != nil {
		return -1, fmt.Errorf("while extracting Partition extra info: %s", err)
	}

	return pinfo.Parttype, nil
}

// GetArch extracts the Arch field from the Extra field of a Partition Descriptor
func (descr *Descriptor) GetArch() ([HdrArchLen]byte, error) {
	if descr.Datatype != DataPartition {
		return [HdrArchLen]byte{}, fmt.Errorf("expected DataPartition, got %v", descr.Datatype)
	}

	var pinfo Partition
	b := bytes.NewReader(descr.Extra[:])
	if err := binary.Read(b, binary.LittleEndian, &pinfo); err != nil {
		return [HdrArchLen]byte{}, fmt.Errorf("while extracting Partition extra info: %s", err)
	}

	return pinfo.Arch, nil
}

// GetHashType extracts the Hashtype field from the Extra field of a Signature Descriptor
func (descr *Descriptor) GetHashType() (Hashtype, error) {
	if descr.Datatype != DataSignature {
		return -1, fmt.Errorf("expected DataSignature, got %v", descr.Datatype)
	}

	var sinfo Signature
	b := bytes.NewReader(descr.Extra[:])
	if err := binary.Read(b, binary.LittleEndian, &sinfo); err != nil {
		return -1, fmt.Errorf("while extracting Signature extra info: %s", err)
	}

	return sinfo.Hashtype, nil
}

// GetEntity extracts the signing entity field from the Extra field of a Signature Descriptor
func (descr *Descriptor) GetEntity() ([]byte, error) {
	if descr.Datatype != DataSignature {
		return nil, fmt.Errorf("expected DataSignature, got %v", descr.Datatype)
	}

	var sinfo Signature
	b := bytes.NewReader(descr.Extra[:])
	if err := binary.Read(b, binary.LittleEndian, &sinfo); err != nil {
		return nil, fmt.Errorf("while extracting Signature extra info: %s", err)
	}

	return sinfo.Entity[:], nil
}

// GetEntityString returns the string version of the stored entity
func (descr *Descriptor) GetEntityString() (string, error) {
	fingerprint, err := descr.GetEntity()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%0X", fingerprint[:20]), nil
}

// GetPartPrimSys returns the primary system partition if present. There should
// be only one primary system partition in a SIF file.
func (fimg *FileImage) GetPartPrimSys() (*Descriptor, int, error) {
	var descr *Descriptor
	index := -1

	for i, v := range fimg.DescrArr {
		if !v.Used {
			continue
		} else {
			if v.Datatype == DataPartition {
				ptype, err := v.GetPartType()
				if err != nil {
					return nil, -1, err
				}
				if ptype == PartPrimSys {
					if index != -1 {
						return nil, -1, ErrMultValues
					}
					index = i
					descr = &fimg.DescrArr[i]
				}
			}
		}
	}

	if index == -1 {
		return nil, -1, ErrNotFound
	}

	return descr, index, nil
}
