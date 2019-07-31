// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	uuid "github.com/satori/go.uuid"
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

const testSquash = "./testdata/squashfs.v4"

func createSIF(t *testing.T, inputDesc []sif.DescriptorInput, corrupted bool) string {
	sifFile, err := fs.MakeTmpFile("", "sif-", 0644)
	if err != nil {
		t.Fatalf("failed to create temporary file: %s", err)
	}
	sifFile.Close()

	for _, d := range inputDesc {
		if f, ok := d.Fp.(*os.File); ok {
			f.Seek(0, 0)
		}
	}

	cinfo := sif.CreateInfo{
		Pathname:   sifFile.Name(),
		Launchstr:  sif.HdrLaunch,
		Sifversion: sif.HdrVersion,
		ID:         uuid.NewV4(),
		InputDescr: inputDesc,
	}

	fp, err := sif.CreateContainer(cinfo)
	if err != nil {
		t.Fatalf("failed to create empty SIF")
	}
	fp.UnloadContainer()

	if corrupted {
		f, err := os.OpenFile(sifFile.Name(), os.O_WRONLY, 0)
		if err != nil {
			t.Fatalf("failed to open %s: %s", sifFile.Name(), err)
		}
		defer f.Close()

		fi, err := f.Stat()
		if err != nil {
			t.Fatalf("failed to stat on %s: %s", sifFile.Name(), err)
		}
		if err := f.Truncate(fi.Size() - 4096); err != nil {
			t.Fatalf("failed to truncate file to %d: %s", fi.Size()-4096, err)
		}
	}

	return sifFile.Name()
}

