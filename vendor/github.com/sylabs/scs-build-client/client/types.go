// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"time"
)

// BuildRequest contains the info necessary for submitting a build to build service
type BuildRequest struct {
	Definition  `json:"definition"`
	LibraryRef  string `json:"libraryRef"`
	LibraryURL  string `json:"libraryURL"`
	CallbackURL string `json:"callbackURL"`
}

// BuildInfo contains the details of an individual build
type BuildInfo struct {
	ID            string     `json:"id"`
	SubmitTime    time.Time  `json:"submitTime"`
	StartTime     *time.Time `json:"startTime,omitempty"`
	IsComplete    bool       `json:"isComplete"`
	CompleteTime  *time.Time `json:"completeTime,omitempty"`
	ImageSize     int64      `json:"imageSize,omitempty"`
	ImageChecksum string     `json:"imageChecksum,omitempty"`
	Definition    Definition `json:"definition"`
	LibraryRef    string     `json:"libraryRef"`
	LibraryURL    string     `json:"libraryURL"`
	CallbackURL   string     `json:"callbackURL"`
}
