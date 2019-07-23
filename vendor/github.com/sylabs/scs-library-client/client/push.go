// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-log/log"
	"github.com/hashicorp/go-retryablehttp"
	jsonresp "github.com/sylabs/json-resp"
	"golang.org/x/sync/errgroup"
)

// UploadCallback defines an interface used to perform a call-out to
// set up the source file Reader.
type UploadCallback interface {
	// Initializes the callback given a file size and source file Reader
	InitUpload(int64, io.Reader)
	// (optionally) can return a proxied Reader
	GetReader() io.Reader
	// called when the upload operation is complete
	Finish()
}

// Default upload callback
type defaultUploadCallback struct {
	r io.Reader
}

func (c *defaultUploadCallback) InitUpload(s int64, r io.Reader) {
	c.r = r
}

func (c *defaultUploadCallback) GetReader() io.Reader {
	return c.r
}

func (c *defaultUploadCallback) Finish() {
}

// calculateChecksums uses a TeeReader to calculate MD5 and SHA256
// checksums concurrently
func calculateChecksums(r io.Reader) (string, string, int64, error) {
	pr, pw := io.Pipe()
	tr := io.TeeReader(r, pw)

	var g errgroup.Group

	var md5checksum string
	var sha256checksum string
	var fileSize int64

	// compute MD5 checksum for comparison with S3 checksum
	g.Go(func() error {
		// The pipe writer must be closed so sha256 computation gets EOF and will
		// complete.
		defer pw.Close()
		var err error

		md5checksum, fileSize, err = md5sum(tr)
		if err != nil {
			return fmt.Errorf("error calculating MD5 checksum: %v", err)
		}
		return nil
	})

	// Compute sha256
	g.Go(func() error {
		var err error
		sha256checksum, _, err = sha256sum(pr)
		if err != nil {
			return fmt.Errorf("error calculating SHA checksum: %v", err)
		}
		return nil
	})

	err := g.Wait()
	return md5checksum, sha256checksum, fileSize, err
}

// UploadImage will push a specified image from an io.ReadSeeker up to the
// Container Library, The timeout value for this operation is set within
// the context. It is recommended to use a large value (ie. 1800 seconds) to
// prevent timeout when uploading large images.
func (c *Client) UploadImage(ctx context.Context, r io.ReadSeeker, path, arch string, tags []string, description string, callback UploadCallback) error {

	entityName, collectionName, containerName, parsedTags := ParseLibraryPath(path)
	if len(parsedTags) != 0 {
		return fmt.Errorf("malformed image path: %s", path)
	}

	// calculate sha256 and md5 checksums for Reader
	md5Checksum, imageHash, fileSize, err := calculateChecksums(r)
	if err != nil {
		return fmt.Errorf("error calculating checksums: %v", err)
	}

	// rollback to top of file
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("error seeking to start stream: %v", err)
	}

	c.Logger.Logf("Image hash computed as %s", imageHash)

	// Find or create entity
	entity, err := c.getEntity(ctx, entityName)
	if err != nil {
		if err != ErrNotFound {
			return err
		}
		c.Logger.Logf("Entity %s does not exist in library - creating it.", entityName)
		entity, err = c.createEntity(ctx, entityName)
		if err != nil {
			return err
		}
	}

	// Find or create collection
	qualifiedCollectionName := fmt.Sprintf("%s/%s", entityName, collectionName)
	collection, err := c.getCollection(ctx, qualifiedCollectionName)
	if err != nil {
		if err != ErrNotFound {
			return err
		}
		// create collection
		c.Logger.Logf("Collection %s does not exist in library - creating it.", collectionName)
		collection, err = c.createCollection(ctx, collectionName, entity.ID)
		if err != nil {
			return err
		}
	}

	// Find or create container
	computedName := fmt.Sprintf("%s/%s", qualifiedCollectionName, containerName)
	container, err := c.getContainer(ctx, computedName)
	if err != nil {
		if err != ErrNotFound {
			return err
		}
		// Create container
		c.Logger.Logf("Container %s does not exist in library - creating it.", containerName)
		container, err = c.createContainer(ctx, containerName, collection.ID)
		if err != nil {
			return err
		}
	}

	// Find or create image
	image, err := c.GetImage(ctx, arch, computedName+":"+imageHash)
	if err != nil {
		if err != ErrNotFound {
			return err
		}
		// Create image
		c.Logger.Logf("Image %s does not exist in library - creating it.", imageHash)
		image, err = c.createImage(ctx, imageHash, container.ID, description)
		if err != nil {
			return err
		}
	}

	if !image.Uploaded {
		c.Logger.Log("Now uploading to the library")
		if c.apiAtLeast(ctx, APIVersionV2Upload) {
			// use v2 post file api
			metadata := map[string]string{
				"md5sum": md5Checksum,
			}
			if err := c.postFileV2(ctx, r, fileSize, image.ID, callback, metadata); err != nil {
				return err
			}
		} else if err := c.postFile(ctx, r, fileSize, image.ID, callback); err != nil {
			return err
		}
		c.Logger.Logf("Upload completed OK")
	} else {
		c.Logger.Logf("Image is already present in the library - not uploading.")
	}

	c.Logger.Logf("Setting tags against uploaded image")

	if c.apiAtLeast(ctx, APIVersionV2ArchTags) {
		return c.setTagsV2(ctx, container.ID, arch, image.ID, append(tags, parsedTags...))
	}
	c.Logger.Logf("This library does not support multiple architecture per tag.")
	c.Logger.Logf("This tag will replace any already uploaded with the same name.")
	return c.setTags(ctx, container.ID, image.ID, append(tags, parsedTags...))
}

