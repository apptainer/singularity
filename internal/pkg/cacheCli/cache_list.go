// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cacheCli

import (
	"fmt"
	"strings"
	"io/ioutil"
	"os"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/client/cache"

)

func join(strs ...string) string {
    var sb strings.Builder
    for _, str := range strs {
        sb.WriteString(str)
    }
    return sb.String()
}

var err error

func visit(path string, f os.FileInfo, err error) error {
	fmt.Printf("Visited: %s\n", path)
	return nil
}

func find_size(size int64) string {
	var size_f float64
	if size <= 10000 {
		size_f = float64(size) / 1000
		return join(fmt.Sprintf("%.2f", size_f), " Kb")
	} else if size <= 1000000000 {
		size_f = float64(size) / 1000000
		return join(fmt.Sprintf("%.2f", size_f), " Mb")
	} else if size >= 1000000000 {
		size_f = float64(size) / 1000000000
		return join(fmt.Sprintf("%.2f", size_f), " Gb")
	}
	return "ERROR: failed to detect file size."
}

func ListSingularityCache() error {

	sylog.Debugf("Starting list...")


	files, err := ioutil.ReadDir(cache.Library())
	if err != nil {
		sylog.Fatalf("%v", err)
		os.Exit(255)
	}

	fmt.Printf("%-22s %-22s %-16s %s\n", "NAME", "DATE CREATED", "SIZE", "TYPE")

	for _, f := range files {
		cont, err := ioutil.ReadDir(join(cache.Library(), "/", f.Name()))
		if err != nil {
			sylog.Fatalf("%v", err)
			os.Exit(255)
		}
		for _, c := range cont {
//			file, err := os.Stat(join(cache.Library(), "/", f.Name()))
			file, err := os.Stat(join(cache.Library(), "/", f.Name(), "/", c.Name()))
			if err != nil {
				fmt.Println(err)
				os.Exit(100)
			}
			fmt.Printf("%-22s %-22s %-16s %s\n", c.Name(), file.ModTime().Format("2006-01-02 15:04:05"), find_size(file.Size()), "Library")
		}
	}


	blobs, err := ioutil.ReadDir(cache.OciTemp())
	if err != nil {
		sylog.Fatalf("%v", err)
		os.Exit(255)
	}

	for _, f := range blobs {

//		fmt.Println("INFO:  ", join(cache.OciTemp(), "/blobs"))
//		fmt.Println("BAR:  ", f.Name())

		blob, err := ioutil.ReadDir(join(cache.OciTemp(), "/", f.Name()))
		if err != nil {
			sylog.Fatalf("%v", err)
			os.Exit(255)
		}
		for _, b := range blob {

//			fmt.Println("INFO1: ", join(cache.OciTemp(), "/", b.Name()))
//			fmt.Println("INFO3: ", b.Name())

			file, err := os.Stat(join(cache.OciTemp(), "/", f.Name(), "/", b.Name()))
			if err != nil {
				fmt.Println(err)
				os.Exit(100)
			}
			fmt.Printf("%-22s %-22s %-16s %s\n", b.Name(), file.ModTime().Format("2006-01-02 15:04:05"), find_size(file.Size()), "Oci Tmp")
		}

	}

	return nil
}
