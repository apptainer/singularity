/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package cli

import (
	"fmt"
	"os"

	"github.com/singularityware/singularity/pkg/build"
	"github.com/spf13/cobra"

    "github.com/singularityware/singularity/docs"
)

var (
	Remote      bool
	RemoteURL   string
	Sandbox     bool
	Writable    bool
	Force       bool
	NoTest      bool
	Sections    []string
    MakeManPage bool
    ManPageDir string
)

func init() {

    manHelp := func(c *cobra.Command, args []string) {
        docs.DispManPg("singularity-build")
    }

	BuildCmd.Flags().SetInterspersed(false)
    BuildCmd.SetHelpFunc(manHelp)
	SingularityCmd.AddCommand(BuildCmd)

	BuildCmd.Flags().BoolVarP(&Sandbox, "sandbox", "s", false, "Build image as sandbox format (chroot directory structure)")
	BuildCmd.Flags().StringSliceVar(&Sections, "section", []string{}, "Only run specific section(s) of deffile (setup, post, files, environment, test, labels, none)")
	BuildCmd.Flags().BoolVarP(&Writable, "writable", "w", false, "Build image as writable (SIF with writable internal overlay)")
	BuildCmd.Flags().BoolVarP(&Force, "force", "f", false, "Delete and overwrite an image if it currently exists")
	BuildCmd.Flags().BoolVarP(&NoTest, "notest", "T", false, "Bootstrap without running tests in %test section")
	BuildCmd.Flags().BoolVarP(&Remote, "remote", "r", false, "Build image remotely")
	BuildCmd.Flags().StringVar(&RemoteURL, "remote-url", "localhost:5050", "Specify the URL of the remote builder")
}

// BuildCmd represents the build command
var BuildCmd = &cobra.Command{
    DisableFlagsInUseLine: true,
	Args: cobra.ExactArgs(2),

    Use: `build [local options...] <IMAGE PATH> <BUILD SPEC>`,

    Short: `The build command compiles a container per a recipe (definition file) or based
on a URI, location, or archive."`,

    Long: `

IMAGE PATH:

When Singularity builds the container, the output can be one of a few formats:

    default:    The compressed Singularity read only image format (default)
    sandbox:    This is a read-write container within a directory structure
    writable:   Legacy writable image format

note: It is a  common workflow to use the "sandbox" mode for development of the container, and then build it as a default Singularity image for production use. The default format is immutable.

BUILD SPEC:

The build spec target is a definition, local image, archive, or URI that can be used to create a Singularity container. Several different local target formats exist:

    def file  : This is a recipe for building a container (examples below)
    directory:  A directory structure containing a (ch)root file system
    image:      A local image on your machine (will convert to squashfs if
                it is legacy or writable format)
    tar/tar.gz: An archive file which contains the above directory format
                (must have .tar in the filename!)

Targets can also be remote and defined by a URI of the following formats:

    shub://     Build from a Singularity registry (Singularity Hub default)
    docker://   This points to a Docker registry (Docker Hub default)`,

    Example: `
DEF FILE BASE OS:

    Singularity Hub:
        Bootstrap: shub
        From: singularityhub/centos

    Docker:
        Bootstrap: docker
        From: tensorflow/tensorflow:latest
        IncludeCmd: yes # Use the CMD as runscript instead of ENTRYPOINT

    YUM/RHEL:
        Bootstrap: yum
        OSVersion: 7
        MirrorURL: http://mirror.centos.org/centos-%{OSVERSION}/%{OSVERSION}/os/$basearch/
        Include: yum

    Debian/Ubuntu:
        Bootstrap: debootstrap
        OSVersion: trusty
        MirrorURL: http://us.archive.ubuntu.com/ubuntu/

    Local Image:
        Bootstrap: localimage
        From: /home/dave/starter.img

DEFFILE SECTIONS:

    %setup
        echo "This is a scriptlet that will be executed on the host, as root, after"
        echo "the container has been bootstrapped. To install things into the container"
        echo "reference the file system location with $SINGULARITY_BUILDROOT"

    %post
        echo "This scriptlet section will be executed from within the container after"
        echo "the bootstrap/base has been created and setup"

    %test
        echo "Define any test commands that should be executed after container has been"
        echo "built. This scriptlet will be executed from within the running container"
        echo "as the root user. Pay attention to the exit/return value of this scriptlet"
        echo "as any non-zero exit code will be assumed as failure"
        exit 0

    %runscript
        echo "Define actions for the container to be executed with the run command or"
        echo "when container is executed."

    %startscript
        echo "Define actions for container to perform when started as an instance."

    %labels
        HELLO MOTO
        KEY VALUE

    %files
        /path/on/host/file.txt /path/on/container/file.txt
        relative_file.txt /path/on/container/relative_file.txt

    %environment
        LUKE=goodguy
        VADER=badguy
        HAN=someguy
        export HAN VADER LUKE

COMMANDS:

    Build a compressed image from a Singularity recipe file:
        $ singularity build /tmp/debian0.simg /path/to/debian.def

    Build a base compressed image from Docker Hub:
        $ singularity build /tmp/debian1.simg docker://debian:latest

    Build a base sandbox from DockerHub, make changes to it, then build image
        $ singularity build --sandbox /tmp/debian docker://debian:latest
        $ singularity exec --writable /tmp/debian apt-get install python
        $ singularity build /tmp/debian2.simg /tmp/debian


For additional help, please visit our public documentation pages which are
found at:

    http://singularity.lbl.gov/
`,

// TODO: Can we plz move this to another file to keep the CLI the CLI
	Run: func(cmd *cobra.Command, args []string) {
		var def build.Definition
		var b build.Builder
		var err error

		if silent {
			fmt.Println("Silent!")
		}

		if Sandbox {
			fmt.Println("Sandbox!")
		}

		if ok, err := build.IsValidURI(args[1]); ok && err == nil {
			// URI passed as arg[1]
			def, err = build.NewDefinitionFromURI(args[1])
			if err != nil {
				fmt.Println("Error: ", err)
				return
			}
		} else if !ok && err == nil {
			// Non-URI passed as arg[1]
			defFile, err := os.Open(args[1])
			if err != nil {
				fmt.Println("Error: ", err)
				return
			}

			def, err = build.ParseDefinitionFile(defFile)
			if err != nil {
				fmt.Println("Error: ", err)
				return
			}
		} else {
			// Error
			fmt.Println("Error: ", err)
			return
		}

		if Remote {
			b = build.NewRemoteBuilder(args[0], def, false, RemoteURL)

		} else {
			b, err = build.NewSifBuilder(args[0], def)
			if err != nil {
				fmt.Println("Error: ", err)
				return
			}
		}

		if err := b.Build(); err != nil {
			fmt.Println("Error: ", err)
			return
		}

		/*
			if Remote {
				doRemoteBuild(args[0], args[1])
			} else {
				if ok, err := build.IsValidURI(args[1]); ok && err == nil {
					u := strings.SplitN(args[1], "://", 2)
					b, err := build.NewSifBuilderFromURI(args[0], args[1])
					if err != nil {
						glog.Errorf("Image build system encountered an error: %s\n", err)
						return
					}
					b.Build()
				} else {
					glog.Fatalf("%s", err)
				}
			}*/

	},
	TraverseChildren: true,
}
