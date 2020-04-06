// Copyright (c) 2019-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package machine

import (
	"bufio"
	"bytes"
	"debug/elf"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/pkg/sylog"
)

// ErrUnknownArch is the error returned for unknown architecture.
var ErrUnknownArch = errors.New("architecture not recognized")

type format struct {
	Arch       string
	Sif        string
	Compatible string
	Machine    elf.Machine
	Class      elf.Class
	Endianness binary.ByteOrder
	ElfMagic   []byte
}

var formats = []format{
	{
		Arch:       "386",
		Sif:        sif.HdrArch386,
		Compatible: "amd64",
		Machine:    elf.EM_386,
		Class:      elf.ELFCLASS32,
		Endianness: binary.LittleEndian,
		ElfMagic:   []byte{0x7F, 0x45, 0x4C, 0x46, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x03, 0x00},
	},
	{
		Arch:       "386",
		Sif:        sif.HdrArch386,
		Compatible: "amd64",
		Machine:    elf.EM_486,
		Class:      elf.ELFCLASS32,
		Endianness: binary.LittleEndian,
		ElfMagic:   []byte{0x7F, 0x45, 0x4C, 0x46, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x06, 0x00},
	},
	{
		Arch:       "amd64",
		Sif:        sif.HdrArchAMD64,
		Machine:    elf.EM_X86_64,
		Class:      elf.ELFCLASS64,
		Endianness: binary.LittleEndian,
		ElfMagic:   []byte{0x7F, 0x45, 0x4C, 0x46, 0x02, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x3e, 0x00},
	},
	{
		Arch:       "arm",
		Sif:        sif.HdrArchARM,
		Compatible: "arm64",
		Machine:    elf.EM_ARM,
		Class:      elf.ELFCLASS32,
		Endianness: binary.LittleEndian,
		ElfMagic:   []byte{0x7F, 0x45, 0x4C, 0x46, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x28, 0x00},
	},
	{
		Arch:       "armbe",
		Sif:        sif.HdrArchARM, // FIXME: add HdrArchARMbe to sif package
		Compatible: "arm64be",
		Machine:    elf.EM_ARM,
		Class:      elf.ELFCLASS32,
		Endianness: binary.BigEndian,
		ElfMagic:   []byte{0x7F, 0x45, 0x4C, 0x46, 0x01, 0x02, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x28},
	},
	{
		Arch:       "arm64",
		Sif:        sif.HdrArchARM64,
		Machine:    elf.EM_AARCH64,
		Class:      elf.ELFCLASS64,
		Endianness: binary.LittleEndian,
		ElfMagic:   []byte{0x7F, 0x45, 0x4C, 0x46, 0x02, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0xb7, 0x00},
	},
	{
		Arch:       "arm64be",
		Sif:        sif.HdrArchARM64, // FIXME: add HdrArchARM64be to sif package
		Machine:    elf.EM_AARCH64,
		Class:      elf.ELFCLASS64,
		Endianness: binary.BigEndian,
		ElfMagic:   []byte{0x7F, 0x45, 0x4C, 0x46, 0x02, 0x02, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0xb7},
	},
	{
		Arch:       "s390x",
		Sif:        sif.HdrArchS390x,
		Machine:    elf.EM_S390,
		Class:      elf.ELFCLASS64,
		Endianness: binary.BigEndian,
		ElfMagic:   []byte{0x7F, 0x45, 0x4C, 0x46, 0x02, 0x02, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x16},
	},
	{
		Arch:       "ppc64",
		Sif:        sif.HdrArchPPC64,
		Machine:    elf.EM_PPC64,
		Class:      elf.ELFCLASS32,
		Endianness: binary.BigEndian,
		ElfMagic:   []byte{0x7F, 0x45, 0x4C, 0x46, 0x02, 0x02, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x15},
	},
	{
		Arch:       "ppc64le",
		Sif:        sif.HdrArchPPC64le,
		Machine:    elf.EM_PPC64,
		Class:      elf.ELFCLASS64,
		Endianness: binary.LittleEndian,
		ElfMagic:   []byte{0x7F, 0x45, 0x4C, 0x46, 0x02, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x15, 0x00},
	},
	{
		Arch:       "mips",
		Sif:        sif.HdrArchMIPS,
		Compatible: "mips64",
		Machine:    elf.EM_MIPS,
		Class:      elf.ELFCLASS32,
		Endianness: binary.BigEndian,
		ElfMagic:   []byte{0x7F, 0x45, 0x4C, 0x46, 0x01, 0x02, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x08},
	},
	{
		Arch:       "mipsle",
		Sif:        sif.HdrArchMIPSle,
		Compatible: "mips64le",
		Machine:    elf.EM_MIPS,
		Class:      elf.ELFCLASS32,
		Endianness: binary.LittleEndian,
		ElfMagic:   []byte{0x7F, 0x45, 0x4C, 0x46, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x08, 0x00},
	},
	{
		Arch:       "mips64",
		Sif:        sif.HdrArchMIPS64,
		Machine:    elf.EM_MIPS,
		Class:      elf.ELFCLASS64,
		Endianness: binary.BigEndian,
		ElfMagic:   []byte{0x7F, 0x45, 0x4C, 0x46, 0x02, 0x02, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x08},
	},
	{
		Arch:       "mips64le",
		Sif:        sif.HdrArchMIPS64le,
		Machine:    elf.EM_MIPS,
		Class:      elf.ELFCLASS64,
		Endianness: binary.LittleEndian,
		ElfMagic:   []byte{0x7F, 0x45, 0x4C, 0x46, 0x02, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x08, 0x00},
	},
}

