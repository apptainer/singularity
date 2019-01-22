

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

func ListSingularityCache() error {

	fmt.Println("HELLO WORLD!!!!!!!!!")
	sylog.Debugf("Starting list...")

	where := join(cache.Library(), "/")

	fmt.Println("WHERE: ", where)



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
			fmt.Printf("%-20s %-14s %s\n", c.Name(), "bar", "size")
//			fmt.Println("Container: ", c.Name())
		}		

	}


	return nil
}
