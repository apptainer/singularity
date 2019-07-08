// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// Copyright (c) 2018, Divya Cote <divya.cote@gmail.com> All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sif

import (
	"fmt"
	"time"
)

// readableSize returns the size in human readable format
func readableSize(size uint64) string {
	var divs int
	var conversion string

	for ; size != 0; size >>= 10 {
		if size < 1024 {
			break
		}
		divs++
	}

	switch divs {
	case 0:
		conversion = fmt.Sprintf("%d", size)
	case 1:
		conversion = fmt.Sprintf("%dKB", size)
	case 2:
		conversion = fmt.Sprintf("%dMB", size)
	case 3:
		conversion = fmt.Sprintf("%dGB", size)
	case 4:
		conversion = fmt.Sprintf("%dTB", size)
	}
	return conversion
}

// FmtHeader formats the output of a SIF file global header
func (fimg *FileImage) FmtHeader() string {
	s := fmt.Sprintln("Launch:  ", cstrToString(fimg.Header.Launch[:]))
	s += fmt.Sprintln("Magic:   ", cstrToString(fimg.Header.Magic[:]))
	s += fmt.Sprintln("Version: ", cstrToString(fimg.Header.Version[:]))
	s += fmt.Sprintln("Arch:    ", GetGoArch(cstrToString(fimg.Header.Arch[:])))
	s += fmt.Sprintln("ID:      ", fimg.Header.ID)
	s += fmt.Sprintln("Ctime:   ", time.Unix(fimg.Header.Ctime, 0))
	s += fmt.Sprintln("Mtime:   ", time.Unix(fimg.Header.Mtime, 0))
	s += fmt.Sprintln("Dfree:   ", fimg.Header.Dfree)
	s += fmt.Sprintln("Dtotal:  ", fimg.Header.Dtotal)
	s += fmt.Sprintln("Descoff: ", fimg.Header.Descroff)
	s += fmt.Sprintln("Descrlen:", readableSize(uint64(fimg.Header.Descrlen)))
	s += fmt.Sprintln("Dataoff: ", fimg.Header.Dataoff)
	s += fmt.Sprintln("Datalen: ", readableSize(uint64(fimg.Header.Datalen)))

	return s
}

// datatypeStr returns a string representation of a datatype
func datatypeStr(dtype Datatype) string {
	switch dtype {
	case DataDeffile:
		return "Def.FILE"
	case DataEnvVar:
		return "Env.Vars"
	case DataLabels:
		return "JSON.Labels"
	case DataPartition:
		return "FS"
	case DataSignature:
		return "Signature"
	case DataGenericJSON:
		return "JSON.Generic"
	case DataGeneric:
		return "Generic/Raw"
	}
	return "Unknown data-type"
}

// fstypeStr returns a string representation of a file system type
func fstypeStr(ftype Fstype) string {
	switch ftype {
	case FsSquash:
		return "Squashfs"
	case FsExt3:
		return "Ext3"
	case FsImmuObj:
		return "Archive"
	case FsRaw:
		return "Raw"
	case FsEncryptedSquashfs:
		return "Encrypted squashfs"
	}
	return "Unknown fs-type"
}

// parttypeStr returns a string representation of a partition type
func parttypeStr(ptype Parttype) string {
	switch ptype {
	case PartSystem:
		return "System"
	case PartPrimSys:
		return "*System"
	case PartData:
		return "Data"
	case PartOverlay:
		return "Overlay"
	}
	return "Unknown part-type"
}

// hashtypeStr returns a string representation of a  hash type
func hashtypeStr(htype Hashtype) string {
	switch htype {
	case HashSHA256:
		return "SHA256"
	case HashSHA384:
		return "SHA384"
	case HashSHA512:
		return "SHA512"
	case HashBLAKE2S:
		return "BLAKE2S"
	case HashBLAKE2B:
		return "BLAKE2B"
	}
	return "Unknown hash-type"
}

