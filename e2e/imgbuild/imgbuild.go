// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
)

var testFileContent = "Test file content\n"

type imgBuildTests struct {
	env e2e.TestEnv
}

func (c *imgBuildTests) buildFrom(t *testing.T) {
	e2e.PrepRegistry(t, c.env)

	tests := []struct {
		name       string
		dependency string
		buildSpec  string
		sandbox    bool
	}{
		{"BusyBox", "", "../examples/busybox/Singularity", false},
		{"Debootstrap", "debootstrap", "../examples/debian/Singularity", true},
		{"DockerURI", "", "docker://busybox", true},
		{"DockerDefFile", "", "../examples/docker/Singularity", true},
		// TODO(mem): reenable this; disabled while shub is down
		// {"ShubURI", "", "shub://GodloveD/busybox", true},
		// TODO(mem): reenable this; disabled while shub is down
		// {"ShubDefFile", "", "../examples/shub/Singularity", true},
		{"LibraryDefFile", "", "../examples/library/Singularity", true},
		{"OrasURI", "", c.env.OrasTestImage, true},
		{"Yum", "yum", "../examples/centos/Singularity", true},
		{"Zypper", "zypper", "../examples/opensuse/Singularity", true},
	}

	for _, tt := range tests {
		imagePath := path.Join(c.env.TestDir, "container")

		// conditionally build a sandbox
		args := []string{}
		if tt.sandbox {
			args = []string{"--sandbox"}
		}
		args = append(args, imagePath, tt.buildSpec)

		e2e.RunSingularity(
			t,
			tt.name,
			e2e.WithPrivileges(true),
			e2e.WithCommand("build"),
			e2e.WithArgs(args...),
			e2e.PreRun(func(t *testing.T) {
				if tt.dependency != "" {
					if _, err := exec.LookPath(tt.dependency); err != nil {
						t.Skipf("%v not found in path", tt.dependency)
					}
				}
			}),
			e2e.PostRun(func(t *testing.T) {
				defer os.RemoveAll(imagePath)

				e2e.ImageVerify(t, c.env.CmdPath, imagePath)
			}),
			e2e.ExpectExit(0),
		)
	}
}

func (c *imgBuildTests) nonRootBuild(t *testing.T) {
	tests := []struct {
		name      string
		buildSpec string
		sandbox   bool
	}{
		{
			name:      "local sif",
			buildSpec: "testdata/busybox.sif",
			sandbox:   false,
		},
		{
			name:      "local sif to sandbox",
			buildSpec: "testdata/busybox.sif",
			sandbox:   true,
		},
		{
			name:      "library sif",
			buildSpec: "library://sylabs/tests/busybox:1.0.0",
			sandbox:   false,
		},
		{
			name:      "library sif sandbox",
			buildSpec: "library://sylabs/tests/busybox:1.0.0",
			sandbox:   true,
		},
		{
			name:      "library sif sha",
			buildSpec: "library://sylabs/tests/busybox:sha256.8b5478b0f2962eba3982be245986eb0ea54f5164d90a65c078af5b83147009ba",
			sandbox:   false,
		},
		// TODO: uncomment when shub is working
		//{
		//		name:      "shub busybox",
		//		buildSpec: "shub://GodloveD/busybox",
		//		sandbox:   false,
		//},
		{
			name:      "docker busybox",
			buildSpec: "docker://busybox:latest",
			sandbox:   false,
		},
	}

	for _, tt := range tests {
		imagePath := path.Join(c.env.TestDir, "container")

		// conditionally build a sandbox
		args := []string{}
		if tt.sandbox {
			args = []string{"--sandbox"}
		}
		args = append(args, imagePath, tt.buildSpec)

		e2e.RunSingularity(
			t,
			tt.name,
			e2e.WithPrivileges(false),
			e2e.WithCommand("build"),
			e2e.WithArgs(args...),
			e2e.PostRun(func(t *testing.T) {
				defer os.RemoveAll(imagePath)

				e2e.ImageVerify(t, c.env.CmdPath, imagePath)
			}),
			e2e.ExpectExit(0),
		)
	}
}