func (c *Client) postFile(ctx context.Context, r io.Reader, fileSize int64, imageID string, callback UploadCallback) error {

	postURL := "/v1/imagefile/" + imageID
	c.Logger.Logf("postFile calling %s", postURL)

	if callback == nil {
		// fallback to default upload callback
		callback = &defaultUploadCallback{}
	}

	// use callback to set up source file reader
	callback.InitUpload(fileSize, r)
	defer callback.Finish()

	// Make an upload request
	req, _ := c.newRequest(http.MethodPost, postURL, "", callback.GetReader())
	// Content length is required by the API
	req.ContentLength = fileSize
	res, err := c.HTTPClient.Do(req.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("error uploading file to server: %s", err.Error())
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		err := jsonresp.ReadError(res.Body)
		if err != nil {
			return fmt.Errorf("sending file did not succeed: %v", err)
		}
		return fmt.Errorf("sending file did not succeed: http status code %d", res.StatusCode)
	}
	return nil
}

// loggingAdapter is an adapter to redirect log messages from retryablehttp
// to our logger
type loggingAdapter struct {
	logger log.Logger
}

// Printf implements interface used by retryablehttp
func (l *loggingAdapter) Printf(fmt string, args ...interface{}) {
	l.logger.Logf(fmt, args)
}

// postFileV2 uses V2 API to upload images to SCS library server. This is
// a three step operation: "create" upload image request, which returns a
// URL to issue an http PUT operation against, and then finally calls the
// completion endpoint once upload is complete.
func (c *Client) postFileV2(ctx context.Context, r io.Reader, fileSize int64, imageID string, callback UploadCallback, metadata map[string]string) error {

	if callback == nil {
		// fallback to default upload callback
		callback = &defaultUploadCallback{}
	}

	postURL := "/v2/imagefile/" + imageID
	c.Logger.Logf("postFileV2 calling %s", postURL)

	// issue upload request (POST) to obtain presigned S3 URL
	body := UploadImageRequest{
		Size:        fileSize,
		MD5Checksum: metadata["md5sum"],
	}

	objJSON, err := c.apiCreate(ctx, postURL, body)
	if err != nil {
		return err
	}

	var res UploadImageResponse
	if err := json.Unmarshal(objJSON, &res); err != nil {
		return nil
	}

	// set up source file reader
	callback.InitUpload(fileSize, r)

	// upload (PUT) directly to S3 presigned URL provided above
	presignedURL := res.Data.UploadURL
	if presignedURL == "" {
		return fmt.Errorf("error getting presigned URL")
	}

	req, err := retryablehttp.NewRequest(http.MethodPut, presignedURL, callback.GetReader())
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.ContentLength = fileSize
	req.Header.Set("Content-Type", "application/octet-stream")

	// redirect log output from retryablehttp to our logger
	l := loggingAdapter{
		logger: c.Logger,
	}

	client := retryablehttp.NewClient()
	client.Logger = &l

	resp, err := client.Do(req.WithContext(ctx))
	callback.Finish()
	if err != nil {
		return fmt.Errorf("error uploading image: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error uploading image: HTTP status %d", resp.StatusCode)
	}

	// send (PUT) image upload completion
	_, err = c.apiUpdate(ctx, postURL+"/_complete", UploadImageCompleteRequest{})
	return err
}
