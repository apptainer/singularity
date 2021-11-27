// Copyright (c) 2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"encoding/csv"
	"fmt"
	"strings"
)

// ParseMountString converts a --mount string into one or more BindPath structs.
//
// Our intention is to support common docker --mount strings, but have
// additional fields for singularity specific concepts (image-src, id when
// binding out of an image file).
//
// We use a CSV reader to parse the fields in a mount string according to CSV
// escaping rules. This is the approach docker uses to allow special characters
// in source / dest etc., and we wish to be as compatible as possible. It also
// allows us to handle multiple newline separated mounts, which is convenient
// for specifying multiple mounts in a single env var.
//
// The fields are in key[=value] format. Flag options have no value, e.g.:
//   type=bind,source=/opt,destination=/other,rw
//
// We only support type=bind at present, so assume this if type is missing and
// error for other types.
func ParseMountString(mount string) (bindPaths []BindPath, err error) {
	r := strings.NewReader(mount)
	c := csv.NewReader(r)
	records, err := c.ReadAll()
	if err != nil {
		return []BindPath{}, fmt.Errorf("error parsing mount: %v", err)
	}

	for _, r := range records {
		bp := BindPath{
			Options: map[string]*BindOption{},
		}

		for _, f := range r {
			kv := strings.SplitN(f, "=", 2)
			key := kv[0]
			val := ""
			if len(kv) > 1 {
				val = kv[1]
			}

			switch key {
			// TODO - Eventually support volume and tmpfs? Requires structural changes to engine mount functionality.
			case "type":
				if val != "bind" {
					return []BindPath{}, fmt.Errorf("unsupported mount type %q, only 'bind' is supported", val)
				}
			case "source", "src":
				if val == "" {
					return []BindPath{}, fmt.Errorf("mount source cannot be empty")
				}
				bp.Source = val
			case "destination", "dst", "target":
				if val == "" {
					return []BindPath{}, fmt.Errorf("mount destination cannot be empty")
				}
				bp.Destination = val
			case "ro", "readonly":
				bp.Options["ro"] = &BindOption{}
			// Singularity only - directory inside an image file source to mount from
			case "image-src":
				if val == "" {
					return []BindPath{}, fmt.Errorf("img-src cannot be empty")
				}
				bp.Options["image-src"] = &BindOption{Value: val}
			// Singularity only - id of the descriptor in a SIF image source to mount from
			case "id":
				if val == "" {
					return []BindPath{}, fmt.Errorf("id cannot be empty")
				}
				bp.Options["id"] = &BindOption{Value: val}
			case "bind-propagation":
				return []BindPath{}, fmt.Errorf("bind-propagation not supported for individual mounts, check singularity.conf for global setting")
			default:
				return []BindPath{}, fmt.Errorf("invalid key %q in mount specification", key)
			}
		}

		if bp.Source == "" || bp.Destination == "" {
			return []BindPath{}, fmt.Errorf("mounts must specify a source and a destination")
		}
		bindPaths = append(bindPaths, bp)
	}

	return bindPaths, nil
}
