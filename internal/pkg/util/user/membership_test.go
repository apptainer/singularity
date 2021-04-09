// Copyright (c) 2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package user

import (
	"fmt"
	"strconv"
	"testing"
)

func TestUserInList(t *testing.T) {
	u, err := Current()
	if err != nil {
		t.Fatalf("Could not identify current user for test: %v", err)
	}

	type args struct {
		uid  int
		list []string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:    "NotInList",
			args:    args{int(u.UID), []string{"9999", "notauser"}},
			want:    false,
			wantErr: false,
		},
		{
			name:    "InListUid",
			args:    args{int(u.UID), []string{"9999", "notauser", strconv.Itoa(int(u.UID))}},
			want:    true,
			wantErr: false,
		},
		{
			name:    "InListName",
			args:    args{int(u.UID), []string{"9999", "notauser", u.Name}},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UIDInList(tt.args.uid, tt.args.list)
			if (err != nil) != tt.wantErr {
				t.Errorf("UIDInList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UIDInList() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserInGroup(t *testing.T) {
	u, err := current()
	if err != nil {
		t.Fatalf("Could not identify current user for test: %v", err)
	}
	g, err := currentGroup()
	if err != nil {
		t.Fatalf("Could not identify current group for test: %v", err)
	}

	type args struct {
		uid  int
		list []string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:    "NotInList",
			args:    args{int(u.UID), []string{"9999", "notagroup"}},
			want:    false,
			wantErr: false,
		},
		{
			name:    "InListUid",
			args:    args{int(u.UID), []string{"9999", "notagroup", fmt.Sprintf("%d", g.GID)}},
			want:    true,
			wantErr: false,
		},
		{
			name:    "InListName",
			args:    args{int(u.UID), []string{"9999", "notagroup", g.Name}},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UIDInAnyGroup(tt.args.uid, tt.args.list)
			if (err != nil) != tt.wantErr {
				t.Errorf("UIDInAnyGroup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UIDInAnyGroup() got = %v, want %v", got, tt.want)
			}
		})
	}
}
