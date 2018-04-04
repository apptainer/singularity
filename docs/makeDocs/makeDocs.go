package main

import (
    "fmt"

    "github.com/singularityware/singularity/internal/pkg/cli"
    "github.com/spf13/cobra/doc"
)

func main() {
    header := &doc.GenManHeader{
        Title: "singularity-build",
        Section: "1",
    }
    err := doc.GenManTree(cli.BuildCmd, header, "/tmp")
        if err != nil {
            fmt.Println("Whoops!")
            return
    }
}

