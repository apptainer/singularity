// Copyright (c) 2019-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	"github.com/hpcng/sif/v2/pkg/sif"
	"github.com/hpcng/singularity/internal/pkg/util/fs"
)

const testSquash = "./testdata/squashfs.v4"

func createSIF(t *testing.T, corrupted bool, fns ...func() (sif.DescriptorInput, error)) string {
	sifFile, err := fs.MakeTmpFile("", "sif-", 0o644)
	if err != nil {
		t.Fatalf("failed to create temporary file: %s", err)
	}
	sifFile.Close()

	var opts []sif.CreateOpt

	for _, fn := range fns {
		di, err := fn()
		if err != nil {
			t.Fatalf("failed to get DescriptorInput: %v", err)
		}

		opts = append(opts, sif.OptCreateWithDescriptors(di))
	}

	fp, err := sif.CreateContainerAtPath(sifFile.Name(), opts...)
	if err != nil {
		t.Fatalf("failed to create SIF: %v", err)
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
	b, err := os.ReadFile(testSquash)
	if err != nil {
		t.Fatalf("failed to read %s: %s", testSquash, err)
	}

	onePart := func() (sif.DescriptorInput, error) {
		return sif.NewDescriptorInput(sif.DataPartition, bytes.NewReader(b),
			sif.OptPartitionMetadata(sif.FsSquash, sif.PartSystem, runtime.GOARCH),
		)
	}

	oneSection := func() (sif.DescriptorInput, error) {
		return sif.NewDescriptorInput(sif.DataGeneric, bytes.NewReader(b))
	}

	primPartOtherArch := func() (sif.DescriptorInput, error) {
		return sif.NewDescriptorInput(sif.DataPartition, bytes.NewReader(b),
			sif.OptPartitionMetadata(sif.FsSquash, sif.PartPrimSys, "s390x"),
		)
	}

	primPart := func() (sif.DescriptorInput, error) {
		return sif.NewDescriptorInput(sif.DataPartition, bytes.NewReader(b),
			sif.OptPartitionMetadata(sif.FsSquash, sif.PartPrimSys, runtime.GOARCH),
		)
	}

	overlayPart := func() (sif.DescriptorInput, error) {
		return sif.NewDescriptorInput(sif.DataPartition, bytes.NewReader(b),
			sif.OptPartitionMetadata(sif.FsSquash, sif.PartOverlay, runtime.GOARCH),
		)
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
			path:               createSIF(t, false),
			writable:           false,
			expectedSuccess:    true,
			expectedPartitions: 0,
			expectedSections:   0,
		},
		{
			name:               "UnkownPartitionSIF",
			path:               createSIF(t, false, onePart),
			writable:           false,
			expectedSuccess:    true,
			expectedPartitions: 0,
			expectedSections:   0,
		},
		{
			name:               "PrimaryPartitionOtherArchSIF",
			path:               createSIF(t, false, primPartOtherArch),
			writable:           false,
			expectedSuccess:    false,
			expectedPartitions: 0,
			expectedSections:   0,
		},
		{
			name:               "PrimaryPartitionSIF",
			path:               createSIF(t, false, primPart),
			writable:           false,
			expectedSuccess:    true,
			expectedPartitions: 1,
			expectedSections:   0,
		},
		{
			name:               "PrimaryPartitionCorruptedSIF",
			path:               createSIF(t, true, primPart),
			writable:           false,
			expectedSuccess:    false,
			expectedPartitions: 0,
			expectedSections:   0,
		},
		{
			name:               "PrimaryAndOverlayPartitionsSIF",
			path:               createSIF(t, false, primPart, overlayPart),
			writable:           false,
			expectedSuccess:    true,
			expectedPartitions: 2,
			expectedSections:   0,
		},
		{
			name:               "SectionSIF",
			path:               createSIF(t, false, oneSection),
			writable:           false,
			expectedSuccess:    true,
			expectedPartitions: 0,
			expectedSections:   1,
		},
		{
			name:               "PartitionAndSectionSIF",
			path:               createSIF(t, false, primPart, oneSection),
			writable:           false,
			expectedSuccess:    true,
			expectedPartitions: 1,
			expectedSections:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error

			sifFmt := new(sifFormat)
			mode := sifFmt.openMode(tt.writable)

			img := &Image{
				Path: tt.path,
				Name: tt.path,
			}

			img.Writable = tt.writable
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

			if (err == nil) != tt.expectedSuccess {
				t.Fatalf("got error %v, expect success %v", err, tt.expectedSuccess)
			} else if tt.expectedPartitions != len(img.Partitions) {
				t.Fatalf("unexpected partitions number: %d instead of %d", len(img.Partitions), tt.expectedPartitions)
			} else if tt.expectedSections != len(img.Sections) {
				t.Fatalf("unexpected sections number: %d instead of %d", len(img.Sections), tt.expectedSections)
			}
		})
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
