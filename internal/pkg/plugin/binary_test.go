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

func (r testSifReader) IsUsed(n int) bool {
	return r[n].used
}

func (r testSifReader) GetDatatype(n int) sif.Datatype {
	return r[n].datatype
}

func (r testSifReader) GetFsType(n int) (sif.Fstype, error) {
	return r[n].fstype, r[n].fserror
}

func (r testSifReader) GetPartType(n int) (sif.Parttype, error) {
	return r[n].parttype, r[n].parterror
}

func (r testSifReader) GetData(n int) []byte {
	return r[n].data
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
					datatype: sif.DataDeffile,
				},
				{
					used:     true,
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
					datatype: sif.DataPartition,
					fstype:   sif.FsRaw,
					fserror:  errors.New("invalid filesystem"),
				},
				{
					used:     true,
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
					datatype: sif.DataPartition,
					fstype:   sif.FsRaw,
					parttype: sif.PartSystem,
				},
				{
					used:     true,
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
					datatype:  sif.DataPartition,
					fstype:    sif.FsRaw,
					parttype:  sif.PartData,
					parterror: errors.New("invalid partition"),
				},
				{
					used:     true,
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
					datatype: sif.DataPartition,
					fstype:   sif.FsRaw,
					parttype: sif.PartData,
				},
				{
					used: false,
				},
			},
			expected: false,
		},
		{
			description: "first descriptor good, bad second descriptor",
			sif: testSifReader{
				{
					used:     true,
					datatype: sif.DataPartition,
					fstype:   sif.FsRaw,
					parttype: sif.PartData,
				},
				{
					used:     true,
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
					datatype: sif.DataPartition,
					fstype:   sif.FsRaw,
					parttype: sif.PartData,
				},
				{
					used:     true,
					datatype: sif.DataGenericJSON,
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
				{used: true},
				{used: true},
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
					data: []byte(testGoodJSON),
				},
			},
			expected: pluginapi.Manifest{
				Name:        "test name",
				Author:      "test author",
				Version:     "test version",
				Description: "test description",
			},
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
