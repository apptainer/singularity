// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/test"
)

var testFileContent = "Test file content\n"

type testingEnv struct {
	// base env for running tests
	CmdPath     string `split_words:"true"`
	TestDir     string `split_words:"true"`
	RunDisabled bool   `default:"false"`
}

var testenv testingEnv

func buildFrom(t *testing.T) {
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
		{"SHubURI", "", "shub://GodloveD/busybox", true},
		{"SHubDefFile", "", "../examples/shub/Singularity", true},
		{"LibraryDefFile", "", "../examples/library/Singularity", true},
		{"Yum", "yum", "../examples/centos/Singularity", true},
		{"Zypper", "zypper", "../examples/opensuse/Singularity", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithPrivilege(func(t *testing.T) {
			if tt.dependency != "" {
				if _, err := exec.LookPath(tt.dependency); err != nil {
					t.Skipf("%v not found in path", tt.dependency)
				}
			}

			opts := e2e.BuildOpts{
				Sandbox: tt.sandbox,
			}

			imagePath := path.Join(testenv.TestDir, "container")
			defer os.RemoveAll(imagePath)

			if b, err := e2e.ImageBuild(testenv.CmdPath, opts, imagePath, tt.buildSpec); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
			e2e.ImageVerify(t, testenv.CmdPath, imagePath, false, testenv.RunDisabled)
		}))
	}
}

func buildMultiStage(t *testing.T) {
	imagePath1 := path.Join(testenv.TestDir, "container1")
	imagePath2 := path.Join(testenv.TestDir, "container2")
	imagePath3 := path.Join(testenv.TestDir, "container3")

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
		labels    bool
	}

	tests := []struct {
		name  string
		steps []testSpec
	}{
		{"SIFToSIF", []testSpec{
			{"BusyBox", imagePath1, "../examples/busybox/Singularity", false, false, false},
			{"SIF", imagePath2, imagePath1, false, false, false},
		}},
		{"SandboxToSIF", []testSpec{
			{"BusyBoxSandbox", imagePath1, "../examples/busybox/Singularity", false, true, false},
			{"SIF", imagePath2, imagePath1, false, false, false},
		}},
		{"LocalImage", []testSpec{
			{"BusyBox", imagePath1, "../examples/busybox/Singularity", false, false, false},
			{"LocalImage", imagePath2, liDefFile, false, false, false},
			{"LocalImageLabel", imagePath3, liLabelDefFile, false, false, true},
		}},
		{"LocalImageSandbox", []testSpec{
			{"BusyBoxSandbox", imagePath2, "../examples/busybox/Singularity", true, true, false},
			{"LocalImageLabel", imagePath3, liLabelDefFile, false, false, true},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithPrivilege(func(t *testing.T) {
			for _, ts := range tt.steps {
				defer os.RemoveAll(ts.imagePath)
			}

			for _, ts := range tt.steps {
				t.Run(ts.name, test.WithPrivilege(func(t *testing.T) {
					opts := e2e.BuildOpts{
						Force:   ts.force,
						Sandbox: ts.sandbox,
					}

					if b, err := e2e.ImageBuild(testenv.CmdPath, opts, ts.imagePath, ts.buildSpec); err != nil {
						t.Log(string(b))
						t.Fatalf("unexpected failure: %v", err)
					}
					e2e.ImageVerify(t, testenv.CmdPath, ts.imagePath, ts.labels, testenv.RunDisabled)
				}))
			}
		}))
	}
}

func badPath(t *testing.T) {
	test.EnsurePrivilege(t)

	imagePath := path.Join(testenv.TestDir, "container")
	defer os.RemoveAll(imagePath)

	if b, err := e2e.ImageBuild(testenv.CmdPath, e2e.BuildOpts{}, imagePath, "/some/dumb/path"); err == nil {
		t.Log(string(b))
		t.Fatal("unexpected success")
	}
}

