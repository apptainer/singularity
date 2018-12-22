// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
// Copyright (c) 2017, Yannick Cote <yhcote@gmail.com> All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sif

import (
	"bytes"
	"log"
)

var (
	sifLoggerBuf bytes.Buffer
	siflog       = log.New(&sifLoggerBuf, "", log.Ldate|log.Ltime|log.Lshortfile)
)

func init() {
}
