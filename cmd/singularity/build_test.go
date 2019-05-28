// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// This file is been deprecated and will disappear on with version 3.3
// of singularity. The functionality has been moved to e2e/imgbuild/imgbuild.go

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/test"
)

var testFileContent = "Test file content\n"

func imageVerify(t *testing.T, imagePath string, labels bool) {
	type testSpec struct {
		name          string
		execArgs      []string
		expectSuccess bool
	}
	tests := []testSpec{
		{"False", []string{"false"}, false},
		{"RunScript", []string{"test", "-f", "/.singularity.d/runscript"}, true},
		{"OneBase", []string{"test", "-f", "/.singularity.d/env/01-base.sh"}, true},
		{"ActionsShell", []string{"test", "-f", "/.singularity.d/actions/shell"}, true},
		{"ActionsExec", []string{"test", "-f", "/.singularity.d/actions/exec"}, true},
		{"ActionsRun", []string{"test", "-f", "/.singularity.d/actions/run"}, true},
		{"Environment", []string{"test", "-L", "/environment"}, true},
		{"Singularity", []string{"test", "-L", "/singularity"}, true},
	}
	if labels && *runDisabled { // TODO
		tests = append(tests, testSpec{"Labels", []string{"test", "-f", "/.singularity.d/labels.json"}, true})
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			_, stderr, exitCode, err := imageExec(t, "exec", opts{}, imagePath, tt.execArgs)
			if tt.expectSuccess && (exitCode != 0) {
				t.Log(stderr)
				t.Fatalf("unexpected failure running '%v': %v", strings.Join(tt.execArgs, " "), err)
			} else if !tt.expectSuccess && (exitCode != 1) {
				t.Log(stderr)
				t.Fatalf("unexpected success running '%v'", strings.Join(tt.execArgs, " "))
			}
		}))
	}
}

type buildOpts struct {
	force   bool
	sandbox bool
	env     []string
}

func imageBuild(opts buildOpts, imagePath, buildSpec string) ([]byte, error) {
	var argv []string
	argv = append(argv, "build")
	if opts.force {
		argv = append(argv, "--force")
	}
	if opts.sandbox {
		argv = append(argv, "--sandbox")
	}
	argv = append(argv, imagePath, buildSpec)

	cmd := exec.Command(cmdPath, argv...)
	cmd.Env = opts.env

	return cmd.CombinedOutput()
}

func TestBuild(t *testing.T) {
	tests := []struct {
		name       string
		dependency string
		buildSpec  string
		sandbox    bool
	}{
		{"BusyBox", "", "../../examples/busybox/Singularity", false},
		{"BusyBoxSandbox", "", "../../examples/busybox/Singularity", true},
		{"Debootstrap", "debootstrap", "../../examples/debian/Singularity", true},
		{"DockerURI", "", "docker://busybox", true},
		{"DockerDefFile", "", "../../examples/docker/Singularity", true},
		{"SHubURI", "", "shub://GodloveD/busybox", true},
		{"SHubDefFile", "", "../../examples/shub/Singularity", true},
		{"LibraryDefFile", "", "../../examples/library/Singularity", true},
		{"Yum", "yum", "../../examples/centos/Singularity", true},
		{"Zypper", "zypper", "../../examples/opensuse/Singularity", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithPrivilege(func(t *testing.T) {
			if tt.dependency != "" {
				if _, err := exec.LookPath(tt.dependency); err != nil {
					t.Skipf("%v not found in path", tt.dependency)
				}
			}

			cacheDir := test.SetCacheDir(t, "")
			defer test.CleanCacheDir(t, cacheDir)

			err := os.Setenv(cache.DirEnv, cacheDir)
			if err != nil {
				t.Fatalf("failed to set %s environment variable: %s", cache.DirEnv, err)
			}

			opts := buildOpts{
				sandbox: tt.sandbox,
			}

			imagePath := path.Join(testDir, "container")
			defer os.RemoveAll(imagePath)

			if b, err := imageBuild(opts, imagePath, tt.buildSpec); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
			imageVerify(t, imagePath, false)
		}))
	}
}

