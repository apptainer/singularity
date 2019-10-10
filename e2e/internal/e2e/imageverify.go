// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/sylabs/singularity/internal/pkg/test/tool/exec"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

// ImageVerify checks for an image integrity.
func (env TestEnv) ImageVerify(t *testing.T, imagePath string, profile Profile) {
	tt := []struct {
		name string
		argv []string
		exit int
	}{
		{
			name: "False",
			argv: []string{imagePath, "false"},
			exit: 1,
		},
		{
			name: "RunScript",
			argv: []string{imagePath, "test", "-f", "/.singularity.d/runscript"},
			exit: 0,
		},
		{
			name: "OneBase",
			argv: []string{imagePath, "test", "-f", "/.singularity.d/env/01-base.sh"},
			exit: 0,
		},
		{
			name: "ActionsShell",
			argv: []string{imagePath, "test", "-f", "/.singularity.d/actions/shell"},
			exit: 0,
		},
		{
			name: "ActionsExec",
			argv: []string{imagePath, "test", "-f", "/.singularity.d/actions/exec"},
			exit: 0,
		},
		{
			name: "ActionsRun",
			argv: []string{imagePath, "test", "-f", "/.singularity.d/actions/run"},
			exit: 0,
		},
		{
			name: "Environment",
			argv: []string{imagePath, "test", "-L", "/environment"},
			exit: 0,
		},
		{
			name: "Singularity",
			argv: []string{imagePath, "test", "-L", "/singularity"},
			exit: 0,
		},
	}

	for _, tc := range tt {
		env.RunSingularity(
			t,
			AsSubtest(tc.name),
			WithProfile(profile),
			WithCommand("exec"),
			WithArgs(tc.argv...),
			ExpectExit(tc.exit),
		)
	}

	tests := []struct {
		name      string
		jsonPath  []string
		expectOut string
	}{
		{
			name:      "LabelCheckType",
			jsonPath:  []string{"type"},
			expectOut: "container",
		},
		{
			name:      "LabelCheckSchemaVersion",
			jsonPath:  []string{"data", "attributes", "labels", "org.label-schema.schema-version"},
			expectOut: "1.0",
		},
	}

	// Verify the label partition
	for _, tt := range tests {
		verifyOutput := func(t *testing.T, r *SingularityCmdResult) {
			jsonOut, err := jsonparser.GetString(r.Stdout, tt.jsonPath...)
			if err != nil {
				t.Fatalf("unable to get expected output from json: %v", err)
			}
			if jsonOut != tt.expectOut {
				t.Fatalf("unexpected failure: got: '%s', expecting: '%s'", jsonOut, tt.expectOut)
			}
		}

		env.RunSingularity(
			t,
			AsSubtest(tt.name),
			WithProfile(UserProfile),
			WithCommand("inspect"),
			WithArgs([]string{"--json", "--labels", imagePath}...),
			ExpectExit(0, verifyOutput),
		)
	}
}

