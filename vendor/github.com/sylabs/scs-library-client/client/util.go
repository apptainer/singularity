// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"regexp"
	"strings"
)

// IsLibraryPullRef returns true if the provided string is a valid library
// reference for a pull operation.
func IsLibraryPullRef(libraryRef string) bool {
	match, _ := regexp.MatchString("^(library://)?([a-z0-9]+(?:[._-][a-z0-9]+)*/){0,2}([a-z0-9]+(?:[._-][a-z0-9]+)*)(:[a-z0-9]+(?:[._-][a-z0-9]+)*)?$", libraryRef)
	return match
}

// IsLibraryPushRef returns true if the provided string is a valid library
// reference for a push operation.
func IsLibraryPushRef(libraryRef string) bool {
	// For push we allow specifying multiple tags, delimited with ,
	match, _ := regexp.MatchString("^(library://)?([a-z0-9]+(?:[._-][a-z0-9]+)*/){2}([a-z0-9]+(?:[._-][a-z0-9]+)*)(:[a-z0-9]+(?:[,._-][a-z0-9]+)*)?$", libraryRef)
	return match
}

// IsRefPart returns true if the provided string is valid as a component of a
// library URI (i.e. a valid entity, collection etc. name)
func IsRefPart(refPart string) bool {
	match, err := regexp.MatchString("^[a-z0-9]+(?:[._-][a-z0-9]+)*$", refPart)
	if err != nil {
		return false
	}
	return match
}

// IsImageHash returns true if the provided string is valid as a unique hash
// for an image
func IsImageHash(refPart string) bool {
	// Legacy images will be sent with hash sha256.[a-f0-9]{64}
	// SIF images will be sent with hash sif.[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}
	//  which is the unique SIF UUID
	match, err := regexp.MatchString("^((sha256\\.[a-f0-9]{64})|(sif\\.[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}))$", refPart)
	if err != nil {
		return false
	}
	return match
}

func ParseLibraryPath(libraryRef string) (entity string, collection string, container string, tags []string) {

	libraryRef = strings.TrimPrefix(libraryRef, "library://")

	refParts := strings.Split(libraryRef, "/")

	switch len(refParts) {
	case 3:
		entity = refParts[0]
		collection = refParts[1]
		container = refParts[2]
	case 2:
		entity = ""
		collection = refParts[0]
		container = refParts[1]
	case 1:
		entity = ""
		collection = ""
		container = refParts[0]
	}

	if strings.Contains(container, ":") {
		imageParts := strings.Split(container, ":")
		container = imageParts[0]
		tags = []string{imageParts[1]}
		if strings.Contains(tags[0], ",") {
			tags = strings.Split(tags[0], ",")
		}
	}

	return entity, collection, container, tags
}

// IDInSlice returns true if ID is present in the slice
func IDInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// SliceWithoutID returns slice with specified ID removed
func SliceWithoutID(list []string, a string) []string {

	var newList []string

	for _, b := range list {
		if b != a {
			newList = append(newList, b)
		}
	}
	return newList
}

// StringInSlice returns true if string is present in the slice
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// PrettyPrint - Debug helper, print nice json for any interface
func PrettyPrint(v interface{}) {
	b, _ := json.MarshalIndent(v, "", "  ")
	println(string(b))
}

// ImageHash returns the appropriate hash for a provided image file
//   e.g. sif.<uuid> or sha256.<sha256>
func ImageHash(filePath string) (result string, err error) {
	// Currently using sha256 always
	// TODO - use sif uuid for sif files!
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	result, _, err = sha256sum(file)

	return result, err
}

// sha256sum computes the sha256sum of the specified reader; caller is
// responsible for resetting file pointer. 'nBytes' indicates number of
// bytes read from reader
func sha256sum(r io.Reader) (result string, nBytes int64, err error) {
	hash := sha256.New()
	nBytes, err = io.Copy(hash, r)
	if err != nil {
		return "", 0, err
	}

	return "sha256." + hex.EncodeToString(hash.Sum(nil)), nBytes, nil
}

// md5sum computes the MD5 checksum of the specified reader; caller is
// responsible for resetting file pointer. nBytes' indicates number of
// bytes read from reader
func md5sum(r io.Reader) (result string, nBytes int64, err error) {
	hash := md5.New()
	nBytes, err = io.Copy(hash, r)
	if err != nil {
		return "", 0, err
	}

	return hex.EncodeToString(hash.Sum(nil)), nBytes, nil
}