func TestMultipleBuilds(t *testing.T) {
	imagePath1 := path.Join(testDir, "container1")
	imagePath2 := path.Join(testDir, "container2")
	imagePath3 := path.Join(testDir, "container3")

	liDefFile := prepareDefFile(DefFileDetail{
		Bootstrap: "localimage",
		From:      imagePath1,
	})
	defer os.Remove(liDefFile)

	labels := make(map[string]string)
	labels["FOO"] = "bar"
	liLabelDefFile := prepareDefFile(DefFileDetail{
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
			{"BusyBox", imagePath1, "../../examples/busybox/Singularity", false, false, false},
			{"SIF", imagePath2, imagePath1, false, false, false},
		}},
		{"SandboxToSIF", []testSpec{
			{"BusyBoxSandbox", imagePath1, "../../examples/busybox/Singularity", false, true, false},
			{"SIF", imagePath2, imagePath1, false, false, false},
		}},
		{"LocalImage", []testSpec{
			{"BusyBox", imagePath1, "../../examples/busybox/Singularity", false, false, false},
			{"LocalImage", imagePath2, liDefFile, false, false, false},
			{"LocalImageLabel", imagePath3, liLabelDefFile, false, false, true},
		}},
		{"LocalImageSandbox", []testSpec{
			{"BusyBoxSandbox", imagePath2, "../../examples/busybox/Singularity", true, true, false},
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
					opts := buildOpts{
						force:   ts.force,
						sandbox: ts.sandbox,
					}

					cacheDir := test.SetCacheDir(t, "")
					defer test.CleanCacheDir(t, cacheDir)

					err := os.Setenv(cache.DirEnv, cacheDir)
					if err != nil {
						t.Fatalf("cannot set %s environment variable: %s", cache.DirEnv, err)
					}

					if b, err := imageBuild(opts, ts.imagePath, ts.buildSpec); err != nil {
						t.Log(string(b))
						t.Fatalf("unexpected failure: %v", err)
					}
					imageVerify(t, ts.imagePath, ts.labels)
				}))
			}
		}))
	}
}

func TestBadPath(t *testing.T) {
	test.EnsurePrivilege(t)

	imagePath := path.Join(testDir, "container")
	defer os.RemoveAll(imagePath)

	cacheDir := test.SetCacheDir(t, "")
	defer test.CleanCacheDir(t, cacheDir)

	err := os.Setenv(cache.DirEnv, cacheDir)
	if err != nil {
		t.Fatalf("failed to set %s environment variable: %s", cache.DirEnv, err)
	}

	if b, err := imageBuild(buildOpts{}, imagePath, "/some/dumb/path"); err == nil {
		t.Log(string(b))
		t.Fatal("unexpected success")
	}
}

