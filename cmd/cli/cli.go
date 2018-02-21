/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package main

import (
	"encoding/json"
	_ "fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	//	image "github.com/sylabs/sy-go/pkg/image"
	//	runtime "github.com/sylabs/sy-go/pkg/runtime"
)

func main() {
	args := os.Args[1:2]

	spec := &specs.Spec{}
	spec.Process = &specs.Process{}
	spec.Root = &specs.Root{}
	spec.Root.Path = os.Args[2]
	spec.Process.Args = os.Args[3:]
	spec.Process.NoNewPrivileges = false
	spec.Linux = &specs.Linux{}
	spec.Linux.Namespaces = []specs.LinuxNamespace{
		specs.LinuxNamespace{Type: specs.PIDNamespace},
		specs.LinuxNamespace{Type: specs.NetworkNamespace},
		specs.LinuxNamespace{Type: specs.MountNamespace},
		specs.LinuxNamespace{Type: specs.IPCNamespace},
		specs.LinuxNamespace{Type: specs.UTSNamespace},
	}

	for _, arg := range args {
		switch arg {
		case "suid":
			cmd := exec.Command("build/wrapper-suid")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			j, err := json.Marshal(spec)
			if err != nil {
				log.Fatalln(err)
			}

			cmd.Stdin = strings.NewReader(string(j))
			err = cmd.Run()
			if err != nil {
				log.Fatalln(err)
			}

		case "userns":
			cmd := exec.Command("build/wrapper")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			spec.Linux.Namespaces = append(spec.Linux.Namespaces, specs.LinuxNamespace{Type: specs.UserNamespace})

			j, err := json.Marshal(spec)
			if err != nil {
				log.Fatalln(err)
			}

			cmd.Stdin = strings.NewReader(string(j))
			err = cmd.Run()
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
	/*
		img := image.SandboxFromPath("/path/to/sandbox")
		img.Root()
		runtime.Shell(spec)
		fmt.Println("Test")*/
}
