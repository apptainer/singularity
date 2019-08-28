package client

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// DeleteImage deletes requested imageRef.
func (c *Client) DeleteImage(ctx context.Context, imageRef, arch string) error {
	if imageRef == "" || arch == "" {
		return errors.New("imageRef and arch are required")
	}

	path := fmt.Sprintf("/v1/images/%s", imageRef)
	apiURL, err := url.Parse(path)
	if err != nil {
		return fmt.Errorf("error constructing API url: %v", err)
	}
	q := url.Values{}
	q.Add("arch", arch)
	apiURL.RawQuery = q.Encode()
	_, err = c.doDeleteRequest(ctx, apiURL.RequestURI())
	return err
}