func TestMultiStageDefinition(t *testing.T) {
	tmpfile, err := ioutil.TempFile(testDir, "testFile-")
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
		dfd     []DefFileDetail
		correct DefFileDetail // a bit hacky, but this allows us to check final image for correct artifacts
	}{
		// Simple copy from stage one to final stage
		{"FileCopySimple", false, true, []DefFileDetail{
			{
				Bootstrap: "docker",
				From:      "alpine:latest",
				Stage:     "one",
				Files: []FilePair{
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
				FilesFrom: []FileSection{
					{
						"one",
						[]FilePair{
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
			DefFileDetail{
				Files: []FilePair{
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
			[]DefFileDetail{
				{
					Bootstrap: "docker",
					From:      "alpine:latest",
					Stage:     "one",
					Files: []FilePair{
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
					Files: []FilePair{
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
					FilesFrom: []FileSection{
						{
							"one",
							[]FilePair{
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
							"two",
							[]FilePair{
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
					FilesFrom: []FileSection{
						{
							"three",
							[]FilePair{
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
			DefFileDetail{
				Files: []FilePair{
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

			defFile := prepareMultipleDefFiles(tt.dfd)
			defer os.Remove(defFile)

			opts := buildOpts{
				sandbox: tt.sandbox,
			}

			imagePath := path.Join(testDir, "container")
			defer os.RemoveAll(imagePath)

			cacheDir := test.SetCacheDir(t, "")
			defer test.CleanCacheDir(t, cacheDir)

			err := os.Setenv(cache.DirEnv, cacheDir)
			if err != nil {
				t.Fatalf("failed to set %s environment variable: %s", cache.DirEnv, err)
			}

			if b, err := imageBuild(opts, imagePath, defFile); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}

			definitionImageVerify(t, imagePath, tt.correct)
		}))
	}

}

func TestBuildDefinition(t *testing.T) {

	tmpfile, err := ioutil.TempFile(testDir, "testFile-")
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
		dfd     DefFileDetail
	}{
		{"Empty", false, true, DefFileDetail{
			Bootstrap: "docker",
			From:      "alpine:latest",
		}},
		{"Help", false, true, DefFileDetail{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Help: []string{
				"help info line 1",
				"help info line 2",
				"help info line 3",
			},
		}},
		{"Files", false, true, DefFileDetail{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Files: []FilePair{
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
		{"Test", false, true, DefFileDetail{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Test: []string{
				"echo testscript line 1",
				"echo testscript line 2",
				"echo testscript line 3",
			},
		}},
		{"Startscript", false, true, DefFileDetail{
			Bootstrap: "docker",
			From:      "alpine:latest",
			StartScript: []string{
				"echo startscript line 1",
				"echo startscript line 2",
				"echo startscript line 3",
			},
		}},
		{"Runscript", false, true, DefFileDetail{
			Bootstrap: "docker",
			From:      "alpine:latest",
			RunScript: []string{
				"echo runscript line 1",
				"echo runscript line 2",
				"echo runscript line 3",
			},
		}},
		{"Env", false, true, DefFileDetail{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Env: []string{
				"testvar1=one",
				"testvar2=two",
				"testvar3=three",
			},
		}},
		{"Labels", false, true, DefFileDetail{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Labels: map[string]string{
				"customLabel1": "one",
				"customLabel2": "two",
				"customLabel3": "three",
			},
		}},
		{"Pre", false, true, DefFileDetail{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Pre: []string{
				filepath.Join(testDir, "PreFile1"),
			},
		}},
		{"Setup", false, true, DefFileDetail{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Setup: []string{
				filepath.Join(testDir, "SetupFile1"),
			},
		}},
		{"Post", false, true, DefFileDetail{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Post: []string{
				"PostFile1",
			},
		}},
		{"AppHelp", false, true, DefFileDetail{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []AppDetail{
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
		{"AppEnv", false, true, DefFileDetail{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []AppDetail{
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
		{"AppLabels", false, true, DefFileDetail{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []AppDetail{
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
		{"AppFiles", false, true, DefFileDetail{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []AppDetail{
				{
					Name: "foo",
					Files: []FilePair{
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
					Files: []FilePair{
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
		{"AppInstall", false, true, DefFileDetail{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []AppDetail{
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
		{"AppRun", false, true, DefFileDetail{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []AppDetail{
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
		{"AppTest", false, true, DefFileDetail{
			Bootstrap: "docker",
			From:      "alpine:latest",
			Apps: []AppDetail{
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

			defFile := prepareDefFile(tt.dfd)
			defer os.Remove(defFile)

			opts := buildOpts{
				sandbox: tt.sandbox,
			}

			imagePath := path.Join(testDir, "container")
			defer os.RemoveAll(imagePath)

			cacheDir := test.SetCacheDir(t, "")
			defer test.CleanCacheDir(t, cacheDir)

			err := os.Setenv(cache.DirEnv, cacheDir)
			if err != nil {
				t.Fatalf("failed to set %s environment variable: %s", cache.DirEnv, err)
			}

			if b, err := imageBuild(opts, imagePath, defFile); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
			definitionImageVerify(t, imagePath, tt.dfd)
		}))
	}
}

func definitionImageVerify(t *testing.T, imagePath string, dfd DefFileDetail) {
	if dfd.Help != nil {
		helpPath := filepath.Join(imagePath, `/.singularity.d/runscript.help`)
		if !fileExists(t, helpPath) {
			t.Fatalf("unexpected failure: Script %v does not exist in container", helpPath)
		}

		if err := verifyHelp(t, helpPath, dfd.Help); err != nil {
			t.Fatalf("unexpected failure: help message: %v", err)
		}
	}

	if dfd.Env != nil {
		if err := verifyEnv(t, imagePath, dfd.Env, nil); err != nil {
			t.Fatalf("unexpected failure: Env in container is incorrect: %v", err)
		}
	}

	// always run this since we should at least have default build labels
	if err := verifyLabels(t, imagePath, dfd.Labels); err != nil {
		t.Fatalf("unexpected failure: Labels in the container are incorrect: %v", err)
	}

	// verify %files section works correctly
	for _, p := range dfd.Files {
		var file string
		if p.Dst == "" {
			file = p.Src
		} else {
			file = p.Dst
		}

		if !fileExists(t, filepath.Join(imagePath, file)) {
			t.Fatalf("unexpected failure: File %v does not exist in container", file)
		}

		if err := verifyFile(t, p.Src, filepath.Join(imagePath, file)); err != nil {
			t.Fatalf("unexpected failure: File %v: %v", file, err)
		}
	}

	if dfd.RunScript != nil {
		scriptPath := filepath.Join(imagePath, `/.singularity.d/runscript`)
		if !fileExists(t, scriptPath) {
			t.Fatalf("unexpected failure: Script %v does not exist in container", scriptPath)
		}

		if err := verifyScript(t, scriptPath, dfd.RunScript); err != nil {
			t.Fatalf("unexpected failure: runscript: %v", err)
		}
	}

	if dfd.StartScript != nil {
		scriptPath := filepath.Join(imagePath, `/.singularity.d/startscript`)
		if !fileExists(t, scriptPath) {
			t.Fatalf("unexpected failure: Script %v does not exist in container", scriptPath)
		}

		if err := verifyScript(t, scriptPath, dfd.StartScript); err != nil {
			t.Fatalf("unexpected failure: startscript: %v", err)
		}
	}

	if dfd.Test != nil {
		scriptPath := filepath.Join(imagePath, `/.singularity.d/test`)
		if !fileExists(t, scriptPath) {
			t.Fatalf("unexpected failure: Script %v does not exist in container", scriptPath)
		}

		if err := verifyScript(t, scriptPath, dfd.Test); err != nil {
			t.Fatalf("unexpected failure: test script: %v", err)
		}
	}

	for _, file := range dfd.Pre {
		if !fileExists(t, file) {
			t.Fatalf("unexpected failure: %%Pre generated file %v does not exist on host", file)
		}
	}

	for _, file := range dfd.Setup {
		if !fileExists(t, file) {
			t.Fatalf("unexpected failure: %%Setup generated file %v does not exist on host", file)
		}
	}

	for _, file := range dfd.Post {
		if !fileExists(t, filepath.Join(imagePath, file)) {
			t.Fatalf("unexpected failure: %%Post generated file %v does not exist in container", file)
		}
	}

	// Verify any apps
	for _, app := range dfd.Apps {
		// %apphelp
		if app.Help != nil {
			helpPath := filepath.Join(imagePath, `/scif/apps/`, app.Name, `/scif/runscript.help`)
			if !fileExists(t, helpPath) {
				t.Fatalf("unexpected failure in app %v: Script %v does not exist in app", app.Name, helpPath)
			}

			if err := verifyHelp(t, helpPath, app.Help); err != nil {
				t.Fatalf("unexpected failure in app %v: app help message: %v", app.Name, err)
			}
		}

		// %appenv
		if app.Env != nil {
			if err := verifyEnv(t, imagePath, app.Env, []string{"--app", app.Name}); err != nil {
				t.Fatalf("unexpected failure in app %v: Env in app is incorrect: %v", app.Name, err)
			}
		}

		// %applabels
		if app.Labels != nil {
			if err := verifyAppLabels(t, imagePath, app.Name, app.Labels); err != nil {
				t.Fatalf("unexpected failure in app %v: Labels in app are incorrect: %v", app.Name, err)
			}
		}

		// %appfiles
		for _, p := range app.Files {
			var file string
			if p.Src == "" {
				file = p.Src
			} else {
				file = p.Dst
			}

			if !fileExists(t, filepath.Join(imagePath, "/scif/apps/", app.Name, file)) {
				t.Fatalf("unexpected failure in app %v: File %v does not exist in app", app.Name, file)
			}

			if err := verifyFile(t, p.Src, filepath.Join(imagePath, "/scif/apps/", app.Name, file)); err != nil {
				t.Fatalf("unexpected failure in app %v: File %v: %v", app.Name, file, err)
			}
		}

		// %appInstall
		for _, file := range app.Install {
			if !fileExists(t, filepath.Join(imagePath, "/scif/apps/", app.Name, file)) {
				t.Fatalf("unexpected failure in app %v: %%Install generated file %v does not exist in container", app.Name, file)
			}
		}

		// %appRun
		if app.Run != nil {
			scriptPath := filepath.Join(imagePath, "/scif/apps/", app.Name, "scif/runscript")
			if !fileExists(t, scriptPath) {
				t.Fatalf("unexpected failure in app %v: Script %v does not exist in app", app.Name, scriptPath)
			}

			if err := verifyScript(t, scriptPath, app.Run); err != nil {
				t.Fatalf("unexpected failure in app %v: runscript: %v", app.Name, err)
			}
		}

		// %appTest
		if app.Test != nil {
			scriptPath := filepath.Join(imagePath, "/scif/apps/", app.Name, "scif/test")
			if !fileExists(t, scriptPath) {
				t.Fatalf("unexpected failure in app %v: Script %v does not exist in app", app.Name, scriptPath)
			}

			if err := verifyScript(t, scriptPath, app.Test); err != nil {
				t.Fatalf("unexpected failure in app %v: test script: %v", app.Name, err)
			}
		}
	}

}

func fileExists(t *testing.T, path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	} else if err != nil {
		t.Fatalf("While stating file: %v", err)
	}

	return true
}

func verifyFile(t *testing.T, original, copy string) error {
	ofi, err := os.Stat(original)
	if err != nil {
		t.Fatalf("While getting file info: %v", err)
	}

	cfi, err := os.Stat(copy)
	if err != nil {
		t.Fatalf("While getting file info: %v", err)
	}

	if ofi.Size() != cfi.Size() {
		return fmt.Errorf("Incorrect file sizes. Original: %v, Copy: %v", ofi.Size(), cfi.Size())
	}

	if ofi.Mode() != cfi.Mode() {
		return fmt.Errorf("Incorrect file modes. Original: %v, Copy: %v", ofi.Mode(), cfi.Mode())
	}

	o, err := ioutil.ReadFile(original)
	if err != nil {
		t.Fatalf("While reading file: %v", err)
	}

	c, err := ioutil.ReadFile(copy)
	if err != nil {
		t.Fatalf("While reading file: %v", err)
	}

	if !bytes.Equal(o, c) {
		return fmt.Errorf("Incorrect file content")
	}

	return nil
}

func verifyHelp(t *testing.T, fileName string, contents []string) error {
	fi, err := os.Stat(fileName)
	if err != nil {
		t.Fatalf("While getting file info: %v", err)
	}

	// do perm check
	if fi.Mode().Perm() != 0644 {
		return fmt.Errorf("Incorrect help script perms: %v", fi.Mode().Perm())
	}

	s, err := ioutil.ReadFile(fileName)
	if err != nil {
		t.Fatalf("While reading file: %v", err)
	}

	helpScript := string(s)
	for _, c := range contents {
		if !strings.Contains(helpScript, c) {
			return fmt.Errorf("Missing help script content")
		}
	}

	return nil
}

func verifyScript(t *testing.T, fileName string, contents []string) error {
	fi, err := os.Stat(fileName)
	if err != nil {
		t.Fatalf("While getting file info: %v", err)
	}

	// do perm check
	if fi.Mode().Perm() != 0755 {
		return fmt.Errorf("Incorrect script perms: %v", fi.Mode().Perm())
	}

	s, err := ioutil.ReadFile(fileName)
	if err != nil {
		t.Fatalf("While reading file: %v", err)
	}

	script := string(s)
	for _, c := range contents {
		if !strings.Contains(script, c) {
			return fmt.Errorf("Missing script content")
		}
	}

	return nil
}

func verifyEnv(t *testing.T, imagePath string, env []string, flags []string) error {
	args := []string{"exec"}
	if flags != nil {
		args = append(args, flags...)
	}
	args = append(args, imagePath, "env")

	cmd := exec.Command(cmdPath, args...)
	b, err := cmd.CombinedOutput()

	out := string(b)

	if err != nil {
		t.Fatalf("Error running command: %v", err)
	}

	for _, e := range env {
		if !strings.Contains(out, e) {
			return fmt.Errorf("Environment is missing: %v", e)
		}
	}

	return nil
}

func verifyLabels(t *testing.T, imagePath string, labels map[string]string) error {
	var fileLabels map[string]string

	b, err := ioutil.ReadFile(filepath.Join(imagePath, "/.singularity.d/labels.json"))
	if err != nil {
		t.Fatalf("While reading file: %v", err)
	}

	if err := json.Unmarshal(b, &fileLabels); err != nil {
		t.Fatalf("While unmarshaling labels.json into map: %v", err)
	}

	for k, v := range labels {
		if l, ok := fileLabels[k]; !ok || v != l {
			return fmt.Errorf("Missing label: %v:%v", k, v)
		}
	}

	//check default labels that are always generated
	defaultLabels := []string{
		"org.label-schema.schema-version",
		"org.label-schema.build-date",
		"org.label-schema.usage.singularity.version",
	}

	for _, l := range defaultLabels {
		if _, ok := fileLabels[l]; !ok {
			return fmt.Errorf("Missing label: %v", l)
		}
	}

	return nil
}

func verifyAppLabels(t *testing.T, imagePath, appName string, labels map[string]string) error {
	var fileLabels map[string]string

	b, err := ioutil.ReadFile(filepath.Join(imagePath, "/scif/apps/", appName, "/scif/labels.json"))
	if err != nil {
		t.Fatalf("While reading file: %v", err)
	}

	if err := json.Unmarshal(b, &fileLabels); err != nil {
		t.Fatalf("While unmarshaling labels.json into map: %v", err)
	}

	for k, v := range labels {
		if l, ok := fileLabels[k]; !ok || v != l {
			return fmt.Errorf("Missing label: %v:%v", k, v)
		}
	}

	return nil
}