func (c *imgBuildTests) buildLocalImage(t *testing.T) {
	imagePath1 := path.Join(c.env.TestDir, "container1")
	imagePath2 := path.Join(c.env.TestDir, "container2")
	imagePath3 := path.Join(c.env.TestDir, "container3")

	liDefFile := e2e.PrepareDefFile(e2e.DefFileDetails{
		Bootstrap: "localimage",
		From:      imagePath1,
	})
	defer os.Remove(liDefFile)

	labels := make(map[string]string)
	labels["FOO"] = "bar"
	liLabelDefFile := e2e.PrepareDefFile(e2e.DefFileDetails{
		Bootstrap: "localimage",
		From:      imagePath2,
		Labels:    labels,
	})
	defer os.Remove(liLabelDefFile)

	type testSpec struct {
		name      string
		imagePath string
		buildSpec string
		force     bool
		sandbox   bool
	}

	tests := []struct {
		name  string
		steps []testSpec
	}{
		{"SIFToSIF", []testSpec{
			{"BusyBox", imagePath1, "../examples/busybox/Singularity", false, false},
			{"SIF", imagePath2, imagePath1, false, false},
		}},
		{"SandboxToSIF", []testSpec{
			{"BusyBoxSandbox", imagePath1, "../examples/busybox/Singularity", false, true},
			{"SIF", imagePath2, imagePath1, false, false},
		}},
		{"LocalImage", []testSpec{
			{"BusyBox", imagePath1, "../examples/busybox/Singularity", false, false},
			{"LocalImage", imagePath2, liDefFile, false, false},
			{"LocalImageLabel", imagePath3, liLabelDefFile, false, false},
		}},
		{"LocalImageSandbox", []testSpec{
			{"BusyBoxSandbox", imagePath2, "../examples/busybox/Singularity", true, true},
			{"LocalImageLabel", imagePath3, liLabelDefFile, false, false},
		}},
	}

	for _, tt := range tests {

		// prep targets for removal
		t.Run(tt.name, e2e.Privileged(func(t *testing.T) {
			for _, ts := range tt.steps {
				defer os.RemoveAll(ts.imagePath)
			}

			for _, ts := range tt.steps {
				args := []string{}
				if ts.force {
					args = append([]string{"--force"}, args...)
				}
				if ts.sandbox {
					args = append([]string{"--sandbox"}, args...)
				}
				args = append(args, ts.imagePath, ts.buildSpec)

				e2e.RunSingularity(
					t,
					tt.name,
					e2e.WithPrivileges(true),
					e2e.WithCommand("build"),
					e2e.WithArgs(args...),
					e2e.PostRun(func(t *testing.T) {
						e2e.ImageVerify(t, c.env.CmdPath, ts.imagePath)
					}),
					e2e.ExpectExit(0),
				)
			}
		}))
	}
}

func (c *imgBuildTests) badPath(t *testing.T) {
	imagePath := path.Join(c.env.TestDir, "container")
	e2e.RunSingularity(
		t,
		"bad path",
		e2e.WithPrivileges(true),
		e2e.WithCommand("build"),
		e2e.WithArgs(imagePath, "/some/dumb/path"),
		e2e.ExpectExit(255),
	)
}

