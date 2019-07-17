// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

// Package apps [apps-plugin] provides the functions which are necessary for adding SCI-F apps support
// to Singularity 3.0.0. In 3.1.0+, this package will be able to be built standalone as
// a plugin so it will be maintainable separately from the core Singularity functionality
package apps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
)

const name = "singularity_apps"

const (
	sectionInstall = "appinstall"
	sectionFiles   = "appfiles"
	sectionEnv     = "appenv"
	sectionTest    = "apptest"
	sectionHelp    = "apphelp"
	sectionRun     = "apprun"
	sectionLabels  = "applabels"
)

var (
	sections = map[string]bool{
		sectionInstall: true,
		sectionFiles:   true,
		sectionEnv:     true,
		sectionTest:    true,
		sectionHelp:    true,
		sectionRun:     true,
		sectionLabels:  true,
	}
)

const (
	globalEnv94Base = `## App Global Exports For: %[1]s
	
SCIF_APPDATA_%[1]s=/scif/data/%[1]s
SCIF_APPMETA_%[1]s=/scif/apps/%[1]s/scif
SCIF_APPROOT_%[1]s=/scif/apps/%[1]s
SCIF_APPBIN_%[1]s=/scif/apps/%[1]s/bin
SCIF_APPLIB_%[1]s=/scif/apps/%[1]s/lib

export SCIF_APPDATA_%[1]s SCIF_APPMETA_%[1]s SCIF_APPROOT_%[1]s SCIF_APPBIN_%[1]s SCIF_APPLIB_%[1]s
`

	globalEnv94AppEnv = `export SCIF_APPENV_%[1]s="/scif/apps/%[1]s/scif/env/90-environment.sh"
`
	globalEnv94AppLabels = `export SCIF_APPLABELS_%[1]s="/scif/apps/%[1]s/scif/labels.json"
`
	globalEnv94AppRun = `export SCIF_APPRUN_%[1]s="/scif/apps/%[1]s/scif/runscript"
`

	scifEnv01Base = `#!/bin/sh

SCIF_APPNAME=%[1]s
SCIF_APPROOT="/scif/apps/%[1]s"
SCIF_APPMETA="/scif/apps/%[1]s/scif"
SCIF_DATA="/scif/data"
SCIF_APPDATA="/scif/data/%[1]s"
SCIF_APPINPUT="/scif/data/%[1]s/input"
SCIF_APPOUTPUT="/scif/data/%[1]s/output"
export SCIF_APPDATA SCIF_APPNAME SCIF_APPROOT SCIF_APPMETA SCIF_APPINPUT SCIF_APPOUTPUT SCIF_DATA
`

	scifRunscriptBase = `#!/bin/sh

%s
`
	scifTestBase = `#!/bin/sh

%s
`

	scifInstallBase = `
cd /
. %[1]s/scif/env/01-base.sh

cd %[1]s
%[2]s

cd /
`
)

// App stores the deffile sections of the app
type App struct {
	Name    string
	Install string
	Files   string
	Env     string
	Test    string
	Help    string
	Run     string
	Labels  string
}

// BuildApp is the type which the build system can use to build an app in a bundle
type BuildApp struct {
	Apps map[string]*App `json:"appsDefined"`
	sync.Mutex
}

// New returns a new BuildPlugin for the plugin registry to hold
func New() *BuildApp {
	return &BuildApp{
		Apps: make(map[string]*App),
	}

}

// Name returns this handler's name [singularity_apps]
func (pl *BuildApp) Name() string {
	return name
}

// HandleSection receives a string of each section from the deffile
func (pl *BuildApp) HandleSection(ident, section string) {
	name, sect := getAppAndSection(ident)
	if name == "" || sect == "" {
		return
	}

	pl.initApp(name)
	app := pl.Apps[name]

	switch sect {
	case sectionInstall:
		app.Install = section
	case sectionFiles:
		app.Files = section
	case sectionEnv:
		app.Env = section
	case sectionTest:
		app.Test = section
	case sectionHelp:
		app.Help = section
	case sectionRun:
		app.Run = section
	case sectionLabels:
		app.Labels = section
	default:
		return
	}
}