func TestSIFInitializer(t *testing.T) {
	fp1, err := os.Open(testSquash)
	if err != nil {
		t.Fatalf("failed to open %s: %s", testSquash, err)
	}
	defer fp1.Close()

	fp2, err := os.Open(testSquash)
	if err != nil {
		t.Fatalf("failed to open %s: %s", testSquash, err)
	}
	defer fp2.Close()

	onePart := sif.DescriptorInput{
		Datatype: sif.DataPartition,
		Groupid:  sif.DescrDefaultGroup,
		Link:     sif.DescrUnusedLink,
		Fname:    "onePart",
		Fp:       fp1,
		Extra: *bytes.NewBuffer([]byte{
			0x01, 0x00, 0x00, 0x00, // fstype
			0x01, 0x00, 0x00, 0x00, // part type
		}),
	}

	oneSection := sif.DescriptorInput{
		Datatype: sif.DataGeneric,
		Groupid:  sif.DescrDefaultGroup,
		Link:     sif.DescrUnusedLink,
		Fname:    "oneSection",
		Fp:       fp1,
	}

	primPartNoArch := sif.DescriptorInput{
		Datatype: sif.DataPartition,
		Groupid:  sif.DescrDefaultGroup,
		Link:     sif.DescrUnusedLink,
		Fname:    "primPart",
		Fp:       fp1,
		Extra: *bytes.NewBuffer([]byte{
			0x01, 0x00, 0x00, 0x00, // fstype
			0x02, 0x00, 0x00, 0x00, // part type
		}),
	}

	primPart := sif.DescriptorInput{
		Datatype: sif.DataPartition,
		Groupid:  sif.DescrDefaultGroup,
		Link:     sif.DescrUnusedLink,
		Fname:    "primPart",
		Fp:       fp1,
		Extra: *bytes.NewBuffer([]byte{
			0x01, 0x00, 0x00, 0x00, // fstype
			0x02, 0x00, 0x00, 0x00, // part type
		}),
	}
	primPart.Extra.WriteString(sif.GetSIFArch(runtime.GOARCH))

	overlayPart := sif.DescriptorInput{
		Datatype: sif.DataPartition,
		Groupid:  sif.DescrDefaultGroup,
		Link:     sif.DescrUnusedLink,
		Fname:    "overlayPart",
		Fp:       fp2,
		Extra: *bytes.NewBuffer([]byte{
			0x01, 0x00, 0x00, 0x00, // fstype
			0x04, 0x00, 0x00, 0x00, // part type
		}),
	}

	tests := []struct {
		name               string
		path               string
		writable           bool
		expectedSuccess    bool
		expectedPartitions int
		expectedSections   int
	}{
		{
			name:               "NoPartitionSIF",
			path:               createSIF(t, nil, false),
			writable:           false,
			expectedSuccess:    false,
			expectedPartitions: 0,
			expectedSections:   0,
		},
		{
			name:               "UnkownPartitionSIF",
			path:               createSIF(t, []sif.DescriptorInput{onePart}, false),
			writable:           false,
			expectedSuccess:    true,
			expectedPartitions: 0,
			expectedSections:   0,
		},
		{
			name:               "PrimaryPartitionNoArchSIF",
			path:               createSIF(t, []sif.DescriptorInput{primPartNoArch}, false),
			writable:           false,
			expectedSuccess:    false,
			expectedPartitions: 0,
			expectedSections:   0,
		},
		{
			name:               "PrimaryPartitionSIF",
			path:               createSIF(t, []sif.DescriptorInput{primPart}, false),
			writable:           false,
			expectedSuccess:    true,
			expectedPartitions: 1,
			expectedSections:   0,
		},
		{
			name:               "PrimaryPartitionCorruptedSIF",
			path:               createSIF(t, []sif.DescriptorInput{primPart}, true),
			writable:           false,
			expectedSuccess:    false,
			expectedPartitions: 0,
			expectedSections:   0,
		},
		{
			name:               "PrimaryAndOverlayPartitionsSIF",
			path:               createSIF(t, []sif.DescriptorInput{primPart, overlayPart}, false),
			writable:           false,
			expectedSuccess:    true,
			expectedPartitions: 2,
			expectedSections:   0,
		},
		{
			name:               "SectionSIF",
			path:               createSIF(t, []sif.DescriptorInput{oneSection}, false),
			writable:           false,
			expectedSuccess:    true,
			expectedPartitions: 0,
			expectedSections:   1,
		},
		{
			name:               "PartitionAndSectionSIF",
			path:               createSIF(t, []sif.DescriptorInput{primPart, oneSection}, false),
			writable:           false,
			expectedSuccess:    true,
			expectedPartitions: 1,
			expectedSections:   1,
		},
	}

	for _, tt := range tests {
		var err error

		sifFmt := new(sifFormat)
		mode := sifFmt.openMode(tt.writable)

		img := &Image{
			Path: tt.path,
			Name: tt.path,
		}

		img.Writable = true
		img.File, err = os.OpenFile(tt.path, mode, 0)
		if err != nil {
			t.Fatalf("cannot open image's file: %s\n", err)
		}
		defer img.File.Close()

		fileinfo, err := img.File.Stat()
		if err != nil {
			t.Fatalf("cannot stat the image file: %s\n", err)
		}

		err = sifFmt.initializer(img, fileinfo)
		os.Remove(tt.path)

		if err != nil && tt.expectedSuccess {
			t.Fatalf("unexpected error for %q: %s\n", tt.name, err)
		} else if err == nil && !tt.expectedSuccess {
			t.Fatalf("unexpected success for %q", tt.name)
		} else if tt.expectedPartitions != len(img.Partitions) {
			t.Fatalf("unexpected partitions number: %d instead of %d", len(img.Partitions), tt.expectedPartitions)
		} else if tt.expectedSections != len(img.Sections) {
			t.Fatalf("unexpected sections number: %d instead of %d", len(img.Sections), tt.expectedSections)
		}
	}
}

func TestSIFOpenMode(t *testing.T) {
	var sifFmt sifFormat

	if sifFmt.openMode(true) != os.O_RDWR {
		t.Fatal("openMode(true) returned the wrong value")
	}
	if sifFmt.openMode(false) != os.O_RDONLY {
		t.Fatal("openMode(false) returned the wrong value")
	}
}
