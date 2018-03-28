/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package image

// #cgo LDFLAGS: -lsycore -luuid
/*
#include <sys/types.h>

#include <uuid/uuid.h>

#include <sif/list.h>
#include <sif/sif.h>
#include <sif/sifaccess.h>
*/
import "C"

import "fmt"

type Sifinfo struct {
	sinfo C.Sifinfo
}

/*
Wrapper for sif_load()
int sif_load(char *filename, Sifinfo *info, int rdonly)
*/
func SifLoad(filename string, info *Sifinfo, rdonly int) error {
	ret := C.sif_load(C.CString(filename), &info.sinfo, C.int(rdonly))
	if ret != 0 {
		err := fmt.Errorf("%s", C.GoString(C.sif_strerror(C.siferrno)))
		return err
	}
	return nil
}

/*
Wrapper for sif_unload()
int sif_unload(Sifinfo *info)
*/
func SifUnload(info *Sifinfo) error {
	ret := C.sif_unload(&info.sinfo)
	if ret != 0 {
		err := fmt.Errorf("%s", C.sif_strerror(C.siferrno))
		return err
	}
	return nil
}

/*
Wrapper for sif_printheader()
sif_printheader(Sifinfo info)
*/
func SifPrintHeader(info *Sifinfo) {
	C.sif_printheader(&info.sinfo)
}
