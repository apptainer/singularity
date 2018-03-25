/*
Copyright (c) 2018, Sylabs, Inc. All rights reserved.
This software is licensed under a 3-clause BSD license.  Please
consult LICENSE file distributed with the sources of this project regarding
your rights to use or distribute this software.
*/

package main

import (
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	config "github.com/singularityware/singularity/internal/pkg/runtime/engine/singularity/config"
)

func main() {
	args := os.Args[1:2]

	oci, runtime := config.NewSingularityConfig("new")
	oci.Root.SetPath(os.Args[2])
	oci.Process.SetArgs(os.Args[3:])
	oci.Process.SetNoNewPrivileges(true)

	oci.RuntimeOciSpec.Linux = &specs.Linux{}
	oci.RuntimeOciSpec.Linux.Namespaces = []specs.LinuxNamespace{
		specs.LinuxNamespace{Type: specs.PIDNamespace},
		specs.LinuxNamespace{Type: specs.NetworkNamespace},
		specs.LinuxNamespace{Type: specs.MountNamespace},
		specs.LinuxNamespace{Type: specs.IPCNamespace},
		specs.LinuxNamespace{Type: specs.UTSNamespace},
	}

	for _, arg := range args {
		switch arg {
		case "suid":
			cmd := exec.Command("/tmp/wrapper-suid")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Env = []string{"MESSAGELEVEL=0", "SRUNTIME=singularity"}
			j, err := runtime.GetConfig()
			if err != nil {
				log.Fatalln(err)
			}

			cmd.Stdin = strings.NewReader(string(j))
			err = cmd.Run()
			if err != nil {
				log.Fatalln(err)
			}

		case "userns":
			cmd := exec.Command("/tmp/wrapper")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Env = []string{"MESSAGELEVEL=0", "SRUNTIME=singularity"}

			oci.RuntimeOciSpec.Linux.Namespaces = append(oci.RuntimeOciSpec.Linux.Namespaces, specs.LinuxNamespace{Type: specs.UserNamespace})

			j, err := runtime.GetConfig()
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
}
