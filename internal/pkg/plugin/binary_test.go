// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package plugin

import (
	"errors"
	"testing"

	"github.com/sylabs/sif/pkg/sif"
	pluginapi "github.com/sylabs/singularity/pkg/plugin"
)

type testSifReader []struct {
	name      string
	data      []byte
	used      bool
	datatype  sif.Datatype
	fstype    sif.Fstype
	parttype  sif.Parttype
	fserror   error
	parterror error
}

func (r testSifReader) Descriptors() int {
	return len(r)
}

func (r testSifReader) IsUsed(name string) bool {
	n := r.findByName(name)
	if n < 0 {
		return false
	}
	return r[n].used
}

func (r testSifReader) GetDatatype(name string) sif.Datatype {
	n := r.findByName(name)
	if n < 0 {
		return -1
	}
	return r[n].datatype
}

func (r testSifReader) GetFsType(name string) (sif.Fstype, error) {
	n := r.findByName(name)
	if n < 0 {
		return -1, r[n].fserror
	}
	return r[n].fstype, r[n].fserror
}

func (r testSifReader) GetPartType(name string) (sif.Parttype, error) {
	n := r.findByName(name)
	if n < 0 {
		return -1, r[n].parterror
	}
	return r[n].parttype, r[n].parterror
}

func (r testSifReader) GetData(name string) []byte {
	n := r.findByName(name)
	if n < 0 {
		return nil
	}
	return r[n].data
}

func (r testSifReader) findByName(name string) int {
	for n, data := range r {
		if data.name == name {
			return n
		}
	}

	return -1
}

func TestIsPluginFile(t *testing.T) {
	cases := []struct {
		description string
		sif         testSifReader
		expected    bool
	}{
		{
			description: "nil image",
			sif:         nil,
			expected:    false,
		},
		{
			description: "short image",
			sif:         testSifReader{{used: true}},
			expected:    false,
		},
		{
			description: "empty image",
			sif: testSifReader{
				{used: false},
				{used: false},
			},
			expected: false,
		},
		{
			description: "wrong data type",
			sif: testSifReader{
				{
					used:     true,
					name:     pluginBinaryName,
					datatype: sif.DataDeffile,
				},
				{
					used:     true,
					name:     pluginManifestName,
					datatype: sif.DataGenericJSON,
				},
			},
			expected: false,
		},
		{
			description: "error for fs type",
			sif: testSifReader{
				{
					used:     true,
					name:     pluginBinaryName,
					datatype: sif.DataPartition,
					fstype:   sif.FsRaw,
					fserror:  errors.New("invalid filesystem"),
				},
				{
					used:     true,
					name:     pluginManifestName,
					datatype: sif.DataGenericJSON,
				},
			},
			expected: false,
		},
		{
			description: "wrong partition type",
			sif: testSifReader{
				{
					used:     true,
					name:     pluginBinaryName,
					datatype: sif.DataPartition,
					fstype:   sif.FsRaw,
					parttype: sif.PartSystem,
				},
				{
					used:     true,
					name:     pluginManifestName,
					datatype: sif.DataGenericJSON,
				},
			},
			expected: false,
		},
		{
			description: "error for partition type",
			sif: testSifReader{
				{
					used:      true,
					name:      pluginBinaryName,
					datatype:  sif.DataPartition,
					fstype:    sif.FsRaw,
					parttype:  sif.PartData,
					parterror: errors.New("invalid partition"),
				},
				{
					used:     true,
					name:     pluginManifestName,
					datatype: sif.DataGenericJSON,
				},
			},
			expected: false,
		},
		{
			description: "first descriptor good, no second descriptor",
			sif: testSifReader{
				{
					used:     true,
					name:     pluginBinaryName,
					datatype: sif.DataPartition,
					fstype:   sif.FsRaw,
					parttype: sif.PartData,
				},
				{
					used: false,
					name: pluginManifestName,
				},
			},
			expected: false,
		},
		{
			description: "first descriptor good, bad second descriptor",
			sif: testSifReader{
				{
					used:     true,
					name:     pluginBinaryName,
					datatype: sif.DataPartition,
					fstype:   sif.FsRaw,
					parttype: sif.PartData,
				},
				{
					used:     true,
					name:     pluginManifestName,
					datatype: sif.DataDeffile,
				},
			},
			expected: false,
		},
		{
			description: "good image",
			sif: testSifReader{
				{
					used:     true,
					name:     pluginBinaryName,
					datatype: sif.DataPartition,
					fstype:   sif.FsRaw,
					parttype: sif.PartData,
				},
				{
					used:     true,
					name:     pluginManifestName,
					datatype: sif.DataGenericJSON,
				},
			},
			expected: true,
		},
		{
			description: "good image out of order",
			sif: testSifReader{
				{
					used:     true,
					name:     pluginManifestName,
					datatype: sif.DataGenericJSON,
				},
				{
					used:     true,
					name:     pluginBinaryName,
					datatype: sif.DataPartition,
					fstype:   sif.FsRaw,
					parttype: sif.PartData,
				},
			},
			expected: true,
		},
		{
			description: "good image extra descriptors",
			sif: testSifReader{
				{
					used:     true,
					name:     pluginManifestName,
					datatype: sif.DataGenericJSON,
				},
				{
					used:     true,
					name:     pluginBinaryName,
					datatype: sif.DataPartition,
					fstype:   sif.FsRaw,
					parttype: sif.PartData,
				},
				{
					used:     true,
					name:     "signature",
					datatype: sif.DataSignature,
				},
			},
			expected: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			if actual := isPluginFile(tc.sif); actual != tc.expected {
				t.Errorf("%s: isPluginFile retuned %v, expected %v",
					tc.description,
					actual,
					tc.expected)
				t.Fail()
			}
		})
	}
}

