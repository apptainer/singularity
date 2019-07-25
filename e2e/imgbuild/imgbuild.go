// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/util/fs"

	uuid "github.com/satori/go.uuid"
	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/test/tool/require"
)

// only user, root and fakeroot profiles
var buildProfiles = []e2e.SingularityProfile{
	e2e.UserProfile,
	e2e.RootProfile,
	e2e.FakerootProfile,
}

// only root and fakeroot could build image from definition
var rootBuildProfiles = []e2e.SingularityProfile{
	e2e.RootProfile,
	e2e.FakerootProfile,
}

const (
	testFileContent = "Test file content\n"
	dumbPath        = "/some/dumb/path"
)

func buildFrom(ctx *e2e.TestContext) {
	t, env, profile := ctx.Get()

	if !profile.In(rootBuildProfiles...) {
		t.Skipf("%q could not run this test", profile)
	}

	e2e.PrepRegistry(t, env)

	tests := []struct {
		name             string
		dependency       string
		buildSpec        string
		sandbox          bool
		excludedProfiles []e2e.SingularityProfile
	}{
		{
			name:             "BusyBox",
			dependency:       "",
			buildSpec:        "../examples/busybox/Singularity",
			sandbox:          false,
			excludedProfiles: nil,
		},
		{
			name:             "Debootstrap",
			dependency:       "debootstrap",
			buildSpec:        "../examples/debian/Singularity",
			sandbox:          true,
			excludedProfiles: []e2e.SingularityProfile{e2e.FakerootProfile},
		},
		{
			name:             "DockerURI",
			dependency:       "",
			buildSpec:        "docker://busybox",
			sandbox:          true,
			excludedProfiles: nil,
		},
		{
			name:             "DockerDefFile",
			dependency:       "",
			buildSpec:        "../examples/docker/Singularity",
			sandbox:          true,
			excludedProfiles: nil,
		},
		// TODO(mem): reenable this; disabled while shub is down
		//{
		//	name:             "ShubURI",
		//	dependency:       "",
		//	buildSpec:        "shub://GodloveD/busybox",
		//	sandbox:          true,
		//	excludedProfiles: nil,
		//},
		// TODO(mem): reenable this; disabled while shub is down
		//{
		//	name:             "ShubDefFile",
		//	dependency:       "",
		//	buildSpec:        "../examples/shub/Singularity",
		//	sandbox:          true,
		//	excludedProfiles: nil,
		//},
		{
			name:             "LibraryDefFile",
			dependency:       "",
			buildSpec:        "../examples/library/Singularity",
			sandbox:          true,
			excludedProfiles: nil,
		},
		{
			name:             "OrasURI",
			dependency:       "",
			buildSpec:        env.OrasTestImage,
			sandbox:          true,
			excludedProfiles: nil,
		},
		{
			name:             "Yum",
			dependency:       "yum",
			buildSpec:        "../examples/centos/Singularity",
			sandbox:          true,
			excludedProfiles: nil,
		},
		{
			name:             "Zypper",
			dependency:       "zypper",
			buildSpec:        "../examples/opensuse/Singularity",
			sandbox:          true,
			excludedProfiles: []e2e.SingularityProfile{e2e.FakerootProfile},
		},
	}

	for _, tt := range tests {
		imagePath := path.Join(env.TestDir, uuid.NewV4().String())

		// conditionally build a sandbox
		args := []string{}
		if tt.sandbox {
			args = []string{"--sandbox"}
		}
		args = append(args, imagePath, tt.buildSpec)

		env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(profile),
			e2e.WithCommand("build"),
			e2e.WithArgs(args...),
			e2e.PreRun(func(t *testing.T) {
				if tt.dependency != "" {
					require.Command(t, tt.dependency)
				}
				if tt.excludedProfiles != nil && profile.In(tt.excludedProfiles...) {
					t.Skipf("%q excluded for this test", profile)
				}
			}),
			e2e.PostRun(func(t *testing.T) {
				defer os.RemoveAll(imagePath)

				e2e.ImageVerify(t, env.CmdPath, imagePath)
			}),
			e2e.ExpectExit(0),
		)
	}
}

func nonPrivilegedBuild(ctx *e2e.TestContext) {
	t, env, profile := ctx.Get()

	if !profile.In(buildProfiles...) {
		t.Skipf("%q could not run this test", profile)
	}

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
		imagePath := path.Join(env.TestDir, "container")

		// conditionally build a sandbox
		args := []string{}
		if tt.sandbox {
			args = []string{"--sandbox"}
		}
		args = append(args, imagePath, tt.buildSpec)

		env.RunSingularity(
			t,
			e2e.WithProfile(profile),
			e2e.WithCommand("build"),
			e2e.WithArgs(args...),
			e2e.PostRun(func(t *testing.T) {
				defer os.RemoveAll(imagePath)

				e2e.ImageVerify(t, env.CmdPath, imagePath)
			}),
			e2e.ExpectExit(0),
		)
	}
}

