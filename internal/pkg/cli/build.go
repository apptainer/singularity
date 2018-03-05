/*
 Copyright (c) 2018, Sylabs, Inc. All rights reserved.

 This software is licensed under a 3-clause BSD license.  Please
 consult LICENSE file distributed with the sources of this project regarding
 your rights to use or distribute this software.
*/
package cli

import (
	"github.com/spf13/cobra"
)

// Build cmd local vars
var (
	sandbox  bool
	writable bool
	remote   bool
	force    bool
	noTest   bool
	section  string
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use: "build",
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func init() {
	singularityCmd.AddCommand(buildCmd)

	buildCmd.PersistentFlags().BoolVarP(&sandbox, "sandbox", "s", false, "")
	buildCmd.PersistentFlags().BoolVarP(&writable, "writable", "w", false, "")
	buildCmd.PersistentFlags().BoolVarP(&force, "force", "f", false, "")
	buildCmd.PersistentFlags().BoolVarP(&noTest, "notest", "T", false, "")
	buildCmd.PersistentFlags().StringVarP(&section, "section", "s", "post", "")

	buildCmd.SetHelpTemplate(`
	The build command compiles a container per a recipe (definition file) or based
	on a URI, location, or archive.
		
	CONTAINER PATH:
		When Singularity builds the container, the output format can be one of
		multiple formats:
		
			default:    The compressed Singularity read only image format (default)
			libray:		The container image will be stored on the libray after the
						build process
			sandbox:    This is a read-write container within a directory structure
			writable:   Legacy writable image format
	
		note: A common workflow is to use the "sandbox" mode for development of the
		container, and then build it as a default Singularity image  when done.
		This format can not be modified.
		
	BUILD SPEC TARGET:
		The build spec target is a Singularity recipe, local image, archive, or URI
		that can be used to create a Singularity container. Several different
		local target formats exist:
	
			def file  : This is a recipe for building a container (examples below)
			directory:  A directory structure containing a (ch)root file system
			image:      A local image on your machine (will convert to squashfs if
						it is legacy or writable format)
			tar/tar.gz: An archive file which contains the above directory format
						(must have .tar in the filename!)
	
		Targets can also be remote and defined by a URI of the following formats:
	
			shub://     Build from a Singularity registry (Singularity Hub default)
			docker://   This points to a Docker registry (Docker Hub default)
		
	BUILD OPTIONS:
		-s|--sandbox    Build a sandbox rather then a read only compressed image
		-w|--writable   Build a writable image (warning: deprecated due to sparse
						file image corruption issues)
		-f|--force   Force a rebootstrap of a base OS (note: this does not
						delete what is currently in the image, just causes the core
						to be reinstalled)
		-T|--notest     Bootstrap without running tests in %test section
		-s|--section    Only run a given section within the recipe file (setup,
						post, files, environment, test, labels, none)		
		
	CHECKS OPTIONS:
		-c|--checks    enable checks
		-t|--tag       specify a check tag (not default)
		-l|--low       Specify low threshold (all checks, default)
		-m|--med       Perform medium and high checks
		-h|--high      Perform only checks at level high

	REMOTE BUILD OPTIONS:
		-d|--detach    run remote build in detached mode (no stdin output from build)
		
		
	See singularity check --help for available tags
		
	DEF FILE BASEOS EXAMPLES:
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
			MirrorURL: http://mirror.centos.org/centos-%{OSVERSION}/%{OSVERSION}/os/\$basearch/
			Include: yum
		
		Debian/Ubuntu:
			Bootstrap: debootstrap
			OSVersion: trusty
			MirrorURL: http://us.archive.ubuntu.com/ubuntu/
		
		Local Image:
			Bootstrap: localimage
			From: /home/dave/starter.img
		
		
	DEFFILE SECTION EXAMPLES:
		
		%setup
			echo "This is a scriptlet that will be executed on the host, as root, after"
			echo "the container has been bootstrapped. To install things into the container"
			echo "reference the file system location with \$SINGULARITY_ROOTFS"
		
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
		
		
	DEFFILE SCI-F SECTION EXAMPLES:
		
		%appinstall app1
			echo "These are steps to install an app using the SCI-F format"
		
		%appenv app1
			APP1VAR=app1value
			export APP1VAR
		
		%apphelp app1
			This is a help doc for running app1
		
		%apprun app1
			echo "this is a runscript for app1"
		
		%applabels app1
			AUTHOR tolkien
		
		%appfiles app1
			/file/on/host/foo.txt /file/in/contianer/foo.txt
		
		%appsetup app1
			echo "a %setup section (see above) for apps"
		
		%apptest app1
			echo "some test for an app"
		
		
	EXAMPLES:
		
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
`)
	buildCmd.SetUsageTemplate(`
USAGE: singularity [...] build [build options...] <container path> <deffile path>
    `)
}
