// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

// Package apps [apps-plugin] provides the functions which are necessary for adding SCI-F apps support
// to Singularity 3.0.0. In 3.1.0+, this package will be able to be built standalone as
// a plugin so it will be maintainable separately from the core Singularity functionality
package apps

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sylabs/singularity/internal/pkg/build/types"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

const name = "singularity_apps"

const (
	sectionInstall = "appinstall"
	sectionFiles   = "appfiles"
	sectionEnv     = "appenv"
	sectionTest    = "apptest"
	sectionHelp    = "apphelp"
	sectionRun     = "apprun"
)

var (
	sections = map[string]bool{
		sectionInstall: true,
		sectionFiles:   true,
		sectionEnv:     true,
		sectionTest:    true,
		sectionHelp:    true,
		sectionRun:     true,
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

	scifInstallBase = `
cd /
. %[1]s/env/01-base.sh

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
}

// BuildPlugin is the type which the build system can understand
type BuildPlugin struct {
	Apps map[string]*App `json:"appsDefined"`
	sync.Mutex
}

// New returns a new BuildPlugin for the plugin registry to hold
func New() interface{} {
	return &BuildPlugin{
		Apps: make(map[string]*App),
	}

}

// Name returns this handler's name [singularity_apps]
func (pl *BuildPlugin) Name() string {
	return name
}

// HandleSection receives a string of each section from the deffile
func (pl *BuildPlugin) HandleSection(ident, section string) {
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
	default:
		return
	}
}

func (pl *BuildPlugin) initApp(name string) {
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
func (pl *BuildPlugin) HandleBundle(b *types.Bundle) {
	if err := pl.createAllApps(b); err != nil {
		sylog.Fatalf("Unable to create apps: %s", err)
	}
}

func (pl *BuildPlugin) createAllApps(b *types.Bundle) error {
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

		if err := writeHelpFile(b, app); err != nil {
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

// %apphelp
func writeHelpFile(b *types.Bundle, a *App) error {
	if a.Help == "" {
		return nil
	}

	return ioutil.WriteFile(filepath.Join(appMeta(b, a), "/runscript.help"), []byte(a.Help), 0755)
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

// HandlePost returns a script that should run after %post
func (pl *BuildPlugin) HandlePost() string {
	post := ""
	for name, app := range pl.Apps {
		sylog.Debugf("Building app[%s] post script section", name)

		post += buildPost(app)
	}

	return post
}

func buildPost(a *App) string {
	return fmt.Sprintf(scifInstallBase, filepath.Join("/scif/apps/", a.Name, "/scif"), a.Install)
}
