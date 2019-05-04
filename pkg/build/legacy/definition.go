// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	buildclient "github.com/sylabs/scs-build-client/client"
)

// NewDefinitionFromURI crafts a new Definition given a URI
func NewDefinitionFromURI(uri string) (d buildclient.Definition, err error) {
	var u []string
	if strings.Contains(uri, "://") {
		u = strings.SplitN(uri, "://", 2)
	} else if strings.Contains(uri, ":") {
		u = strings.SplitN(uri, ":", 2)
	} else {
		return d, fmt.Errorf("build URI must start with prefix:// or prefix: ")
	}

	d = buildclient.Definition{
		Header: map[string]string{
			"bootstrap": u[0],
			"from":      u[1],
		},
	}

	var buf bytes.Buffer
	populateRaw(&d, &buf)
	d.Raw = buf.Bytes()

	return d, nil
}

// NewDefinitionFromJSON creates a new Definition using the supplied JSON.
func NewDefinitionFromJSON(r io.Reader) (d buildclient.Definition, err error) {
	decoder := json.NewDecoder(r)

	for {
		if err = decoder.Decode(&d); err == io.EOF {
			break
		} else if err != nil {
			return
		}
	}

	// if JSON definition doesn't have a raw data section, add it
	if len(d.Raw) == 0 {
		var buf bytes.Buffer
		populateRaw(&d, &buf)
		d.Raw = buf.Bytes()
	}

	return d, nil
}

func writeSectionIfExists(w io.Writer, ident string, s string) {
	if len(s) > 0 {
		w.Write([]byte("%"))
		w.Write([]byte(ident))
		w.Write([]byte("\n"))
		w.Write([]byte(s))
		w.Write([]byte("\n\n"))
	}
}

func writeFilesIfExists(w io.Writer, f []buildclient.FileTransport) {

	if len(f) > 0 {

		w.Write([]byte("%"))
		w.Write([]byte("files"))
		w.Write([]byte("\n"))

		for _, ft := range f {
			w.Write([]byte("\t"))
			w.Write([]byte(ft.Src))
			w.Write([]byte("\t"))
			w.Write([]byte(ft.Dst))
			w.Write([]byte("\n"))
		}
		w.Write([]byte("\n"))
	}
}

func writeLabelsIfExists(w io.Writer, l map[string]string) {

	if len(l) > 0 {

		w.Write([]byte("%"))
		w.Write([]byte("labels"))
		w.Write([]byte("\n"))

		for k, v := range l {
			w.Write([]byte("\t"))
			w.Write([]byte(k))
			w.Write([]byte(" "))
			w.Write([]byte(v))
			w.Write([]byte("\n"))
		}
		w.Write([]byte("\n"))
	}
}

// populateRaw is a helper func to output a Definition struct
// into a definition file.
func populateRaw(d *buildclient.Definition, w io.Writer) {
	for k, v := range d.Header {
		w.Write([]byte(k))
		w.Write([]byte(": "))
		w.Write([]byte(v))
		w.Write([]byte("\n"))
	}
	w.Write([]byte("\n"))

	writeLabelsIfExists(w, d.ImageData.Labels)
	writeFilesIfExists(w, d.BuildData.Files)

	writeSectionIfExists(w, "help", d.ImageData.Help)
	writeSectionIfExists(w, "environment", d.ImageData.Environment)
	writeSectionIfExists(w, "runscript", d.ImageData.Runscript)
	writeSectionIfExists(w, "test", d.ImageData.Test)
	writeSectionIfExists(w, "startscript", d.ImageData.Startscript)
	writeSectionIfExists(w, "pre", d.BuildData.Pre)
	writeSectionIfExists(w, "setup", d.BuildData.Setup)
	writeSectionIfExists(w, "post", d.BuildData.Post)
}