func TestGetManifest(t *testing.T) {
	testGoodJSON := `{"name":"test name", "author":"test author", "version":"test version", "description":"test description"}`
	testGoodManifest := pluginapi.Manifest{
		Name:        "test name",
		Author:      "test author",
		Version:     "test version",
		Description: "test description",
	}
	testBadJSON := `{123`

	cases := []struct {
		description string
		sif         testSifReader
		expected    pluginapi.Manifest
	}{
		{
			description: "nil image",
			sif:         nil,
			expected:    pluginapi.Manifest{},
		},
		{
			description: "short image",
			sif:         testSifReader{{used: true}},
			expected:    pluginapi.Manifest{},
		},
		{
			description: "empty image",
			sif: testSifReader{
				{used: false},
				{used: false},
			},
			expected: pluginapi.Manifest{},
		},
		{
			description: "empty manifest",
			sif: testSifReader{
				{
					used: true,
				},
				{
					used: true,
					name: pluginManifestName,
				},
			},
			expected: pluginapi.Manifest{},
		},
		{
			description: "bad JSON",
			sif: testSifReader{
				{
					used: true,
				},
				{
					used: true,
					name: pluginManifestName,
					data: []byte(testBadJSON),
				},
			},
			expected: pluginapi.Manifest{},
		},
		{
			description: "good manifest",
			sif: testSifReader{
				{
					used: true,
				},
				{
					used: true,
					name: pluginManifestName,
					data: []byte(testGoodJSON),
				},
			},
			expected: testGoodManifest,
		},
		{
			description: "good manifest out of order",
			sif: testSifReader{
				{
					used: true,
					name: pluginManifestName,
					data: []byte(testGoodJSON),
				},
				{
					used: true,
				},
			},
			expected: testGoodManifest,
		},
		{
			description: "good manifest extra descriptors",
			sif: testSifReader{
				{
					used: true,
				},
				{
					used: true,
				},
				{
					used: true,
					name: pluginManifestName,
					data: []byte(testGoodJSON),
				},
			},
			expected: testGoodManifest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			if actual := getManifest(tc.sif); actual != tc.expected {
				t.Errorf("%s: getManifest retuned %#v, expected %#v",
					tc.description,
					actual,
					tc.expected)
				t.Fail()
			}
		})
	}
}
