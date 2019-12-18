// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package plugin

// Manifest is the plugin manifest, stored as a data object in the plugin SIF.
type Manifest struct {
	// Name is, by convention, a fully-qualified domain name which uniquely identifies a plugin.
	// This convention is not enforced, but rather is best practice.
	//
	// Good Names:
	//     - sylabs.io/test-plugin
	//     - github.com/user/repo
	//
	// Bad Names:
	//     - test-plugin
	Name string `json:"name"`
	// Author of the plugin.
	Author string `json:"author"`
	// Version describes the SemVer of the plugin.
	Version string `json:"version"`
	// Description describes the plugin.
	Description string `json:"description"`
}
