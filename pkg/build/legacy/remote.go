// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package types

import (
	"time"

	"github.com/globalsign/mgo/bson"
)

// RequestData contains the info necessary for submitting a build to a remote service
type RequestData struct {
	Definition  `json:"definition"`
	LibraryRef  string `json:"libraryRef"`
	LibraryURL  string `json:"libraryURL"`
	CallbackURL string `json:"callbackURL"`
}

// ResponseData contains the details of an individual build
type ResponseData struct {
	ID            bson.ObjectId `json:"id"`
	CreatedBy     string        `json:"createdBy"`
	SubmitTime    time.Time     `json:"submitTime"`
	StartTime     *time.Time    `json:"startTime,omitempty" bson:",omitempty"`
	IsComplete    bool          `json:"isComplete"`
	CompleteTime  *time.Time    `json:"completeTime,omitempty"`
	ImageSize     int64         `json:"imageSize,omitempty"`
	ImageChecksum string        `json:"imageChecksum,omitempty"`
	Definition    Definition    `json:"definition"`
	WSURL         string        `json:"wsURL,omitempty" bson:"-"`
	LibraryRef    string        `json:"libraryRef"`
	LibraryURL    string        `json:"libraryURL"`
	CallbackURL   string        `json:"callbackURL"`
}
