package client

import (
	"regexp"
	"strings"
)

func isLibraryRef(libraryRef string) bool {
	// TODO - THIS ISN'T A PERFECT MATCHER YET
	match, _ := regexp.MatchString("(library://)?[a-zA-Z0-9-]+/[a-zA-Z0-9-]+/[a-zA-Z0-9-]+(:[a-zA-Z0-9-]*)?", libraryRef)
	return match
}

func parseLibraryRef(libraryRef string) (entity string, collection string, container string, image string) {

	libraryRef = strings.TrimLeft(libraryRef, "library://")

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