func (pl *BuildApp) initApp(name string) {
	pl.Lock()
	defer pl.Unlock()

	_, ok := pl.Apps[name]
	if !ok {
		pl.Apps[name] = &App{
			Name:    name,
			Install: "",
			Files:   "",
			Env:     "",
			Test:    "",
			Help:    "",
			Run:     "",
		}
	}
}

// getAppAndSection returns the app name and section name from the header of the section:
//     %SECTION APP ... returns APP, SECTION
func getAppAndSection(ident string) (appName string, sectionName string) {
	identSplit := strings.Split(ident, " ")

	if len(identSplit) < 2 {
		return "", ""
	}

	if _, ok := sections[identSplit[0]]; !ok {
		return "", ""
	}

	return identSplit[1], identSplit[0]
}

// HandleBundle is a hook where we can modify the bundle
func (pl *BuildApp) HandleBundle(b *types.Bundle) {
	if err := pl.createAllApps(b); err != nil {
		sylog.Fatalf("Unable to create apps: %s", err)
	}
}

func (pl *BuildApp) createAllApps(b *types.Bundle) error {
	globalEnv94 := ""

	for name, app := range pl.Apps {
		sylog.Debugf("Creating %s app in bundle", name)
		if err := createAppRoot(b, app); err != nil {
			return err
		}

		if err := writeEnvFile(b, app); err != nil {
			return err
		}

		if err := writeRunscriptFile(b, app); err != nil {
			return err
		}

		if err := writeTestFile(b, app); err != nil {
			return err
		}

		if err := writeHelpFile(b, app); err != nil {
			return err
		}

		if err := copyFiles(b, app); err != nil {
			return err
		}

		if err := writeLabels(b, app); err != nil {
			return err
		}

		globalEnv94 += globalAppEnv(b, app)
	}

	return ioutil.WriteFile(filepath.Join(b.Rootfs(), "/.singularity.d/env/94-appsbase.sh"), []byte(globalEnv94), 0755)
}

func createAppRoot(b *types.Bundle, a *App) error {
	if err := os.MkdirAll(appBase(b, a), 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(appBase(b, a), "/scif/"), 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(appBase(b, a), "/bin/"), 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(appBase(b, a), "/lib/"), 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(appBase(b, a), "/scif/env/"), 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(appData(b, a), "/input/"), 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(appData(b, a), "/output/"), 0755); err != nil {
		return err
	}

	return nil
}

// %appenv and 01-base.sh
func writeEnvFile(b *types.Bundle, a *App) error {
	content := fmt.Sprintf(scifEnv01Base, a.Name)
	if err := ioutil.WriteFile(filepath.Join(appMeta(b, a), "/env/01-base.sh"), []byte(content), 0755); err != nil {
		return err
	}

	if a.Env == "" {
		return nil
	}

	return ioutil.WriteFile(filepath.Join(appMeta(b, a), "/env/90-environment.sh"), []byte(a.Env), 0755)
}

func globalAppEnv(b *types.Bundle, a *App) string {
	content := fmt.Sprintf(globalEnv94Base, a.Name)

	if _, err := os.Stat(filepath.Join(appMeta(b, a), "/env/90-environment.sh")); err == nil {
		content += fmt.Sprintf(globalEnv94AppEnv, a.Name)
	}

	if _, err := os.Stat(filepath.Join(appMeta(b, a), "/labels.json")); err == nil {
		content += fmt.Sprintf(globalEnv94AppLabels, a.Name)
	}

	if _, err := os.Stat(filepath.Join(appMeta(b, a), "/runscript")); err == nil {
		content += fmt.Sprintf(globalEnv94AppRun, a.Name)
	}

	return content
}

// %apprun
func writeRunscriptFile(b *types.Bundle, a *App) error {
	if a.Run == "" {
		return nil
	}

	content := fmt.Sprintf(scifRunscriptBase, a.Run)
	return ioutil.WriteFile(filepath.Join(appMeta(b, a), "/runscript"), []byte(content), 0755)
}

