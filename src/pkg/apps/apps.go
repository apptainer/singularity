// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

// Package apps provides the functions which are necessary for adding SCI-F apps support
// to Singularity 3.0.0. In 3.1.0+, this package will be able to be built standalone as
// a plugin so it will be maintainable separately from the core Singularity functionality
package apps

import (
	"os"
	"strings"
	"sync"

	"github.com/sylabs/singularity/src/pkg/sylog"
	"github.com/sylabs/singularity/src/pkg/syplugin"
)

const name = "singularity_apps"

const (
	sectionInstall = "appinstall"
	sectionFiles   = "appfiles"
	sectionEnv     = "appenv"
	sectionTest    = "apptest"
	sectionHelp    = "apphelp"
)

// App represents a Singularity app at build time
type App struct {
	Install string
	Files   string
	Env     string
	Test    string
	Help    string
}

// BuildPlugin is the type which the build system can understand
type BuildPlugin struct {
	Apps map[string]App `json:"appsDefined"`
	*sync.Mutex
}

func init() {
	if err := syplugin.RegisterBuildPlugin(BuildPlugin{
		Apps:  make(map[string]App),
		Mutex: &sync.Mutex{},
	}); err != nil {
		os.Exit(1)
	}
}

// Name returns this handler's name [singularity_apps]
func (pl BuildPlugin) Name() string {
	return name
}

// HandleSection receives a string of each section from the deffile
func (pl BuildPlugin) HandleSection(ident, section string) {
	name, sect := getAppAndSection(ident)
	app := *(pl.initApp(name))

	switch sect {
	case sectionInstall:
		app.Install = sect
	case sectionFiles:
		app.Files = sect
	case sectionEnv:
		app.Env = sect
	case sectionTest:
		app.Test = sect
	case sectionHelp:
		app.Help = sect
	default:
		return
	}

	sylog.Debugf("App: %s - Section: %s\n", name, sect)
}

func (pl BuildPlugin) initApp(name string) *App {
	pl.Lock()
	defer pl.Unlock()

	app, ok := pl.Apps[name]
	if !ok {
		pl.Apps[name] = App{}
	}

	return &app
}

// getAppAndSection returns the app name and section name from the header of the section:
//     %SECTION APP ... returns APP, SECTION
func getAppAndSection(ident string) (appName string, sectionName string) {
	identSplit := strings.Split(ident, " ")

	if len(identSplit) < 2 {
		return "", ""
	}

	return identSplit[1], identSplit[0]
}

// func createBase(b *types.Bundle) {
// }

// func createAppBase(b *types.Bundle, name string) {

// }

// // OnPost hook allows custom code to be run after %post script is done running. This happens in
// // the same environment as %post
// func OnPost() {

// }

// // OnPre hook allows custom code to be run after %pre script is done running. This happens in
// // the same environment as %pre
// func OnPre() {

// }

// // OnSetup hook allows custom code to be run after %setup script is done running. This happens in
// // the same environment as %setup
// func OnSetup() {

// }
