

package cacheCli

import (
	"fmt"
	"strings"
	"io/ioutil"
	"os"
//	"math"

//	"strconv"
//	"time"

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

//	fmt.Println("HELLO WORLD!!!!!!!!!")
	sylog.Debugf("Starting list...")

//	where := join(cache.Library(), "/")

//	fmt.Println("WHERE: ", where)



	files, err := ioutil.ReadDir(cache.Library())
	if err != nil {
		sylog.Fatalf("%v", err)
		os.Exit(255)
	}

	fmt.Printf("%-20s %-14s %s\n", "NAME", "DATE CREATED", "SIZE")
//	for _, file := range files {
//		fmt.Printf("%-16s %-8d %s\n", file.Name, file.Pid, file.Image)
//	}


	for _, f := range files {
//		fmt.Println("dir", f.Name())
		cont, err := ioutil.ReadDir(join(cache.Library(), "/", f.Name()))
		if err != nil {
			sylog.Fatalf("%v", err)
			os.Exit(255)
		}
		for _, c := range cont {
			// get last modified time
//			file, err := os.Stat(join(cache.Library(), "/", f.Name()))
			file, err := os.Stat(join(cache.Library(), "/", f.Name(), "/", c.Name()))
			if err != nil {
				fmt.Println(err)
				os.Exit(100)
			}

//			fi, err := os.Stat(join(cache.Library(), "/", f.Name()))
//			if err != nil {
//				fmt.Println(err)
//				os.Exit(100)
//			}
			// get the size

//			fmt.Println("INFO:  ", find_size(file.Size()))

//			fmt.Printf("The file is: %v bytes long:  %v\n", size, join(cache.Library(), "/", f.Name(), "/", c.Name()))

			fmt.Printf("%-20s %-14s %s\n", c.Name(), file.ModTime().Format("2006-01-02"), find_size(file.Size()))
		}		

	}


	return nil
}