func (c *imgBuildTests) buildMultiStageDefinition(t *testing.T) {
	tmpfile, err := e2e.WriteTempFile(c.env.TestDir, "testFile-", testFileContent)
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpfile) // clean up

	tests := []struct {
		name    string
		force   bool
		sandbox bool
		dfd     []e2e.DefFileDetails
		correct e2e.DefFileDetails // a bit hacky, but this allows us to check final image for correct artifacts
	}{
		// Simple copy from stage one to final stage
		{"FileCopySimple", false, true, []e2e.DefFileDetails{
			{
				Bootstrap: "docker",
				From:      "alpine:latest",
				Stage:     "one",
				Files: []e2e.FilePair{
					{
						Src: tmpfile,
						Dst: "StageOne2.txt",
					},
					{
						Src: tmpfile,
						Dst: "StageOne.txt",
					},
				},
			},
			{
				Bootstrap: "docker",
				From:      "alpine:latest",
				FilesFrom: []e2e.FileSection{
					{
						Stage: "one",
						Files: []e2e.FilePair{
							{
								Src: "StageOne2.txt",
								Dst: "StageOneCopy2.txt",
							},
							{
								Src: "StageOne.txt",
								Dst: "StageOneCopy.txt",
							},
						}}},
			}},
			e2e.DefFileDetails{
				Files: []e2e.FilePair{
					{
						Src: tmpfile,
						Dst: "StageOneCopy2.txt",
					},
					{
						Src: tmpfile,
						Dst: "StageOneCopy.txt",
					},
				},
			},
		},
		// Complex copy of files from stage one and two to stage three, then final copy from three to final stage
		{"FileCopyComplex", false, true,
			[]e2e.DefFileDetails{
				{
					Bootstrap: "docker",
					From:      "alpine:latest",
					Stage:     "one",
					Files: []e2e.FilePair{
						{
							Src: tmpfile,
							Dst: "StageOne2.txt",
						},
						{
							Src: tmpfile,
							Dst: "StageOne.txt",
						},
					},
				},
				{
					Bootstrap: "docker",
					From:      "alpine:latest",
					Stage:     "two",
					Files: []e2e.FilePair{
						{
							Src: tmpfile,
							Dst: "StageTwo2.txt",
						},
						{
							Src: tmpfile,
							Dst: "StageTwo.txt",
						},
					},
				},
				{
					Bootstrap: "docker",
					From:      "alpine:latest",
					Stage:     "three",
					FilesFrom: []e2e.FileSection{
						{
							Stage: "one",
							Files: []e2e.FilePair{
								{
									Src: "StageOne2.txt",
									Dst: "StageOneCopy2.txt",
								},
								{
									Src: "StageOne.txt",
									Dst: "StageOneCopy.txt",
								},
							}},
						{
							Stage: "two",
							Files: []e2e.FilePair{
								{
									Src: "StageTwo2.txt",
									Dst: "StageTwoCopy2.txt",
								},
								{
									Src: "StageTwo.txt",
									Dst: "StageTwoCopy.txt",
								},
							},
						}},
				},
				{
					Bootstrap: "docker",
					From:      "alpine:latest",
					FilesFrom: []e2e.FileSection{
						{
							Stage: "three",
							Files: []e2e.FilePair{
								{
									Src: "StageOneCopy2.txt",
									Dst: "StageOneCopyFinal2.txt",
								},
								{
									Src: "StageOneCopy.txt",
									Dst: "StageOneCopyFinal.txt",
								},
								{
									Src: "StageTwoCopy2.txt",
									Dst: "StageTwoCopyFinal2.txt",
								},
								{
									Src: "StageTwoCopy.txt",
									Dst: "StageTwoCopyFinal.txt",
								},
							}}},
				},
			},
			e2e.DefFileDetails{
				Files: []e2e.FilePair{
					{
						Src: tmpfile,
						Dst: "StageOneCopyFinal2.txt",
					},
					{
						Src: tmpfile,
						Dst: "StageOneCopyFinal.txt",
					},
					{
						Src: tmpfile,
						Dst: "StageTwoCopyFinal2.txt",
					},
					{
						Src: tmpfile,
						Dst: "StageTwoCopyFinal.txt",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		imagePath := path.Join(c.env.TestDir, "container")
		defFile := e2e.PrepareMultiStageDefFile(tt.dfd)

		args := []string{}
		if tt.force {
			args = append([]string{"--force"}, args...)
		}
		if tt.sandbox {
			args = append([]string{"--sandbox"}, args...)
		}
		args = append(args, imagePath, defFile)

		e2e.RunSingularity(
			t,
			tt.name,
			e2e.WithPrivileges(true),
			e2e.WithCommand("build"),
			e2e.WithArgs(args...),
			e2e.PostRun(func(t *testing.T) {
				defer os.Remove(defFile)
				defer os.RemoveAll(imagePath)

				e2e.DefinitionImageVerify(t, c.env.CmdPath, imagePath, tt.correct)
			}),
			e2e.ExpectExit(0),
		)
	}
}

func (c *imgBuildTests) buildDefinition(t *testing.T) {
	tmpfile, err := e2e.WriteTempFile(c.env.TestDir, "testFile-", testFileContent)
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpfile) // clean up

	tests := []struct {
		name    string
		force   bool
		sandbox bool
		dfd     e2e.DefFileDetails
	}{
		{"Empty", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
		}},
		{"Help", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Help: []string{
				"help info line 1",
				"help info line 2",
				"help info line 3",
			},
		}},
		{"Files", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Files: []e2e.FilePair{
				{
					Src: tmpfile,
					Dst: "NewName2.txt",
				},
				{
					Src: tmpfile,
					Dst: "NewName.txt",
				},
			},
		}},
		{"Test", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Test: []string{
				"echo testscript line 1",
				"echo testscript line 2",
				"echo testscript line 3",
			},
		}},
		{"Startscript", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			StartScript: []string{
				"echo startscript line 1",
				"echo startscript line 2",
				"echo startscript line 3",
			},
		}},
		{"Runscript", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			RunScript: []string{
				"echo runscript line 1",
				"echo runscript line 2",
				"echo runscript line 3",
			},
		}},
		{"Env", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Env: []string{
				"testvar1=one",
				"testvar2=two",
				"testvar3=three",
			},
		}},
		{"Labels", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Labels: map[string]string{
				"customLabel1": "one",
				"customLabel2": "two",
				"customLabel3": "three",
			},
		}},
		{"Pre", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Pre: []string{
				filepath.Join(c.env.TestDir, "PreFile1"),
			},
		}},
		{"Setup", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Setup: []string{
				filepath.Join(c.env.TestDir, "SetupFile1"),
			},
		}},
		{"Post", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Post: []string{
				"PostFile1",
			},
		}},
		{"AppHelp", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []e2e.AppDetail{
				{
					Name: "foo",
					Help: []string{
						"foo help info line 1",
						"foo help info line 2",
						"foo help info line 3",
					},
				},
				{
					Name: "bar",
					Help: []string{
						"bar help info line 1",
						"bar help info line 2",
						"bar help info line 3",
					},
				},
			},
		}},
		{"AppEnv", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []e2e.AppDetail{
				{
					Name: "foo",
					Env: []string{
						"testvar1=fooOne",
						"testvar2=fooTwo",
						"testvar3=fooThree",
					},
				},
				{
					Name: "bar",
					Env: []string{
						"testvar1=barOne",
						"testvar2=barTwo",
						"testvar3=barThree",
					},
				},
			},
		}},
		{"AppLabels", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []e2e.AppDetail{
				{
					Name: "foo",
					Labels: map[string]string{
						"customLabel1": "fooOne",
						"customLabel2": "fooTwo",
						"customLabel3": "fooThree",
					},
				},
				{
					Name: "bar",
					Labels: map[string]string{
						"customLabel1": "barOne",
						"customLabel2": "barTwo",
						"customLabel3": "barThree",
					},
				},
			},
		}},
		{"AppFiles", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []e2e.AppDetail{
				{
					Name: "foo",
					Files: []e2e.FilePair{
						{
							Src: tmpfile,
							Dst: "FooFile2.txt",
						},
						{
							Src: tmpfile,
							Dst: "FooFile.txt",
						},
					},
				},
				{
					Name: "bar",
					Files: []e2e.FilePair{
						{
							Src: tmpfile,
							Dst: "BarFile2.txt",
						},
						{
							Src: tmpfile,
							Dst: "BarFile.txt",
						},
					},
				},
			},
		}},
		{"AppInstall", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []e2e.AppDetail{
				{
					Name: "foo",
					Install: []string{
						"FooInstallFile1",
					},
				},
				{
					Name: "bar",
					Install: []string{
						"BarInstallFile1",
					},
				},
			},
		}},
		{"AppRun", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []e2e.AppDetail{
				{
					Name: "foo",
					Run: []string{
						"echo foo runscript line 1",
						"echo foo runscript line 2",
						"echo foo runscript line 3",
					},
				},
				{
					Name: "bar",
					Run: []string{
						"echo bar runscript line 1",
						"echo bar runscript line 2",
						"echo bar runscript line 3",
					},
				},
			},
		}},
		{"AppTest", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []e2e.AppDetail{
				{
					Name: "foo",
					Test: []string{
						"echo foo testscript line 1",
						"echo foo testscript line 2",
						"echo foo testscript line 3",
					},
				},
				{
					Name: "bar",
					Test: []string{
						"echo bar testscript line 1",
						"echo bar testscript line 2",
						"echo bar testscript line 3",
					},
				},
			},
		}},
	}

	for _, tt := range tests {
		imagePath := path.Join(c.env.TestDir, "container")
		defFile := e2e.PrepareDefFile(tt.dfd)

		args := []string{}
		if tt.force {
			args = append([]string{"--force"}, args...)
		}
		if tt.sandbox {
			args = append([]string{"--sandbox"}, args...)
		}
		args = append(args, imagePath, defFile)

		e2e.RunSingularity(
			t,
			tt.name,
			e2e.WithPrivileges(true),
			e2e.WithCommand("build"),
			e2e.WithArgs(args...),
			e2e.PostRun(func(t *testing.T) {
				defer os.Remove(defFile)
				defer os.RemoveAll(imagePath)

				e2e.DefinitionImageVerify(t, c.env.CmdPath, imagePath, tt.dfd)
			}),
			e2e.ExpectExit(0),
		)
	}
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &imgBuildTests{
		env: env,
	}

	return func(t *testing.T) {
		// builds from definition file and URI
		t.Run("From", c.buildFrom)
		// build and image from an existing image
		t.Run("FromLocalImage", c.buildLocalImage)
		// build sifs from non-root
		t.Run("NonRootBuild", c.nonRootBuild)
		// try to build from a non existen path
		t.Run("badPath", c.badPath)
		// builds from definition template
		t.Run("Definition", c.buildDefinition)
		// multistage build from definition templates
		t.Run("MultiStage", c.buildMultiStageDefinition)
	}
}
