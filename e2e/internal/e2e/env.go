// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

type TestEnv struct {
	RunDisabled   bool
	CmdPath       string
	ImagePath     string
	OrasTestImage string
	TestDir       string
	TestRegistry  string
	KeyringDir    string
}