// ArchFromElf returns the architecture string after inspection of the
// provided elf binary.
func ArchFromElf(binary string) (string, error) {
	e, err := elf.Open(binary)
	if err != nil {
		return "", fmt.Errorf("failed to open elf binary %s: %s", binary, err)
	}
	defer e.Close()

	for _, f := range formats {
		if f.Machine == e.Machine && f.Class == e.Class && f.Endianness == e.ByteOrder {
			return f.Arch, nil
		}
	}

	return "", ErrUnknownArch
}

// ArchFromContainer walks through a container filesystem until it
// find an elf binary to read target architecture from and returns it.
// If there is no suitable elf binary or if the architecture is not
// recognized it will return an empty string.
func ArchFromContainer(container string) string {
	// fast path if we can get architecture from shell binary
	shell := fs.EvalRelative("/bin/sh", container)
	arch, err := ArchFromElf(filepath.Join(container, shell))
	if err == nil {
		return arch
	}

	sylog.Verbosef("No /bin/sh in container, looking at executable files to find architecture")

	filepath.Walk(container, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.Mode().IsRegular() {
			return nil
		}
		// ignore not executable files
		if info.Mode().Perm()&0111 == 0 {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		elfMagic := make([]byte, len(elf.ELFMAG))
		if _, err := f.Read(elfMagic); err != nil {
			return nil
		}
		if string(elfMagic) == string(elf.ELFMAG) {
			arch, err = ArchFromElf(path)
			if err == ErrUnknownArch {
				return err
			} else if err != nil {
				return nil
			}
			return fmt.Errorf("found elf binary at %s", path)
		}
		return nil
	})

	return arch
}

const binfmtMisc = "/proc/sys/fs/binfmt_misc"

type binfmtEntry struct {
	magic      string
	enabled    bool
	persistent bool
}

func canEmulate(arch string) bool {
	var format format

	for _, f := range formats {
		if arch == f.Arch {
			format = f
			break
		}
	}

	// no architecture format found
	if format.Arch == "" {
		return false
	}

	// look at /proc/sys/fs/binfmt_misc
	content, _ := ioutil.ReadFile(filepath.Join(binfmtMisc, "status"))
	if string(content) != "enabled\n" {
		return false
	}

	infos, err := ioutil.ReadDir(binfmtMisc)
	if err != nil {
		return false
	}

	archMagic := hex.EncodeToString(format.ElfMagic)

	for _, fi := range infos {
		f := filepath.Join(binfmtMisc, fi.Name())
		b, err := ioutil.ReadFile(f)
		if err != nil {
			continue
		}

		entry := new(binfmtEntry)

		scanner := bufio.NewScanner(bytes.NewReader(b))
		for scanner.Scan() {
			t := scanner.Text()

			if t == "enabled" {
				entry.enabled = true
			} else if strings.HasPrefix(t, "magic") {
				splitted := strings.Split(t, " ")
				if len(splitted) > 1 {
					entry.magic = splitted[1]
				}
			} else if strings.HasPrefix(t, "flags") {
				splitted := strings.Split(t, " ")
				if len(splitted) > 1 {
					entry.persistent = strings.Contains(splitted[1], "F")
				}
			}
		}

		if entry.enabled && entry.persistent && entry.magic == archMagic {
			return true
		}
	}

	return false
}

// CompatibleWith returns if the current machine architecture is
// compatible or can run via emulation the architecture passed in
// argument.
func CompatibleWith(arch string) bool {
	currentArch := runtime.GOARCH

	if currentArch == arch {
		return true
	}

	for _, f := range formats {
		if arch == f.Arch && f.Compatible == currentArch {
			return true
		}
	}

	return canEmulate(arch)
}