func multiStageDefinition(t *testing.T) {
	tmpfile, err := ioutil.TempFile(testenv.TestDir, "testFile-")
	if err != nil {
		log.Fatal(err)
	}

	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write([]byte(testFileContent)); err != nil {
		log.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)
	}

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
						Src: tmpfile.Name(),
						Dst: "StageOne2.txt",
					},
					{
						Src: tmpfile.Name(),
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
						Src: tmpfile.Name(),
						Dst: "StageOneCopy2.txt",
					},
					{
						Src: tmpfile.Name(),
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
							Src: tmpfile.Name(),
							Dst: "StageOne2.txt",
						},
						{
							Src: tmpfile.Name(),
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
							Src: tmpfile.Name(),
							Dst: "StageTwo2.txt",
						},
						{
							Src: tmpfile.Name(),
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
						Src: tmpfile.Name(),
						Dst: "StageOneCopyFinal2.txt",
					},
					{
						Src: tmpfile.Name(),
						Dst: "StageOneCopyFinal.txt",
					},
					{
						Src: tmpfile.Name(),
						Dst: "StageTwoCopyFinal2.txt",
					},
					{
						Src: tmpfile.Name(),
						Dst: "StageTwoCopyFinal.txt",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithPrivilege(func(t *testing.T) {

			defFile := e2e.PrepareMultiStageDefFile(tt.dfd)
			defer os.Remove(defFile)

			opts := e2e.BuildOpts{
				Sandbox: tt.sandbox,
			}

			imagePath := path.Join(testenv.TestDir, "container")
			defer os.RemoveAll(imagePath)

			if b, err := e2e.ImageBuild(testenv.CmdPath, opts, imagePath, defFile); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}

			e2e.DefinitionImageVerify(t, testenv.CmdPath, imagePath, tt.correct)
		}))
	}

}

func buildDefinition(t *testing.T) {

	tmpfile, err := ioutil.TempFile(testenv.TestDir, "testFile-")
	if err != nil {
		log.Fatal(err)
	}

	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write([]byte(testFileContent)); err != nil {
		log.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)
	}

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
					Src: tmpfile.Name(),
					Dst: "NewName2.txt",
				},
				{
					Src: tmpfile.Name(),
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
				filepath.Join(testenv.TestDir, "PreFile1"),
			},
		}},
		{"Setup", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Setup: []string{
				filepath.Join(testenv.TestDir, "SetupFile1"),
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
							Src: tmpfile.Name(),
							Dst: "FooFile2.txt",
						},
						{
							Src: tmpfile.Name(),
							Dst: "FooFile.txt",
						},
					},
				},
				{
					Name: "bar",
					Files: []e2e.FilePair{
						{
							Src: tmpfile.Name(),
							Dst: "BarFile2.txt",
						},
						{
							Src: tmpfile.Name(),
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
		t.Run(tt.name, test.WithPrivilege(func(t *testing.T) {

			defFile := e2e.PrepareDefFile(tt.dfd)
			defer os.Remove(defFile)

			opts := e2e.BuildOpts{
				Sandbox: tt.sandbox,
			}

			imagePath := path.Join(testenv.TestDir, "container")
			defer os.RemoveAll(imagePath)

			if b, err := e2e.ImageBuild(testenv.CmdPath, opts, imagePath, defFile); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
			e2e.DefinitionImageVerify(t, testenv.CmdPath, imagePath, tt.dfd)
		}))
	}
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	e2e.LoadEnv(t, &testenv)

	t.Log(testenv)

	// builds from definition file and URI
	t.Run("From", buildFrom)
	// build and image from an existing image
	t.Run("multistage", buildMultiStage)
	// try to build from a non existen path
	t.Run("badPath", badPath)
	// builds from definition template
	t.Run("Definition", buildDefinition)
	// multistage build from definition templates
	t.Run("MultiStage", buildMultiStage)
}
