// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

//go:build singularity_engine
// +build singularity_engine

package unpacker

import (
	"reflect"
	"strings"
	"testing"
)

// Library listing on Fedora 35 AMD64 - simple case
// $ /lib64/ld-linux-x86-64.so.2 --list /usr/sbin/unsquashfs
const ldListSimple = `        linux-vdso.so.1 (0x00007ffe9ebcb000)
        libm.so.6 => /lib64/libm.so.6 (0x00007f5dd1e6a000)
        libz.so.1 => /lib64/libz.so.1 (0x00007f5dd1e50000)
        liblzma.so.5 => /lib64/liblzma.so.5 (0x00007f5dd1e24000)
        liblzo2.so.2 => /lib64/liblzo2.so.2 (0x00007f5dd1e03000)
        liblz4.so.1 => /lib64/liblz4.so.1 (0x00007f5dd1ddf000)
        libzstd.so.1 => /lib64/libzstd.so.1 (0x00007f5dd1d30000)
        libgcc_s.so.1 => /lib64/libgcc_s.so.1 (0x00007f5dd1d13000)
        libc.so.6 => /lib64/libc.so.6 (0x00007f5dd1b09000)
        /lib64/ld-linux-x86-64.so.2 (0x00007f5dd2104000)`

// Library listing on EL8 POWER8 - complex case
// glibc-hwcaps and dependency filename not matching resolved filename
// $ /lib64/ld64.so.2 --list /usr/sbin/unsquashfs
const ldListComplex = `        linux-vdso64.so.1 (0x00007fff80d70000)
        libpthread.so.0 => /lib64/glibc-hwcaps/power9/libpthread-2.28.so (0x00007fff80b50000)
        libm.so.6 => /lib64/glibc-hwcaps/power9/libm-2.28.so (0x00007fff80a20000)
        libz.so.1 => /lib64/libz.so.1 (0x00007fff809e0000)
        liblzma.so.5 => /lib64/liblzma.so.5 (0x00007fff80980000)
        liblzo2.so.2 => /lib64/liblzo2.so.2 (0x00007fff80930000)
        liblz4.so.1 => /lib64/liblz4.so.1 (0x00007fff808e0000)
        libc.so.6 => /lib64/glibc-hwcaps/power9/libc-2.28.so (0x00007fff806d0000)
        /lib64/ld64.so.2 (0x00007fff80d90000)`

// Library listing on EL7 - old case
// The linux-vdso.so.1 line has a => field that doesn't point to an absolute path
const ldListOld = `        linux-vdso.so.1 =>  (0x00007ffccf1de000)
        libpthread.so.0 => /lib64/libpthread.so.0 (0x00007f5ab0e3d000)
        libm.so.6 => /lib64/libm.so.6 (0x00007f5ab0b3b000)
        libz.so.1 => /lib64/libz.so.1 (0x00007f5ab0925000)
        liblzma.so.5 => /lib64/liblzma.so.5 (0x00007f5ab06ff000)
        liblzo2.so.2 => /lib64/liblzo2.so.2 (0x00007f5ab04de000)
        libgcc_s.so.1 => /lib64/libgcc_s.so.1 (0x00007f5ab02c8000)
        libc.so.6 => /lib64/libc.so.6 (0x00007f5aafefa000)
        /lib64/ld-linux-x86-64.so.2 (0x00007f5ab1059000)`

func Test_parseLibraryBinds(t *testing.T) {
	tests := []struct {
		name    string
		ldList  string
		want    []libBind
		wantErr bool
	}{
		{
			name:    "empty",
			ldList:  "",
			want:    []libBind{},
			wantErr: false,
		},
		{
			name:   "simple",
			ldList: ldListSimple,
			want: []libBind{
				{"/lib64/libm.so.6", "/lib64/libm.so.6"},
				{"/lib64/libz.so.1", "/lib64/libz.so.1"},
				{"/lib64/liblzma.so.5", "/lib64/liblzma.so.5"},
				{"/lib64/liblzo2.so.2", "/lib64/liblzo2.so.2"},
				{"/lib64/liblz4.so.1", "/lib64/liblz4.so.1"},
				{"/lib64/libzstd.so.1", "/lib64/libzstd.so.1"},
				{"/lib64/libgcc_s.so.1", "/lib64/libgcc_s.so.1"},
				{"/lib64/libc.so.6", "/lib64/libc.so.6"},
				{"/lib64/ld-linux-x86-64.so.2", "/lib64/ld-linux-x86-64.so.2"},
			},
			wantErr: false,
		},
		{
			name:   "complex",
			ldList: ldListComplex,
			want: []libBind{
				{
					"/lib64/glibc-hwcaps/power9/libpthread-2.28.so",
					"/lib64/glibc-hwcaps/power9/libpthread.so.0",
				},
				{"/lib64/glibc-hwcaps/power9/libm-2.28.so", "/lib64/glibc-hwcaps/power9/libm.so.6"},
				{"/lib64/libz.so.1", "/lib64/libz.so.1"},
				{"/lib64/liblzma.so.5", "/lib64/liblzma.so.5"},
				{"/lib64/liblzo2.so.2", "/lib64/liblzo2.so.2"},
				{"/lib64/liblz4.so.1", "/lib64/liblz4.so.1"},
				{"/lib64/glibc-hwcaps/power9/libc-2.28.so", "/lib64/glibc-hwcaps/power9/libc.so.6"},
				{"/lib64/ld64.so.2", "/lib64/ld64.so.2"},
			},
			wantErr: false,
		},
		{
			name:   "old",
			ldList: ldListOld,
			want: []libBind{
				{"/lib64/libpthread.so.0", "/lib64/libpthread.so.0"},
				{"/lib64/libm.so.6", "/lib64/libm.so.6"},
				{"/lib64/libz.so.1", "/lib64/libz.so.1"},
				{"/lib64/liblzma.so.5", "/lib64/liblzma.so.5"},
				{"/lib64/liblzo2.so.2", "/lib64/liblzo2.so.2"},
				{"/lib64/libgcc_s.so.1", "/lib64/libgcc_s.so.1"},
				{"/lib64/libc.so.6", "/lib64/libc.so.6"},
				{"/lib64/ld-linux-x86-64.so.2", "/lib64/ld-linux-x86-64.so.2"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := strings.NewReader(tt.ldList)
			got, err := parseLibraryBinds(buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLibraryBinds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseLibraryBinds() = %v, want %v", got, tt.want)
			}
		})
	}
}
