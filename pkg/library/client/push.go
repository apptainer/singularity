package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"

	"gopkg.in/cheggaaa/pb.v1"
	"gopkg.in/mgo.v2/bson"
)

var baseURL string

func UploadImage(filePath string, libraryRef string, libraryURL string) error {

	baseURL = libraryURL

	if !isLibraryRef(libraryRef) {
		log.Fatalf("Not a valid library reference: %s", libraryRef)
	}

	entity, collection, container, image := parseLibraryRef(libraryRef)

	entityID, err := entityExists(entity)
	if err != nil {
		return err
	}
	if entityID == "" {
		log.Printf("Entity %s does not exist in library - creating it.\n", entity)
		entityID, err = createEntity(entity)
		if err != nil {
			return err
		}
	}
	collectionID, err := collectionExists(entity, collection)
	if err != nil {
		return err
	}
	if collectionID == "" {
		log.Printf("Collection %s/%s does not exist in library - creating it.\n", entity, collection)
		collectionID, err = createCollection(collection, entityID)
		if err != nil {
			return err
		}
	}
	containerID, err := containerExists(entity, collection, container)
	if err != nil {
		return err
	}
	if containerID == "" {
		log.Printf("Container %s/%s/%s does not exist in library - creating it.\n", entity, collection, container)
		containerID, err = createContainer(container, collectionID)
		if err != nil {
			return err
		}
	}
	imageID, err := imageExists(entity, collection, container, image)
	if err != nil {
		return err
	}
	if imageID == "" {
		log.Printf("Image %s/%s/%s:%s does not exist in library - creating it.\n", entity, collection, container, image)
		imageID, err = createImage(image, containerID)
		if err != nil {
			return err
		}
	} else {
		log.Println("This image already exists in the library - it will be overwritten.")
	}
	log.Printf("Now uploading %s to the library\n", filePath)

	err = postFile(filePath, imageID)
	if err != nil {
		return err
	}

	log.Printf("Upload Complete!")
	return nil
}

func entityExists(entity string) (id string, err error) {
	url := (baseURL + "/v1/entities/" + entity)
	return apiExists(url)
}

func collectionExists(entity string, collection string) (id string, err error) {
	url := baseURL + "/v1/collections/" + entity + "/" + collection
	return apiExists(url)
}

func containerExists(entity string, collection string, container string) (id string, err error) {
	url := baseURL + "/v1/containers/" + entity + "/" + collection + "/" + container
	return apiExists(url)
}

func imageExists(entity string, collection string, container string, image string) (id string, err error) {
	url := baseURL + "/v1/images/" + entity + "/" + collection + "/" + container + "/" + image
	return apiExists(url)
}

func createEntity(name string) (id string, err error) {
	e := Entity{
		Name:        name,
		Description: "No description",
	}
	return apiCreate(e, baseURL+"/v1/entities")

}

func createCollection(name string, entityID string) (id string, err error) {
	c := Collection{
		Name:        name,
		Description: "No description",
		Entity:      bson.ObjectIdHex(entityID),
	}
	return apiCreate(c, baseURL+"/v1/collections")
}

func createContainer(name string, collectionID string) (id string, err error) {
	c := Container{
		Name:        name,
		Description: "No description",
		Collection:  bson.ObjectIdHex(collectionID),
	}
	return apiCreate(c, baseURL+"/v1/containers")
}

func createImage(name string, containerID string) (id string, err error) {
	i := Image{
		Name:        name,
		Description: "No description",
		Container:   bson.ObjectIdHex(containerID),
	}

	return apiCreate(i, baseURL+"/v1/images")
}

func apiCreate(o interface{}, url string) (id string, err error) {
	s, err := json.Marshal(o)
	if err != nil {
		return "", fmt.Errorf("Error encoding object to JSON:\n\t%v", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(s))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Error making request to server:\n\t%v", err)
	}
	if res.StatusCode != http.StatusOK {
		jRes, err := ParseErrorBody(res.Body)
		if err != nil {
			jRes = ParseErrorResponse(res)
		}
		return "", fmt.Errorf("Creation did not succeed: %d %s\n\t%v",
			jRes.Error.Code, jRes.Error.Status, jRes.Error.Message)
	}

	// Decode the returned created object to find its ID
	c := make(map[string]map[string]interface{})
	err = json.NewDecoder(res.Body).Decode(&c)
	if err != nil {
		return "", fmt.Errorf("Error decoding ID from server response:\n\t%v", err)
	}

	return c["data"]["id"].(string), nil

}

func apiExists(url string) (id string, err error) {
	res, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("Error making request to server:\n\t%v", err)
	}
	if res.StatusCode == http.StatusOK {
		c := make(map[string]map[string]interface{})
		json.NewDecoder(res.Body).Decode(&c)
		if err != nil {
			return "", fmt.Errorf("Error decoding ID from server response:\n\t%v", err)
		}
		return c["data"]["id"].(string), nil
	}
	return "", nil
}

func postFile(filePath string, imageID string) error {

	var b bytes.Buffer

	w := multipart.NewWriter(&b)
	f, err := os.Open(filePath)
	if err != nil {
		fmt.Errorf("Could not open the image file to upload: %v", err)
		return err
	}
	fi, _ := f.Stat()
	fileSize := fi.Size()

	defer f.Close()

	fw, err := w.CreateFormFile("imagefile", filePath)
	if err != nil {
		fmt.Errorf("Could not prepare the image file upload: %v", err)
		return err
	}
	if _, err = io.Copy(fw, f); err != nil {
		fmt.Errorf("Could not prepare the image file upload: %v", err)
		return err
	}

	w.Close()

	// create and start bar
	bar := pb.New(int(fileSize)).SetUnits(pb.U_BYTES)
	bar.ShowTimeLeft = true
	bar.ShowSpeed = true
	bar.Start()
	// create proxy reader
	bodyProgress := bar.NewProxyReader(&b)
	// Make an upload request
	req, _ := http.NewRequest("POST", baseURL+"/v1/imagefile/"+imageID, bodyProgress)
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())
	client := &http.Client{}
	res, err := client.Do(req)

	bar.Finish()

	if err != nil {
		return fmt.Errorf("Error uploading file to server: %s", err.Error())
	}
	if res.StatusCode != http.StatusOK {
		jRes, err := ParseErrorBody(res.Body)
		if err != nil {
			jRes = ParseErrorResponse(res)
		}
		return fmt.Errorf("Sending file did not succeed: %d %s\n\t%v",
			jRes.Error.Code, jRes.Error.Status, jRes.Error.Message)
	}

	return nil

}
