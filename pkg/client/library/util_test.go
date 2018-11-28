// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"reflect"
	"strings"
	"testing"

	"github.com/globalsign/mgo/bson"
	"github.com/sylabs/singularity/internal/pkg/test"
)

func Test_isLibraryPullRef(t *testing.T) {
	tests := []struct {
		name       string
		libraryRef string
		want       bool
	}{
		{"Good long ref 1", "library://entity/collection/image:tag", true},
		{"Good long ref 2", "entity/collection/image:tag", true},
		{"Good long ref 3", "entity/collection/image", true},
		{"Good short ref 1", "library://image:tag", true},
		{"Good short ref 2", "library://image", true},
		{"Good short ref 3", "library://collection/image:tag", true},
		{"Good short ref 4", "library://image", true},
		{"Good long sha ref 1", "library://entity/collection/image:sha256.e50a30881ace3d5944f5661d222db7bee5296be9e4dc7c1fcb7604bcae926e88", true},
		{"Good long sha ref 2", "entity/collection/image:sha256.e50a30881ace3d5944f5661d222db7bee5296be9e4dc7c1fcb7604bcae926e88", true},
		{"Good short sha ref 1", "library://image:sha256.e50a30881ace3d5944f5661d222db7bee5296be9e4dc7c1fcb7604bcae926e88", true},
		{"Good short sha ref 2", "image:sha256.e50a30881ace3d5944f5661d222db7bee5296be9e4dc7c1fcb7604bcae926e88", true},
		{"Good short sha ref 3", "library://collection/image:sha256.e50a30881ace3d5944f5661d222db7bee5296be9e4dc7c1fcb7604bcae926e88", true},
		{"Good short sha ref 4", "collection/image:sha256.e50a30881ace3d5944f5661d222db7bee5296be9e4dc7c1fcb7604bcae926e88", true},
		{"Too many components", "library://entity/collection/extra/image:tag", false},
		{"Bad character", "library://entity/collection/im,age:tag", false},
		{"Bad initial character", "library://entity/collection/-image:tag", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if got := IsLibraryPullRef(tt.libraryRef); got != tt.want {
				t.Errorf("isLibraryPullRef() = %v, want %v", got, tt.want)
			}
		}))
	}
}

func Test_isLibraryPushRef(t *testing.T) {
	tests := []struct {
		name       string
		libraryRef string
		want       bool
	}{
		{"Good long ref 1", "library://entity/collection/image:tag", true},
		{"Good long ref 2", "entity/collection/image:tag", true},
		{"Good long ref 3", "entity/collection/image", true},
		{"Short ref not allowed 1", "library://image:tag", false},
		{"Short ref not allowed 2", "library://image", false},
		{"Short ref not allowed 3", "library://collection/image:tag", false},
		{"Short ref not allowed 4", "library://image", false},
		{"Good long sha ref 1", "library://entity/collection/image:sha256.e50a30881ace3d5944f5661d222db7bee5296be9e4dc7c1fcb7604bcae926e88", true},
		{"Good long sha ref 2", "entity/collection/image:sha256.e50a30881ace3d5944f5661d222db7bee5296be9e4dc7c1fcb7604bcae926e88", true},
		{"Too many components", "library://entity/collection/extra/image:tag", false},
		{"Bad character", "library://entity/collection/im,age:tag", false},
		{"Bad initial character", "library://entity/collection/-image:tag", false},
		{"No capitals", "library://Entity/collection/image:tag", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if got := IsLibraryPushRef(tt.libraryRef); got != tt.want {
				t.Errorf("isLibraryPushRef() = %v, want %v", got, tt.want)
			}
		}))
	}
}

func Test_IsRefPart(t *testing.T) {
	tests := []struct {
		name       string
		libraryRef string
		want       bool
	}{
		{"Good ref 1", "abc123", true},
		{"Good ref 2", "abc-123", true},
		{"Good ref 3", "abc_123", true},
		{"Good ref 4", "abc.123", true},
		{"Bad character", "abc,123", false},
		{"Bad initial character", "-abc123", false},
		{"No capitals", "Abc123", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if got := IsRefPart(tt.libraryRef); got != tt.want {
				t.Errorf("IsRefPart() = %v, want %v", got, tt.want)
			}
		}))
	}
}

