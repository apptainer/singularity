package client

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"bytes"
	"fmt"
	"net/http"

	"github.com/globalsign/mgo/bson"
	"github.com/golang/glog"
)

func isLibraryPullRef(libraryRef string) bool {
	match, _ := regexp.MatchString("^(library://)?([a-z0-9]+(?:[._-][a-z0-9]+)*/){2}([a-z0-9]+(?:[._-][a-z0-9]+)*)(:[a-z0-9]+(?:[._-][a-z0-9]+)*)?$", libraryRef)
	return match
}

func isLibraryPushRef(libraryRef string) bool {
	// For push we allow specifying multiple tags, delimited with ,
	match, _ := regexp.MatchString("^(library://)?([a-z0-9]+(?:[._-][a-z0-9]+)*/){2}([a-z0-9]+(?:[._-][a-z0-9]+)*)(:[a-z0-9]+(?:[,._-][a-z0-9]+)*)?$", libraryRef)
	return match
}

// IsRefPart returns true if the provided string is valid as a component of a
// library URI (i.e. a valid entity, collection etc. name)
func IsRefPart(refPart string) bool {
	match, err := regexp.MatchString("^[a-z0-9]+(?:[._-][a-z0-9]+)*$", refPart)
	if err != nil {
		glog.Errorf("Error in regex matching: %v", err)
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
	match, err := regexp.MatchString("^(sha256\\.[a-f0-9]{64})|(sif\\.[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})", refPart)
	if err != nil {
		glog.Errorf("Error in regex matching: %v", err)
		return false
	}
	return match
}

// IsTag returns true if the provided string is valid as a tag
func IsTag(tag string) bool {
	match, err := regexp.MatchString("^[a-z0-9]+(?:[._-][a-z0-9]+)*$", tag)
	if err != nil {
		glog.Errorf("Error in regex matching: %v", err)
		return false
	}
	return match
}

func parseLibraryRef(libraryRef string) (entity string, collection string, container string, tags []string) {

	libraryRef = strings.TrimPrefix(libraryRef, "library://")

	refParts := strings.Split(libraryRef, "/")

	entity = refParts[0]
	collection = refParts[1]
	container = refParts[2]

	// Default tag is latest
	tags = []string{"latest"}

	if strings.Contains(container, ":") {
		imageParts := strings.Split(container, ":")
		container = imageParts[0]
		tags = []string{imageParts[1]}
		if strings.Contains(tags[0], ",") {
			tags = strings.Split(tags[0], ",")
		}
	}

	return

}

// ParseErrorBody - Parse an API format rror out of the body
func ParseErrorBody(r io.Reader) (jRes JSONResponse, err error) {
	err = json.NewDecoder(r).Decode(&jRes)
	if err != nil {
		return jRes, fmt.Errorf("The server returned a response that could not be decoded: %v", err)
	}
	return jRes, nil
}

// ParseErrorResponse - Create a JSONResponse out of a raw HTTP response
func ParseErrorResponse(res *http.Response) (jRes JSONResponse) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	s := buf.String()
	jRes.Error.Code = res.StatusCode
	jRes.Error.Status = http.StatusText(res.StatusCode)
	jRes.Error.Message = s
	return jRes
}

// IDInSlice returns true if ID is present in the slice
func IDInSlice(a bson.ObjectId, list []bson.ObjectId) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// SliceWithoutID returns slice with specified ID removed
func SliceWithoutID(list []bson.ObjectId, a bson.ObjectId) []bson.ObjectId {

	var newList []bson.ObjectId

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

// BsonUTCNow returns a time.Time in UTC, with the precision supported by BSON
func BsonUTCNow() time.Time {
	return bson.Now().UTC()
}

// ImageHash returns the appropriate hash for a provided image file
//   e.g. sif.<uuid> or sha256.<sha256>
func ImageHash(filePath string) (result string, err error) {
	// Currently using sha256 always
	// TODO - use sif uuid for sif files!
	return sha256sum(filePath)
}

// SHA256Sum computes the sha256sum of a file
func sha256sum(filePath string) (result string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	return "sha256." + hex.EncodeToString(hash.Sum(nil)), nil
}