// %apptest
func writeTestFile(b *types.Bundle, a *App) error {
	if a.Test == "" {
		return nil
	}

	content := fmt.Sprintf(scifTestBase, a.Test)
	return ioutil.WriteFile(filepath.Join(appMeta(b, a), "/test"), []byte(content), 0755)
}

// %apphelp
func writeHelpFile(b *types.Bundle, a *App) error {
	if a.Help == "" {
		return nil
	}

	return ioutil.WriteFile(filepath.Join(appMeta(b, a), "/runscript.help"), []byte(a.Help), 0644)
}

// %appfile
func copyFiles(b *types.Bundle, a *App) error {
	if a.Files == "" {
		return nil
	}

	appBase := filepath.Join(b.Rootfs(), "/scif/apps/", a.Name)
	for _, line := range strings.Split(a.Files, "\n") {

		// skip empty or comment lines
		if line = strings.TrimSpace(line); line == "" || strings.Index(line, "#") == 0 {
			continue
		}

		// trim any comments and whitespace
		trimLine := strings.Split(strings.TrimSpace(line), "#")[0]
		splitLine := strings.SplitN(strings.TrimSpace(trimLine), " ", 2)

		// copy to dst of same name in app if no dst is specified
		var src, dst string
		if len(splitLine) < 2 {
			src = splitLine[0]
			dst = splitLine[0]
		} else {
			src = splitLine[0]
			dst = splitLine[1]
		}

		if err := copy(src, filepath.Join(appBase, dst)); err != nil {
			return err
		}
	}

	return nil
}

// %applabels
func writeLabels(b *types.Bundle, a *App) error {
	lines := strings.Split(strings.TrimSpace(a.Labels), "\n")
	labels := make(map[string]string)

	// add default label
	labels["SCIF_APP_NAME"] = a.Name

	for _, line := range lines {

		// skip empty or comment lines
		if line = strings.TrimSpace(line); line == "" || strings.Index(line, "#") == 0 {
			continue
		}
		var key, val string
		lineSubs := strings.SplitN(line, " ", 2)
		if len(lineSubs) < 2 {
			key = strings.TrimSpace(lineSubs[0])
			val = ""
		} else {
			key = strings.TrimSpace(lineSubs[0])
			val = strings.TrimSpace(lineSubs[1])
		}

		labels[key] = val
	}

	// make new map into json
	text, err := json.MarshalIndent(labels, "", "\t")
	if err != nil {
		return err
	}

	appBase := filepath.Join(b.Rootfs(), "/scif/apps/", a.Name)
	err = ioutil.WriteFile(filepath.Join(appBase, "scif/labels.json"), text, 0644)
	return err
}

//util funcs

func appBase(b *types.Bundle, a *App) string {
	return filepath.Join(b.Rootfs(), "/scif/apps/", a.Name)
}

func appMeta(b *types.Bundle, a *App) string {
	return filepath.Join(appBase(b, a), "/scif/")
}

func appData(b *types.Bundle, a *App) string {
	return filepath.Join(b.Rootfs(), "/scif/data/", a.Name)
}

func copy(src, dst string) error {
	var stderr bytes.Buffer
	copy := exec.Command("cp", "-fLr", src, dst)
	copy.Stderr = &stderr
	sylog.Debugf("Copying %v to %v", src, dst)
	if err := copy.Run(); err != nil {
		return fmt.Errorf("while copying %v to %v: %v: %v", src, dst, err, stderr.String())
	}

	return nil
}

// HandlePost returns a script that should run after %post
func (pl *BuildApp) HandlePost() string {
	post := ""
	for name, app := range pl.Apps {
		sylog.Debugf("Building app[%s] post script section", name)

		post += buildPost(app)
	}

	return post
}

func buildPost(a *App) string {
	return fmt.Sprintf(scifInstallBase, filepath.Join("/scif/apps/", a.Name), a.Install)
}