func Test_IsImageHash(t *testing.T) {
	tests := []struct {
		name       string
		libraryRef string
		want       bool
	}{
		{"Good sha256", "sha256.e50a30881ace3d5944f5661d222db7bee5296be9e4dc7c1fcb7604bcae926e88", true},
		{"Good sif", "sif.5574b72c-7705-49cc-874e-424fc3b78116", true},
		{"sha256 too long", "sha256.e50a30881ace3d5944f5661d222db7bee5296be9e4dc7c1fcb7604bcae926e88a", false},
		{"sha256 too short", "sha256.e50a30881ace3d5944f5661d222db7bee5296be9e4dc7c1fcb7604bcae926e8", false},
		{"sha256 bad character", "sha256.g50a30881ace3d5944f5661d222db7bee5296be9e4dc7c1fcb7604bcae926e88", false},
		{"sif too long", "sif.5574b72c-7705-49cc-874e-424fc3b78116a", false},
		{"sif too short", "sif.5574b72c-7705-49cc-874e-424fc3b7811", false},
		{"sif bad character", "sif.g574b72c-7705-49cc-874e-424fc3b78116", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if got := IsImageHash(tt.libraryRef); got != tt.want {
				t.Errorf("IsImageHash() = %v, want %v", got, tt.want)
			}
		}))
	}
}

func Test_parseLibraryRef(t *testing.T) {
	tests := []struct {
		name       string
		libraryRef string
		wantEnt    string
		wantCol    string
		wantCon    string
		wantTags   []string
	}{
		{"Good long ref 1", "library://entity/collection/image:tag", "entity", "collection", "image", []string{"tag"}},
		{"Good long ref 2", "entity/collection/image:tag", "entity", "collection", "image", []string{"tag"}},
		{"Good long ref latest", "library://entity/collection/image", "entity", "collection", "image", []string{"latest"}},
		{"Good long ref multi tag", "library://entity/collection/image:tag1,tag2,tag3", "entity", "collection", "image", []string{"tag1", "tag2", "tag3"}},
		{"Good short ref 1", "library://image:tag", "", "", "image", []string{"tag"}},
		{"Good short ref 2", "image:tag", "", "", "image", []string{"tag"}},
		{"Good short ref 3", "library://collection/image:tag", "", "collection", "image", []string{"tag"}},
		{"Good short ref 4", "collection/image:tag", "", "collection", "image", []string{"tag"}},
		{"Good short ref latest", "library://image", "", "", "image", []string{"latest"}},
		{"Good short ref multi tag", "library://image:tag1,tag2,tag3", "", "", "image", []string{"tag1", "tag2", "tag3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			ent, col, con, tags := parseLibraryRef(tt.libraryRef)
			if ent != tt.wantEnt {
				t.Errorf("parseLibraryRef() = entity %v, want %v", ent, tt.wantEnt)
			}
			if col != tt.wantCol {
				t.Errorf("parseLibraryRef() = collection %v, want %v", col, tt.wantCol)
			}
			if con != tt.wantCon {
				t.Errorf("parseLibraryRef() = container %v, want %v", con, tt.wantCon)
			}
			if !reflect.DeepEqual(tags, tt.wantTags) {
				t.Errorf("parseLibraryRef() = entity %v, want %v", tags, tt.wantTags)
			}
		}))
	}
}

func Test_ParseErrorBody(t *testing.T) {

	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	eb := JSONError{
		Code:    500,
		Status:  "Internal Server Error",
		Message: "The server had a problem",
	}
	ebJSON := "{ \"error\": {\"code\": 500, \"status\": \"Internal Server Error\", \"message\": \"The server had a problem\"}}"
	r := strings.NewReader(ebJSON)

	jRes, err := ParseErrorBody(r)

	if err != nil {
		t.Errorf("Decoding good error response did not succeed: %v", err)
	}

	if !reflect.DeepEqual(jRes.Error, eb) {
		t.Errorf("Decoding error body expected %v, got %v", eb, jRes)
	}

	ebJSON = "{ \"error {\"code\": 500, \"status\": \"Internal Server Error\", \"message\": \"The server had a problem\"}}"
	jRes, err = ParseErrorBody(r)

	if err == nil {
		t.Errorf("Decoding bad error response succeeded, but should return an error: %v", ebJSON)
	}

}