// DefinitionImageVerify checks for image correctness based off off supplied DefFileDetail
func DefinitionImageVerify(t *testing.T, cmdPath, imagePath string, dfd DefFileDetails) {
	if dfd.Help != nil {
		helpPath := filepath.Join(imagePath, `/.singularity.d/runscript.help`)
		if !fs.IsFile(helpPath) {
			t.Fatalf("unexpected failure: Script %v does not exist in container", helpPath)
		}

		if err := verifyHelp(t, helpPath, dfd.Help); err != nil {
			t.Fatalf("unexpected failure: help message: %v", err)
		}
	}

	if dfd.Env != nil {
		if err := verifyEnv(t, cmdPath, imagePath, dfd.Env, nil); err != nil {
			t.Fatalf("unexpected failure: Env in container is incorrect: %v", err)
		}
	}

	// verify %files section works correctly
	for _, p := range dfd.Files {
		var file string
		if p.Dst == "" {
			file = p.Src
		} else {
			file = p.Dst
		}

		if !fs.IsFile(filepath.Join(imagePath, file)) {
			t.Fatalf("unexpected failure: File %v does not exist in container", file)
		}

		if err := verifyFile(t, p.Src, filepath.Join(imagePath, file)); err != nil {
			t.Fatalf("unexpected failure: File %v: %v", file, err)
		}
	}

	if dfd.RunScript != nil {
		scriptPath := filepath.Join(imagePath, `/.singularity.d/runscript`)
		if !fs.IsFile(scriptPath) {
			t.Fatalf("unexpected failure: Script %v does not exist in container", scriptPath)
		}

		if err := verifyScript(t, scriptPath, dfd.RunScript); err != nil {
			t.Fatalf("unexpected failure: runscript: %v", err)
		}
	}

	if dfd.StartScript != nil {
		scriptPath := filepath.Join(imagePath, `/.singularity.d/startscript`)
		if !fs.IsFile(scriptPath) {
			t.Fatalf("unexpected failure: Script %v does not exist in container", scriptPath)
		}

		if err := verifyScript(t, scriptPath, dfd.StartScript); err != nil {
			t.Fatalf("unexpected failure: startscript: %v", err)
		}
	}

	if dfd.Test != nil {
		scriptPath := filepath.Join(imagePath, `/.singularity.d/test`)
		if !fs.IsFile(scriptPath) {
			t.Fatalf("unexpected failure: Script %v does not exist in container", scriptPath)
		}

		if err := verifyScript(t, scriptPath, dfd.Test); err != nil {
			t.Fatalf("unexpected failure: test script: %v", err)
		}
	}

	for _, file := range dfd.Pre {
		if !fs.IsFile(file) {
			t.Fatalf("unexpected failure: %%Pre generated file %v does not exist on host", file)
		}
		if err := os.Remove(file); err != nil {
			t.Fatalf("could not remove %s: %s", file, err)
		}
	}

	for _, file := range dfd.Setup {
		if !fs.IsFile(file) {
			t.Fatalf("unexpected failure: %%Setup generated file %v does not exist on host", file)
		}
		if err := os.Remove(file); err != nil {
			t.Fatalf("could not remove %s: %s", file, err)
		}
	}

	for _, file := range dfd.Post {
		if !fs.IsFile(filepath.Join(imagePath, file)) {
			t.Fatalf("unexpected failure: %%Post generated file %v does not exist in container", file)
		}
	}

	// Verify any apps
	for _, app := range dfd.Apps {
		// %apphelp
		if app.Help != nil {
			helpPath := filepath.Join(imagePath, `/scif/apps/`, app.Name, `/scif/runscript.help`)
			if !fs.IsFile(helpPath) {
				t.Fatalf("unexpected failure in app %v: Script %v does not exist in app", app.Name, helpPath)
			}

			if err := verifyHelp(t, helpPath, app.Help); err != nil {
				t.Fatalf("unexpected failure in app %v: app help message: %v", app.Name, err)
			}
		}

		// %appenv
		if app.Env != nil {
			if err := verifyEnv(t, cmdPath, imagePath, app.Env, []string{"--app", app.Name}); err != nil {
				t.Fatalf("unexpected failure in app %v: Env in app is incorrect: %v", app.Name, err)
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

			if !fs.IsFile(filepath.Join(imagePath, "/scif/apps/", app.Name, file)) {
				t.Fatalf("unexpected failure in app %v: File %v does not exist in app", app.Name, file)
			}

			if err := verifyFile(t, p.Src, filepath.Join(imagePath, "/scif/apps/", app.Name, file)); err != nil {
				t.Fatalf("unexpected failure in app %v: File %v: %v", app.Name, file, err)
			}
		}

		// %appInstall
		for _, file := range app.Install {
			if !fs.IsFile(filepath.Join(imagePath, "/scif/apps/", app.Name, file)) {
				t.Fatalf("unexpected failure in app %v: %%Install generated file %v does not exist in container", app.Name, file)
			}
		}

		// %appRun
		if app.Run != nil {
			scriptPath := filepath.Join(imagePath, "/scif/apps/", app.Name, "scif/runscript")
			if !fs.IsFile(scriptPath) {
				t.Fatalf("unexpected failure in app %v: Script %v does not exist in app", app.Name, scriptPath)
			}

			if err := verifyScript(t, scriptPath, app.Run); err != nil {
				t.Fatalf("unexpected failure in app %v: runscript: %v", app.Name, err)
			}
		}

		// %appTest
		if app.Test != nil {
			scriptPath := filepath.Join(imagePath, "/scif/apps/", app.Name, "scif/test")
			if !fs.IsFile(scriptPath) {
				t.Fatalf("unexpected failure in app %v: Script %v does not exist in app", app.Name, scriptPath)
			}

			if err := verifyScript(t, scriptPath, app.Test); err != nil {
				t.Fatalf("unexpected failure in app %v: test script: %v", app.Name, err)
			}
		}
	}

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

func verifyEnv(t *testing.T, cmdPath, imagePath string, env []string, flags []string) error {
	args := []string{"exec"}
	if flags != nil {
		args = append(args, flags...)
	}
	args = append(args, imagePath, "env")

	cmd := exec.Command(cmdPath, args...)
	res := cmd.Run(t)

	if res.Error != nil {
		t.Fatalf("Error running command.\n%s", res)
	}

	out := res.Stdout()

	for _, e := range env {
		if !strings.Contains(out, e) {
			return fmt.Errorf("Environment is missing: %v", e)
		}
	}

	return nil
}
