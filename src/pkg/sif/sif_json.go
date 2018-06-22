// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sif

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

// JSONMarshaler is to ensure (for the time being) that a JSON object in a SIF file
// has a Name so that a user can
type JSONMarshaler struct {
	Name    string `json:"name"`
	Content []byte `json:"content"`
}

// NewJSONMarshaler creates an object for cleanly storing JSON in SIF files
func NewJSONMarshaler(name string, content interface{}) (*JSONMarshaler, error) {
	b, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}

	j := &JSONMarshaler{
		Name:    name,
		Content: b,
	}

	return j, nil
}

// ToFile writes the contents of the JSONMarshaler into the file
func (j *JSONMarshaler) ToFile(path string, perm os.FileMode) error {
	b, err := json.Marshal(j)
	if err != nil {

	}

	err = ioutil.WriteFile(path, b, perm)
	if err != nil {
		return err
	}

	return nil
}