func TestIdInSlice(t *testing.T) {

	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	trueID := bson.NewObjectId()

	slice := []bson.ObjectId{trueID, bson.NewObjectId(), bson.NewObjectId(), bson.NewObjectId()}
	if !IDInSlice(trueID, slice) {
		t.Errorf("should find %v in %v", trueID, slice)
	}

	slice = []bson.ObjectId{bson.NewObjectId(), bson.NewObjectId(), trueID, bson.NewObjectId()}
	if !IDInSlice(trueID, slice) {
		t.Errorf("should find %v in %v", trueID, slice)
	}

	slice = []bson.ObjectId{bson.NewObjectId(), bson.NewObjectId(), bson.NewObjectId(), trueID}
	if !IDInSlice(trueID, slice) {
		t.Errorf("should find %v in %v", trueID, slice)
	}

	falseID := bson.NewObjectId()
	if IDInSlice(falseID, slice) {
		t.Errorf("should not find %v in %v", trueID, slice)
	}

}

func TestSliceWithoutID(t *testing.T) {

	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	a := bson.NewObjectId()
	b := bson.NewObjectId()
	c := bson.NewObjectId()
	d := bson.NewObjectId()
	z := bson.NewObjectId()
	slice := []bson.ObjectId{a, b, c, d}

	result := SliceWithoutID(slice, a)
	if !reflect.DeepEqual([]bson.ObjectId{b, c, d}, result) {
		t.Errorf("error removing a from {a, b, c, d}, got: %v", result)
	}
	result = SliceWithoutID(slice, b)
	if !reflect.DeepEqual([]bson.ObjectId{a, c, d}, result) {
		t.Errorf("error removing b from {a, b, c, d}, got: %v", result)
	}
	result = SliceWithoutID(slice, d)
	if !reflect.DeepEqual([]bson.ObjectId{a, b, c}, result) {
		t.Errorf("error removing c from {a, b, c, d}, got: %v", result)
	}
	result = SliceWithoutID(slice, z)
	if !reflect.DeepEqual([]bson.ObjectId{a, b, c, d}, result) {
		t.Errorf("error removing non-existent z from {a, b, c, d}, got: %v", result)
	}
}

func TestStringInSlice(t *testing.T) {

	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	trueID := bson.NewObjectId().Hex()

	slice := []string{trueID, bson.NewObjectId().Hex(), bson.NewObjectId().Hex(), bson.NewObjectId().Hex()}
	if !StringInSlice(trueID, slice) {
		t.Errorf("should find %v in %v", trueID, slice)
	}

	slice = []string{bson.NewObjectId().Hex(), bson.NewObjectId().Hex(), trueID, bson.NewObjectId().Hex()}
	if !StringInSlice(trueID, slice) {
		t.Errorf("should find %v in %v", trueID, slice)
	}

	slice = []string{bson.NewObjectId().Hex(), bson.NewObjectId().Hex(), bson.NewObjectId().Hex(), trueID}
	if !StringInSlice(trueID, slice) {
		t.Errorf("should find %v in %v", trueID, slice)
	}

	falseID := bson.NewObjectId().Hex()
	if StringInSlice(falseID, slice) {
		t.Errorf("should not find %v in %v", trueID, slice)
	}

}

func Test_imageHash(t *testing.T) {

	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	expectedSha256 := "sha256.d7d356079af905c04e5ae10711ecf3f5b34385e9b143c5d9ddbf740665ce2fb7"

	shasum, err := ImageHash("no_such_file.txt")
	if err == nil {
		t.Error("Invalid file must return an error")
	}

	shasum, err = ImageHash("test_data/test_sha256")
	if err != nil {
		t.Errorf("ImageHash on valid file should not raise error: %v", err)
	}
	if shasum != expectedSha256 {
		t.Errorf("ImageHash returned %v - expected %v", shasum, expectedSha256)
	}
}

func Test_sha256sum(t *testing.T) {

	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	expectedSha256 := "sha256.d7d356079af905c04e5ae10711ecf3f5b34385e9b143c5d9ddbf740665ce2fb7"

	shasum, err := sha256sum("no_such_file.txt")
	if err == nil {
		t.Error("Invalid file must return an error")
	}

	shasum, err = sha256sum("test_data/test_sha256")
	if err != nil {
		t.Errorf("sha256sum on valid file should not raise error: %v", err)
	}
	if shasum != expectedSha256 {
		t.Errorf("sha256sum returned %v - expected %v", shasum, expectedSha256)
	}
}