func buildLocalImage(ctx *e2e.TestContext) {
	t, env, profile := ctx.Get()

	if !profile.In(rootBuildProfiles...) {
		t.Skipf("%q could not run this test", profile)
	}

	e2e.EnsureImage(t, env)

	tmpdir, err := ioutil.TempDir(env.TestDir, "build-local-image.")
	if err != nil {
		t.Errorf("Cannot create temporary directory: %+v", err)
	}

	defer os.RemoveAll(tmpdir)

	liDefFile := e2e.PrepareDefFile(e2e.DefFileDetails{
		Bootstrap: "localimage",
		From:      env.ImagePath,
	})
	defer os.Remove(liDefFile)

	labels := make(map[string]string)
	labels["FOO"] = "bar"
	liLabelDefFile := e2e.PrepareDefFile(e2e.DefFileDetails{
		Bootstrap: "localimage",
		From:      env.ImagePath,
		Labels:    labels,
	})
	defer os.Remove(liLabelDefFile)

	sandboxImage := path.Join(tmpdir, "test-sandbox")

	args := []string{"--sandbox", sandboxImage, env.ImagePath}

	env.RunSingularity(
		t,
		e2e.WithProfile(profile),
		e2e.WithCommand("build"),
		e2e.WithArgs(args...),
		e2e.PostRun(func(t *testing.T) {
			e2e.ImageVerify(t, env.CmdPath, sandboxImage)
		}),
		e2e.ExpectExit(0),
	)

	localSandboxDefFile := e2e.PrepareDefFile(e2e.DefFileDetails{
		Bootstrap: "localimage",
		From:      sandboxImage,
		Labels:    labels,
	})
	defer os.Remove(localSandboxDefFile)

	tests := []struct {
		name      string
		buildSpec string
	}{
		{"SIFToSIF", env.ImagePath},
		{"SandboxToSIF", sandboxImage},
		{"LocalImage", liDefFile},
		{"LocalImageLabel", liLabelDefFile},
		{"LocalImageSandbox", localSandboxDefFile},
	}

	for i, tt := range tests {
		imagePath := filepath.Join(tmpdir, fmt.Sprintf("image-%d", i))
		args := []string{imagePath, tt.buildSpec}

		env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(profile),
			e2e.WithCommand("build"),
			e2e.WithArgs(args...),
			e2e.PostRun(func(t *testing.T) {
				e2e.ImageVerify(t, env.CmdPath, imagePath)
			}),
			e2e.ExpectExit(0),
		)
	}
}

func badPath(ctx *e2e.TestContext) {
	t, env, profile := ctx.Get()

	if !profile.In(rootBuildProfiles...) {
		t.Skipf("%q could not run this test", profile)
	}

	imagePath := path.Join(env.TestDir, "container")
	args := []string{imagePath, dumbPath}

	env.RunSingularity(
		t,
		e2e.WithProfile(profile),
		e2e.WithCommand("build"),
		e2e.WithArgs(args...),
		e2e.ExpectExit(255),
	)
}

func buildMultiStageDefinition(ctx *e2e.TestContext) {
	t, env, profile := ctx.Get()

	if !profile.In(rootBuildProfiles...) {
		t.Skipf("%q could not run this test", profile)
	}

	tmpfile, err := e2e.WriteTempFile(env.TestDir, "testFile-", testFileContent)
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
		imagePath := path.Join(env.TestDir, "container")
		defFile := e2e.PrepareMultiStageDefFile(tt.dfd)

		args := []string{}
		if tt.force {
			args = append([]string{"--force"}, args...)
		}
		if tt.sandbox {
			args = append([]string{"--sandbox"}, args...)
		}
		args = append(args, imagePath, defFile)

		env.RunSingularity(
			t,
			e2e.WithProfile(profile),
			e2e.WithCommand("build"),
			e2e.WithArgs(args...),
			e2e.PostRun(func(t *testing.T) {
				defer os.Remove(defFile)
				defer os.RemoveAll(imagePath)

				e2e.DefinitionImageVerify(t, env.CmdPath, imagePath, tt.correct)
			}),
			e2e.ExpectExit(0),
		)
	}
}

func buildDefinition(ctx *e2e.TestContext) {
	t, env, profile := ctx.Get()

	if !profile.In(rootBuildProfiles...) {
		t.Skipf("%q could not run this test", profile)
	}

	defDir, err := fs.MakeTmpDir(env.TestDir, "definition-", 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(defDir)

	tmpfile, err := e2e.WriteTempFile(env.TestDir, "testFile-", testFileContent)
	if err != nil {
		t.Fatal(err)
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
				filepath.Join(defDir, "PreFile1"),
			},
		}},
		{"Setup", false, true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Setup: []string{
				filepath.Join(defDir, "SetupFile1"),
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
		imagePath := path.Join(env.TestDir, "container")
		defFile := e2e.PrepareDefFile(tt.dfd)

		args := []string{}
		if tt.force {
			args = append([]string{"--force"}, args...)
		}
		if tt.sandbox {
			args = append([]string{"--sandbox"}, args...)
		}
		args = append(args, imagePath, defFile)

		env.RunSingularity(
			t,
			e2e.WithProfile(profile),
			e2e.WithCommand("build"),
			e2e.WithArgs(args...),
			e2e.PostRun(func(t *testing.T) {
				defer os.Remove(defFile)
				defer os.RemoveAll(imagePath)

				e2e.DefinitionImageVerify(t, env.CmdPath, imagePath, tt.dfd)
			}),
			e2e.ExpectExit(0),
		)
	}
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	return func(t *testing.T) {
		tests := map[string]func(*e2e.TestContext){
			"From":               buildFrom,                 // builds from definition file and URI
			"FromLocalImage":     buildLocalImage,           // build and image from an existing image
			"NonPrivilegedBuild": nonPrivilegedBuild,        // build sifs with all profiles
			"BadPath":            badPath,                   // try to build from a non existent path
			"Definition":         buildDefinition,           // builds from definition template
			"MultiStage":         buildMultiStageDefinition, // multistage build from definition templates
		}

		for _, profile := range e2e.Profiles {
			t.Run(profile.Name(), func(t *testing.T) {
				profile.Require(t)

				for name, fn := range tests {
					t.Run(name, func(t *testing.T) {
						ctx := e2e.NewTestContext(t, env, profile)
						fn(ctx)
					})
				}
			})
		}
	}
}
