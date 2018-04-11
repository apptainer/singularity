/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.
  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package build

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type ReqTag string

const (
	build  ReqTag = "builderinit"
	status ReqTag = "statusrequest"
	pull   ReqTag = "imagepull"
)

// RemoteBuilder contains the build request and response
type RemoteBuilder struct {
	ImagePath string
	Client    *http.Client
	buildURL  url.URL
	request
	response
}

type request struct {
	RequestData RequestData
}

// RequestData contains the info necessary for submitting a build to a remote service
type RequestData struct {
	Definition `json:"definition"`
	IsDetached bool `json:"isDetached"`
}

type response struct {
	ResponseData ResponseData
	Responses    map[ReqTag]*http.Response
}

// ResponseData contains the details of an individual build
type ResponseData struct {
	ID           uuid.UUID  `json:"id"`
	SubmitTime   time.Time  `json:"submitTime"`
	IsComplete   bool       `json:"isComplete"`
	CompleteTime *time.Time `json:"completeTime,omitempty"`
	IsDetached   bool       `json:"isDetached"`
	WSURL        string     `json:"wsURL,omitempty"`
	ImageURL     string     `json:"imageURL,omitempty"`
	ImagePath    string     `json:"-"`
	Definition   Definition `json:"definition"`
}

// NewRemoteBuilder initializes the RemoteBuilder struct
func NewRemoteBuilder(p string, d Definition, isDetached bool, addr string) (b *RemoteBuilder) {
	b = &RemoteBuilder{
		ImagePath: p,
		Client:    &http.Client{},
		buildURL: url.URL{
			Scheme: "http",
			Host:   addr,
			Path:   "build",
		},
		request: request{
			RequestData: RequestData{
				Definition: d,
				IsDetached: isDetached,
			},
		},

		response: response{
			ResponseData: ResponseData{},
			Responses:    make(map[ReqTag]*http.Response),
		},
	}

	return
}

// Build is responsible for making the request via the REST API to the remote builder
func (b *RemoteBuilder) Build() error {
	b.doBuildRequest()

	// Update buildURL to include UUID for status requests
	b.buildURL.Path = "build/" + b.ResponseData.ID.String()

	// Dial websocket
	c, _, err := websocket.DefaultDialer.Dial(b.ResponseData.WSURL, nil)
	if err != nil {
		glog.Fatal(err)
	}

	// Output runtime
	done := make(chan struct{})
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Stream output
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				glog.Infoln("read:", err)
				return
			}
			fmt.Printf("%s\n", message)
		}
	}()

	// Wait for completion or SIGTERM
	for {
		select {
		case <-interrupt:
			glog.Infoln("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				glog.Infoln("write close:", err)
				return err
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return err
		case _ = <-done:
			glog.Infoln("Woohoo Your build complete! ")
			b.doStatusRequest()
			b.doPullRequest()
			return err
		}

	}
	return nil
}

func (b *RemoteBuilder) doBuildRequest() {
	// Marshal RequestData into JSON format for Build Request
	reqData, err := json.Marshal(b.RequestData)
	if err != nil {
		panic(err)
	}

	// Create Build Request
	req, err := http.NewRequest("POST", b.buildURL.String(), bytes.NewBuffer(reqData))
	if err != nil {
		panic(err)
	}

	// Do Build Request
	b.Responses[build], err = b.Client.Do(req)
	if err != nil {
		panic(err)
	}

	// Parse Build Response
	json.NewDecoder(b.Responses[build].Body).Decode(&b.ResponseData)
}

func (b *RemoteBuilder) doStatusRequest() {
	// Create Status Request
	req, err := http.NewRequest("GET", b.buildURL.String(), nil)
	if err != nil {
		panic(err)
	}

	// Do Status Request
	b.Responses[status], err = b.Client.Do(req)
	if err != nil {
		panic(err)
	}

	// Parse Status Response
	json.NewDecoder(b.Responses[status].Body).Decode(&b.ResponseData)

}

func (b *RemoteBuilder) doPullRequest() {
	// Create Image Request
	req, err := http.NewRequest("GET", b.ResponseData.ImageURL, nil)
	if err != nil {
		panic(err)
	}

	// Do Image Request
	b.Responses[pull], err = b.Client.Do(req)
	if err != nil {
		panic(err)
	}

	glog.Infof("Pulling image from %v to %v...", b.ResponseData.ImageURL, b.ImagePath)

	// Save image file to disk
	imageFile, err := os.OpenFile(b.ImagePath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		glog.Fatal(err)
	}
	io.Copy(imageFile, b.Responses[pull].Body)
	imageFile.Close()

	glog.Infof("done!\n")
}

/* ==================================================================================== */

// DefFileRequest is used by Singularity 2.x Python CLI to reqeuest a parsed Deffile
type DefFileRequest struct {
	RawDefFile string `json:"rawDefFile"`
}
