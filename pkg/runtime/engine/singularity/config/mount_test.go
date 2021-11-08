// Copyright (c) 2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"reflect"
	"testing"
)

func TestParseMountString(t *testing.T) {
	tests := []struct {
		name        string
		mountString string
		want        []BindPath
		wantErr     bool
	}{
		{
			name:        "sourceOnly",
			mountString: "type=bind,source=/opt",
			want:        []BindPath{},
			wantErr:     true,
		},
		{
			name:        "destinationOnly",
			mountString: "type=bind,destination=/opt",
			want:        []BindPath{},
			wantErr:     true,
		},
		{
			name:        "emptySource",
			mountString: "type=bind,source=,destination=/opt",
			want:        []BindPath{},
			wantErr:     true,
		},
		{
			name:        "emptyDestination",
			mountString: "type=bind,source=/opt,destination=",
			want:        []BindPath{},
			wantErr:     true,
		},
		{
			name:        "invalidType",
			mountString: "type=potato,source=/opt,destination=/opt",
			want:        []BindPath{},
			wantErr:     true,
		},
		{
			name:        "invalidField",
			mountString: "type=bind,source=/opt,destination=/opt,color=turquoise",
			want:        []BindPath{},
			wantErr:     true,
		},
		{
			name:        "simple",
			mountString: "type=bind,source=/opt,destination=/opt",
			want: []BindPath{
				{
					Source:      "/opt",
					Destination: "/opt",
					Options:     map[string]*BindOption{},
				},
			},
			wantErr: false,
		},
		{
			name:        "simpleSrc",
			mountString: "type=bind,src=/opt,destination=/opt",
			want: []BindPath{
				{
					Source:      "/opt",
					Destination: "/opt",
					Options:     map[string]*BindOption{},
				},
			},
			wantErr: false,
		},
		{
			name:        "simpleDst",
			mountString: "type=bind,source=/opt,dst=/opt",
			want: []BindPath{
				{
					Source:      "/opt",
					Destination: "/opt",
					Options:     map[string]*BindOption{},
				},
			},
			wantErr: false,
		},
		{
			name:        "simpleTarget",
			mountString: "type=bind,source=/opt,target=/opt",
			want: []BindPath{
				{
					Source:      "/opt",
					Destination: "/opt",
					Options:     map[string]*BindOption{},
				},
			},
			wantErr: false,
		},
		{
			name:        "noType",
			mountString: "source=/opt,destination=/opt",
			want: []BindPath{
				{
					Source:      "/opt",
					Destination: "/opt",
					Options:     map[string]*BindOption{},
				},
			},
			wantErr: false,
		},
		{
			name:        "ro",
			mountString: "type=bind,source=/opt,destination=/opt,ro",
			want: []BindPath{
				{
					Source:      "/opt",
					Destination: "/opt",
					Options: map[string]*BindOption{
						"ro": {},
					},
				},
			},
			wantErr: false,
		},
		{
			name:        "readonly",
			mountString: "type=bind,source=/opt,destination=/opt,readonly",
			want: []BindPath{
				{
					Source:      "/opt",
					Destination: "/opt",
					Options: map[string]*BindOption{
						"ro": {},
					},
				},
			},
			wantErr: false,
		},
		{
			name:        "imagesrc",
			mountString: "type=bind,source=test.sif,destination=/opt,image-src=/opt",
			want: []BindPath{
				{
					Source:      "test.sif",
					Destination: "/opt",
					Options: map[string]*BindOption{
						"image-src": {Value: "/opt"},
					},
				},
			},
			wantErr: false,
		},
		{
			name:        "imagesrcNoValue",
			mountString: "type=bind,source=test.sif,destination=/opt,image-src",
			want:        []BindPath{},
			wantErr:     true,
		},
		{
			name:        "imagesrcEmpty",
			mountString: "type=bind,source=test.sif,destination=/opt,image-src=",
			want:        []BindPath{},
			wantErr:     true,
		},
		{
			name:        "id",
			mountString: "type=bind,source=test.sif,destination=/opt,image-src=/opt,id=2",
			want: []BindPath{
				{
					Source:      "test.sif",
					Destination: "/opt",
					Options: map[string]*BindOption{
						"image-src": {Value: "/opt"},
						"id":        {Value: "2"},
					},
				},
			},
			wantErr: false,
		},
		{
			name:        "idNoValue",
			mountString: "type=bind,source=test.sif,destination=/opt,image-src=/opt,id",
			want:        []BindPath{},
			wantErr:     true,
		},
		{
			name:        "idEmpty",
			mountString: "type=bind,source=test.sif,destination=/opt,image-src=/opt,id=",
			want:        []BindPath{},
			wantErr:     true,
		},
		{
			name:        "bindpropagation",
			mountString: "type=bind,source=/opt,destination=/opt,bind-propagation=shared",
			want:        []BindPath{},
			wantErr:     true,
		},
		{
			name:        "csvEscaped",
			mountString: `type=bind,"source=/comma,dir","destination=/quote""dir"`,
			want: []BindPath{
				{
					Source:      "/comma,dir",
					Destination: "/quote\"dir",
					Options:     map[string]*BindOption{},
				},
			},
			wantErr: false,
		},
		{
			name:        "multiple",
			mountString: "type=bind,source=/opt,destination=/opt\ntype=bind,source=/srv,destination=/srv",
			want: []BindPath{
				{
					Source:      "/opt",
					Destination: "/opt",
					Options:     map[string]*BindOption{},
				},
				{
					Source:      "/srv",
					Destination: "/srv",
					Options:     map[string]*BindOption{},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMountString(tt.mountString)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMountString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseMountString() = %v, want %v", got, tt.want)
			}
		})
	}
}
