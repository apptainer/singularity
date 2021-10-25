// Copyright (c) 2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package gpu

import (
	"os"
	"reflect"
	"sort"
	"testing"
)

func TestNVCLIEnvToFlags(t *testing.T) {
	tests := []struct {
		name      string
		env       map[string]string
		wantFlags []string
		wantErr   bool
	}{
		{
			name: "defaults",
			wantFlags: []string{
				"--no-cgroups",
				"--compute",
				"--utility",
			},
			wantErr: false,
		},
		{
			name: "device",
			env: map[string]string{
				"NVIDIA_VISIBLE_DEVICES": "all",
			},
			wantFlags: []string{
				"--no-cgroups",
				"--device=all",
				"--compute",
				"--utility",
			},
			wantErr: false,
		},
		{
			name: "mig-config",
			env: map[string]string{
				"NVIDIA_MIG_CONFIG_DEVICES": "all",
			},
			wantFlags: []string{
				"--no-cgroups",
				"--mig-config=all",
				"--compute",
				"--utility",
			},
			wantErr: false,
		},
		{
			name: "mig-monitor",
			env: map[string]string{
				"NVIDIA_MIG_MONITOR_DEVICES": "all",
			},
			wantFlags: []string{
				"--no-cgroups",
				"--mig-monitor=all",
				"--compute",
				"--utility",
			},
			wantErr: false,
		},
		{
			name: "compute-only",
			env: map[string]string{
				"NVIDIA_DRIVER_CAPABILITIES": "compute",
			},
			wantFlags: []string{
				"--no-cgroups",
				"--compute",
			},
			wantErr: false,
		},
		{
			name: "all-caps",
			env: map[string]string{
				"NVIDIA_DRIVER_CAPABILITIES": "compute,compat32,graphics,utility,video,display",
			},
			wantFlags: []string{
				"--no-cgroups",
				"--compute",
				"--compat32",
				"--graphics",
				"--utility",
				"--video",
				"--display",
			},
			wantErr: false,
		},
		{
			name: "invalid-caps",
			env: map[string]string{
				"NVIDIA_DRIVER_CAPABILITIES": "notacap",
			},
			wantErr: true,
		},
		{
			name: "single-require",
			env: map[string]string{
				"NVIDIA_REQUIRE_CUDA": "cuda>=9.0",
			},
			wantFlags: []string{
				"--no-cgroups",
				"--compute",
				"--utility",
				"--require=cuda>=9.0",
			},
			wantErr: false,
		},
		{
			name: "multi-require",
			env: map[string]string{
				"NVIDIA_REQUIRE_BRAND": "brand=GRID",
				"NVIDIA_REQUIRE_CUDA":  "cuda>=9.0",
			},
			wantFlags: []string{
				"--no-cgroups",
				"--compute",
				"--utility",
				"--require=brand=GRID",
				"--require=cuda>=9.0",
			},
			wantErr: false,
		},
		{
			name: "disable-require",
			env: map[string]string{
				"NVIDIA_REQUIRE_BRAND":   "brand=GRID",
				"NVIDIA_REQUIRE_CUDA":    "cuda>=9.0",
				"NVIDIA_DISABLE_REQUIRE": "1",
			},
			wantFlags: []string{
				"--no-cgroups",
				"--compute",
				"--utility",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, val := range tt.env {
				os.Setenv(key, val)
				defer os.Unsetenv(key)
			}

			gotFlags, err := NVCLIEnvToFlags()
			if (err != nil) != tt.wantErr {
				t.Errorf("NVCLIEnvToFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			sort.Strings(gotFlags)
			sort.Strings(tt.wantFlags)
			if !reflect.DeepEqual(gotFlags, tt.wantFlags) {
				t.Errorf("NVCLIEnvToFlags() = %v, want %v", gotFlags, tt.wantFlags)
			}
		})
	}
}
