package client

import (
	"encoding/json"
	"io"
	"regexp"
	"strings"

	"github.com/golang/glog"
)

// JSONError - Struct for standard error returns over REST API
type JSONError struct {
	Code    int    `json:"code,omitempty"`
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

// JSONResponse - Top level container of a REST API response
type JSONResponse struct {
	Data  interface{} `json:"data"`
	Error JSONError   `json:"error,omitempty"`
}

func isLibraryRef(libraryRef string) bool {
	match, _ := regexp.MatchString("^(library://)?([a-z0-9]+(?:[._-][a-z0-9]+)*/){2}([a-z0-9]+(?:[._-][a-z0-9]+)*)(:[a-z0-9]+(?:[._-][a-z0-9]+)*)?$", libraryRef)
	return match
}

func parseLibraryRef(libraryRef string) (entity string, collection string, container string, image string) {

	libraryRef = strings.TrimPrefix(libraryRef, "library://")

	refParts := strings.Split(libraryRef, "/")

	entity = refParts[0]
	collection = refParts[1]
	container = refParts[2]
	image = "latest"

	if strings.Contains(container, ":") {
		imageParts := strings.Split(container, ":")
		container = imageParts[0]
		image = imageParts[1]
	}

	return

}

func ParseBody(r io.Reader) (jRes JSONResponse) {
	err := json.NewDecoder(r).Decode(&jRes)
	if err != nil {
		glog.Fatalf("The server returned a response that could not be decoded: %v", err)
	}
	return jRes
}
