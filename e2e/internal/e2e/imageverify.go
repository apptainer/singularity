// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

// ImageVerify checks for an image integrity
func ImageVerify(t *testing.T, cmdPath string, imagePath string, labels bool, runDisabled bool) {
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
	if labels && runDisabled { // TODO
		tests = append(tests, testSpec{"Labels", []string{"test", "-f", "/.singularity.d/labels.json"}, true})
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			_, stderr, exitCode, err := ImageExec(t, cmdPath, "exec", ExecOpts{}, imagePath, tt.execArgs)
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

// DefinitionImageVerify checks for image correctness based off off supplied DefFileDetail
func DefinitionImageVerify(t *testing.T, cmdPath, imagePath string, dfd DefFileDetails) {
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
		if err := verifyEnv(t, cmdPath, imagePath, dfd.Env, nil); err != nil {
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

		fmt.Println(p.Src, filepath.Join(imagePath, file))
		//os.Exit(1)
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
			if err := verifyEnv(t, cmdPath, imagePath, app.Env, []string{"--app", app.Name}); err != nil {
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

func verifyEnv(t *testing.T, cmdPath, imagePath string, env []string, flags []string) error {
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
