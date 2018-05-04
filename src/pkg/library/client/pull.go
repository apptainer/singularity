/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package client

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"log"

	"gopkg.in/cheggaaa/pb.v1"
)

func DownloadImage(filePath string, libraryRef string, libraryURL string) error {

	if !isLibraryRef(libraryRef) {
		log.Fatalf("Not a valid library reference: %s", libraryRef)
	}

	url := libraryURL + "/v1/imagefile/" + strings.TrimPrefix(libraryRef, "library://")

	fmt.Println(url)

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
		return fmt.Errorf("The requested image was not found in the library")
	}

	if res.StatusCode != http.StatusOK {
		jRes, err := ParseErrorBody(res.Body)
		if err != nil {
			jRes = ParseErrorResponse(res)
		}
		return fmt.Errorf("Download did not succeed: %d %s\n\t%v",
			jRes.Error.Code, jRes.Error.Status, jRes.Error.Message)
	}

	bodySize := res.ContentLength
	bar := pb.New(int(bodySize)).SetUnits(pb.U_BYTES)
	bar.ShowTimeLeft = true
	bar.ShowSpeed = true
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
