// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package plugin

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sylabs/singularity/pkg/image"
	pluginapi "github.com/sylabs/singularity/pkg/plugin"
	"github.com/sylabs/singularity/pkg/sylog"
)

// Install installs a plugin from a SIF image under rootDir. It will:
//     1. Check that the SIF is a valid plugin
//     2. Use name from Manifest and calculate the installation path
//     3. Copy the SIF into the plugin path
//     4. Extract the binary object into the path
//     5. Generate a default config file in the path
//     6. Write the Meta struct onto disk in dirRoot
func Install(sifPath string) error {
	sylog.Debugf("Installing plugin from SIF to %q", rootDir)

	img, err := image.Init(sifPath, false)
	if err != nil {
		return fmt.Errorf("could not load plugin: %w", err)
	} else if !isPluginFile(img) {
		return fmt.Errorf("%s is not a valid plugin", sifPath)
	}

	manifest, err := getManifest(img)
	if err != nil {
		return fmt.Errorf("could not get manifest: %s", err)
	} else if manifest.Name == "" {
		return fmt.Errorf("empty plugin in manifest")
	}

	// as the name determine the path inside the plugin root
	// directory, we first ensure that the name doesn't trick us
	// with a path traversal
	cleanName := filepath.Join("/", filepath.Clean(manifest.Name))
	if manifest.Name[0] != '/' {
		cleanName = cleanName[1:]
	}
	if cleanName != manifest.Name {
		return fmt.Errorf("plugin manifest name %q contains path traversal", manifest.Name)
	}

	m := &Meta{
		Name:    manifest.Name,
		Enabled: true,
	}

	err = m.install(img)
	if err != nil {
		return fmt.Errorf("could not install plugin: %w", err)
	}
	return nil
}

// Uninstall removes the plugin matching "name" from the singularity
// plugin installation directory.
func Uninstall(name string) error {
	sylog.Debugf("Uninstalling plugin %q from %q", name, rootDir)

	meta, err := loadMetaByName(name)
	if err != nil {
		return err
	}

	sylog.Debugf("Found plugin %q, meta=%#v", name, meta)

	return meta.uninstall()
}

// List returns all the singularity plugins installed in
// rootDir in the form of a list of Meta information.
func List() ([]*Meta, error) {
	pattern := filepath.Join(rootDir, "*.meta")
	entries, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("cannot list plugins in directory %q", rootDir)
	}

	var metas []*Meta
	for _, entry := range entries {
		fi, err := os.Stat(entry)
		if err != nil {
			sylog.Debugf("Error stating %s: %s. Skip\n", entry, err)
			continue
		}

		if !fi.Mode().IsRegular() {
			continue
		}

		meta, err := loadMetaByFilename(entry)
		if err != nil {
			sylog.Debugf("Error loading %s: %s. Skip", entry, err)
			continue
		}

		metas = append(metas, meta)
	}

	return metas, nil
}

// Enable enables the plugin named "name" found under rootDir.
func Enable(name string) error {
	sylog.Debugf("Enabling plugin %q in %q", name, rootDir)

	meta, err := loadMetaByName(name)
	if err != nil {
		return err
	}

	sylog.Debugf("Found plugin %q, meta=%#v", name, meta)

	if meta.Enabled {
		sylog.Infof("Plugin %q is already enabled", name)
		return nil
	}

	return meta.enable()
}

// Disable disables the plugin named "name" found under rootDir.
func Disable(name string) error {
	sylog.Debugf("Disabling plugin %q in %q", name, rootDir)

	meta, err := loadMetaByName(name)
	if err != nil {
		return err
	}

	sylog.Debugf("Found plugin %q, meta=%#v", name, meta)

	if !meta.Enabled {
		sylog.Infof("Plugin %q is already disabled", name)
		return nil
	}

	return meta.disable()
}

// Inspect obtains information about the plugin "name".
//
// "name" can be either the name of plugin installed under rootDir
// or the name of an image file corresponding to a plugin.
func Inspect(name string) (pluginapi.Manifest, error) {
	var manifest pluginapi.Manifest

	// LoadContainer returns a decorated error, no it's not possible
	// to ask whether the error happens because the file does not
	// exist or something else. Check for the file _before_ trying
	// to load it as a container.
	_, err := os.Stat(name)
	if err != nil {
		if !os.IsNotExist(err) {
			// There seems to be a file here, but we cannot
			// read it.
			return manifest, err
		}

		// no file, try to find the installed plugin
		meta, err := loadMetaByName(name)
		if err != nil {
			// Metafile not found, or we cannot read
			// it. There's nothing we can do.
			return manifest, err
		}

		// Replace the original name, which seems to be
		// the name of a plugin, by the path to the
		// installed manifest file for that plugin.
		data, err := ioutil.ReadFile(meta.manifestName())
		if err != nil {
			return manifest, err
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			return manifest, err
		}
	} else {
		// at this point, either the file is there under the original
		// name or we found one by looking at the metafile.
		img, err := image.Init(name, false)
		if err != nil {
			return manifest, fmt.Errorf("could not load plugin: %w", err)
		} else if !isPluginFile(img) {
			return manifest, fmt.Errorf("%s is not a valid plugin", name)
		}
		return getManifest(img)
	}

	return manifest, nil
}

//
// Misc helper functions
//

// pathFromName returns a partial path for the plugin
// relative to the plugin installation directory.
func pathFromName(name string) string {
	return filepath.FromSlash(name)
}

// pluginIDFromName returns a unique ID for the plugin given its name.
func pluginIDFromName(name string) string {
	sum := sha256.Sum256([]byte(name))
	return fmt.Sprintf("%x", sum)
}
