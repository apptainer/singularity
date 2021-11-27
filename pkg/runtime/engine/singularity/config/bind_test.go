// Copyright (c) 2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"reflect"
	"testing"
)

func TestParseBindPath(t *testing.T) {
	tests := []struct {
		name      string
		bindpaths []string
		want      []BindPath
		wantErr   bool
	}{
		{
			name:      "srcOnly",
			bindpaths: []string{"/opt"},
			want: []BindPath{
				{
					Source:      "/opt",
					Destination: "/opt",
				},
			},
		},
		{
			name:      "srcOnlyMultiple",
			bindpaths: []string{"/opt,/tmp"},
			want: []BindPath{
				{
					Source:      "/opt",
					Destination: "/opt",
				},
				{
					Source:      "/tmp",
					Destination: "/tmp",
				},
			},
		},
		{
			name:      "srcDst",
			bindpaths: []string{"/opt:/other"},
			want: []BindPath{
				{
					Source:      "/opt",
					Destination: "/other",
				},
			},
		},
		{
			name:      "srcDstMultiple",
			bindpaths: []string{"/opt:/other,/tmp:/other2,"},
			want: []BindPath{
				{
					Source:      "/opt",
					Destination: "/other",
				},
				{
					Source:      "/tmp",
					Destination: "/other2",
				},
			},
		},
		{
			name:      "srcDstRO",
			bindpaths: []string{"/opt:/other:ro"},
			want: []BindPath{
				{
					Source:      "/opt",
					Destination: "/other",
					Options: map[string]*BindOption{
						"ro": {},
					},
				},
			},
		},
		{
			name:      "srcDstROMultiple",
			bindpaths: []string{"/opt:/other:ro,/tmp:/other2:ro"},
			want: []BindPath{
				{
					Source:      "/opt",
					Destination: "/other",
					Options: map[string]*BindOption{
						"ro": {},
					},
				},
				{
					Source:      "/tmp",
					Destination: "/other2",
					Options: map[string]*BindOption{
						"ro": {},
					},
				},
			},
		},
		{
			// This doesn't make functional sense (ro & rw), but is testing
			// parsing multiple simple options.
			name:      "srcDstRORW",
			bindpaths: []string{"/opt:/other:ro,rw"},
			want: []BindPath{
				{
					Source:      "/opt",
					Destination: "/other",
					Options: map[string]*BindOption{
						"ro": {},
						"rw": {},
					},
				},
			},
		},
		{
			// This doesn't make functional sense (ro & rw), but is testing
			// parsing multiple binds, with multiple options each. Note the
			// complex parsing here that has to distinguish between comma
			// delimiting an additional option, vs an additional bind.
			name:      "srcDstRORWMultiple",
			bindpaths: []string{"/opt:/other:ro,rw,/tmp:/other2:ro,rw"},
			want: []BindPath{
				{
					Source:      "/opt",
					Destination: "/other",
					Options: map[string]*BindOption{
						"ro": {},
						"rw": {},
					},
				},
				{
					Source:      "/tmp",
					Destination: "/other2",
					Options: map[string]*BindOption{
						"ro": {},
						"rw": {},
					},
				},
			},
		},
		{
			name:      "srcDstImageSrc",
			bindpaths: []string{"test.sif:/other:image-src=/opt"},
			want: []BindPath{
				{
					Source:      "test.sif",
					Destination: "/other",
					Options: map[string]*BindOption{
						"image-src": {"/opt"},
					},
				},
			},
		},
		{
			// Can't use image-src without a value
			name:      "srcDstImageSrcNoVal",
			bindpaths: []string{"test.sif:/other:image-src"},
			want:      []BindPath{},
			wantErr:   true,
		},
		{
			name:      "srcDstId",
			bindpaths: []string{"test.sif:/other:image-src=/opt,id=2"},
			want: []BindPath{
				{
					Source:      "test.sif",
					Destination: "/other",
					Options: map[string]*BindOption{
						"image-src": {"/opt"},
						"id":        {"2"},
					},
				},
			},
		},
		{
			name:      "invalidOption",
			bindpaths: []string{"/opt:/other:invalid"},
			want:      []BindPath{},
			wantErr:   true,
		},
		{
			name:      "invalidSpec",
			bindpaths: []string{"/opt:/other:rw:invalid"},
			want:      []BindPath{},
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBindPath(tt.bindpaths)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBindPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseBindPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