// FmtDescrList formats the output of a list of all active descriptors from a SIF file
func (fimg *FileImage) FmtDescrList() string {
	s := fmt.Sprintf("%-4s %-8s %-8s %-26s %s\n", "ID", "|GROUP", "|LINK", "|SIF POSITION (start-end)", "|TYPE")
	s += fmt.Sprintln("------------------------------------------------------------------------------")

	for _, v := range fimg.DescrArr {
		if !v.Used {
			continue
		} else {
			s += fmt.Sprintf("%-4d ", v.ID)
			if v.Groupid == DescrUnusedGroup {
				s += fmt.Sprintf("|%-7s ", "NONE")
			} else {
				s += fmt.Sprintf("|%-7d ", v.Groupid&^DescrGroupMask)
			}
			if v.Link == DescrUnusedLink {
				s += fmt.Sprintf("|%-7s ", "NONE")
			} else {
				if v.Link&DescrGroupMask == DescrGroupMask {
					s += fmt.Sprintf("|%-3d (G) ", v.Link&^DescrGroupMask)
				} else {
					s += fmt.Sprintf("|%-7d ", v.Link)
				}
			}

			fposbuf := fmt.Sprintf("|%d-%d ", v.Fileoff, v.Fileoff+v.Filelen)
			s += fmt.Sprintf("%-26s ", fposbuf)

			switch v.Datatype {
			case DataPartition:
				f, _ := v.GetFsType()
				p, _ := v.GetPartType()
				a, _ := v.GetArch()
				s += fmt.Sprintf("|%s (%s/%s/%s)\n", datatypeStr(v.Datatype), fstypeStr(f), parttypeStr(p), GetGoArch(cstrToString(a[:])))
			case DataSignature:
				h, _ := v.GetHashType()
				s += fmt.Sprintf("|%s (%s)\n", datatypeStr(v.Datatype), hashtypeStr(h))
			default:
				s += fmt.Sprintf("|%s\n", datatypeStr(v.Datatype))
			}
		}
	}

	return s
}

// FmtDescrInfo formats the output of detailed info about a descriptor from a SIF file
func (fimg *FileImage) FmtDescrInfo(id uint32) string {
	var s string

	for i, v := range fimg.DescrArr {
		if !v.Used {
			continue
		} else if v.ID == uint32(id) {
			s = fmt.Sprintln("Descr slot#:", i)
			s += fmt.Sprintln("  Datatype: ", datatypeStr(v.Datatype))
			s += fmt.Sprintln("  ID:       ", v.ID)
			s += fmt.Sprintln("  Used:     ", v.Used)
			if v.Groupid == DescrUnusedGroup {
				s += fmt.Sprintln("  Groupid:  ", "NONE")
			} else {
				s += fmt.Sprintln("  Groupid:  ", v.Groupid&^DescrGroupMask)
			}
			if v.Link == DescrUnusedLink {
				s += fmt.Sprintln("  Link:     ", "NONE")
			} else {
				if v.Link&DescrGroupMask == DescrGroupMask {
					s += fmt.Sprintln("  Link:     ", v.Link&^DescrGroupMask, "(G)")
				} else {
					s += fmt.Sprintln("  Link:     ", v.Link)
				}
			}
			s += fmt.Sprintln("  Fileoff:  ", v.Fileoff)
			s += fmt.Sprintln("  Filelen:  ", v.Filelen)
			s += fmt.Sprintln("  Ctime:    ", time.Unix(v.Ctime, 0))
			s += fmt.Sprintln("  Mtime:    ", time.Unix(v.Mtime, 0))
			s += fmt.Sprintln("  UID:      ", v.UID)
			s += fmt.Sprintln("  Gid:      ", v.Gid)
			s += fmt.Sprintln("  Name:     ", string(v.Name[:]))
			switch v.Datatype {
			case DataPartition:
				f, _ := v.GetFsType()
				p, _ := v.GetPartType()
				a, _ := v.GetArch()
				s += fmt.Sprintln("  Fstype:   ", fstypeStr(f))
				s += fmt.Sprintln("  Parttype: ", parttypeStr(p))
				s += fmt.Sprintln("  Arch:     ", GetGoArch(cstrToString(a[:])))
			case DataSignature:
				h, _ := v.GetHashType()
				e, _ := v.GetEntityString()
				s += fmt.Sprintln("  Hashtype: ", hashtypeStr(h))
				s += fmt.Sprintln("  Entity:   ", e)
			}

			return s
		}
	}

	return ""
}
