// Copyright (c) 2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package user

import (
	"fmt"
	"os/user"
	"testing"
)

func TestUserInList(t *testing.T) {
	u, err := user.Current()
	if err != nil{
		t.Fatalf("Could not identify current user for test: %v", err)
	}

	type args struct {
		uid  string
		list []string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "NotInList",
			args: args{u.Uid, []string{"0", "root"}},
			want: false,
			wantErr: false,
		},
		{
			name: "InListUid",
			args: args{u.Uid, []string{"0", "root", u.Uid}},
			want: true,
			wantErr: false,
		},
		{
			name: "InListName",
			args: args{u.Uid, []string{"0", "root", u.Name}},
			want: true,
			wantErr: false,
		},
	}
		for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UserInList(tt.args.uid, tt.args.list)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserInList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UserInList() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserInGroup(t *testing.T) {
	u, err := user.Current()
	if err != nil{
		t.Fatalf("Could not identify current user for test: %v", err)
	}
	g, err := currentGroup()
	if err != nil{
		t.Fatalf("Could not identify current group for test: %v", err)
	}

	type args struct {
		uid  string
		list []string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "NotInList",
			args: args{u.Uid, []string{"0", "root"}},
			want: false,
			wantErr: false,
		},
		{
			name: "InListUid",
			args: args{u.Uid, []string{"0", "root", fmt.Sprintf("%d",g.GID)}},
			want: true,
			wantErr: false,
		},
		{
			name: "InListName",
			args: args{u.Uid, []string{"0", "root", g.Name}},
			want: true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UserInGroup(tt.args.uid, tt.args.list)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserInGroup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UserInGroup() got = %v, want %v", got, tt.want)
			}
		})
	}
}