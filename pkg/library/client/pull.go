package client

import (
	"log"
	"os"
	"net/http"
	"io"
	"github.com/cheggaaa/pb"
	"strings"
	"fmt"
)

func DownloadImage(filePath string, libraryRef string, libraryURL string) error{

	if ! isLibraryRef(libraryRef) {
		log.Fatalf("Not a valid library URI: %s", libraryRef)
	}

	url := libraryURL + "/v1/imagefile/" + strings.TrimPrefix(libraryRef, "library://")

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return fmt.Errorf("Requested image was not found in the library")
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Unexpected response from the library server: %d", res.StatusCode)
	}

	bodySize := res.ContentLength
	bar := pb.New(int(bodySize)).SetUnits(pb.U_BYTES)
	bar.Start()

	// create proxy reader
	bodyProgress := bar.NewProxyReader(res.Body)

	// Write the body to file
	_, err = io.Copy(out, bodyProgress)
	if err != nil {
		return err
	}

	bar.Finish()

	log.Printf("Download Complete!")

	return nil



}
