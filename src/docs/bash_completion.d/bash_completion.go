package main

import (
	"fmt"
	"os"

	"github.com/singularityware/singularity/src/cmd/singularity/cli"
)

func main() {

	if err := cli.SingularityCmd.GenBashCompletionFile(os.Args[1]); err != nil {
		fmt.Println(err)
		return
	}
}
