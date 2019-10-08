// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package plugin

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	pluginapi "github.com/sylabs/singularity/pkg/plugin"
)

// InstallFromSIF returns a new meta object which hasn't yet been installed from
// a pointer to an on disk SIF. It will:
//     1. Check that the SIF is a valid plugin
//     2. Open the Manifest to retrieve name and calculate the path
//     3. Copy the SIF into the plugin path
//     4. Extract the binary object into the path
//     5. Generate a default config file in the path
//     6. Write the Meta struct onto disk in dirRoot
func InstallFromSIF(fimg *sif.FileImage, libexecdir string) (*Meta, error) {
	sylog.Debugf("Installing plugin from SIF to %q", libexecdir)

	sr := newSifFileImageReader(fimg)

	if !isPluginFile(sr) {
		return nil, fmt.Errorf("while opening SIF file: not a valid plugin")
	}

	manifest := getManifest(sr)

	plugindir := filepath.Join(libexecdir, dirRoot)

	dstdir, err := filepath.Abs(filepath.Join(plugindir, pathFromName(manifest.Name)))
	if err != nil {
		return nil, fmt.Errorf("while getting absolute path to plugin installation: %s", err)
	}

	m := &Meta{
		Name:    manifest.Name,
		Path:    dstdir,
		Enabled: true,

		fimg: fimg,
	}

	err = m.install(plugindir)
	return m, err
}

// Uninstall removes the plugin matching "name" from the specified
// singularity installation directory
func Uninstall(name, libexecdir string) error {
	pluginDir := filepath.Join(libexecdir, dirRoot)
	sylog.Debugf("Uninstalling plugin %q from %q", name, pluginDir)

	meta, err := loadMetaByName(name, pluginDir)
	if err != nil {
		return err
	}

	sylog.Debugf("Found plugin %q, meta=%#v", name, meta)

	return meta.uninstall()
}

// List returns all the singularity plugins installed in
// libexecdir in the form of a list of Meta information.
func List(libexecdir string) ([]*Meta, error) {
	pluginDir := filepath.Join(libexecdir, dirRoot)
	pattern := filepath.Join(pluginDir, "*.meta")
	entries, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("cannot list plugins in directory %q", pluginDir)
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

// Enable enables the plugin named "name" found under "libexecdir".
func Enable(name, libexecdir string) error {
	pluginDir := filepath.Join(libexecdir, dirRoot)
	sylog.Debugf("Enabling plugin %q in %q", name, pluginDir)

	meta, err := loadMetaByName(name, pluginDir)
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

// Disable disables the plugin named "name" found under "libexecdir"
func Disable(name, libexecdir string) error {
	pluginDir := filepath.Join(libexecdir, dirRoot)
	sylog.Debugf("Disabling plugin %q in %q", name, pluginDir)

	meta, err := loadMetaByName(name, pluginDir)
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
// "name" can be either the name of plugin installed under "libexecdir"
// or the name of an image file corresponding to a plugin.
func Inspect(name, libexecdir string) (pluginapi.Manifest, error) {
	var manifest pluginapi.Manifest

	// LoadContainer returns a decorated error, no it's not possible
	// to ask whether the error happens because the file does not
	// exist or something else. Check for the file _before_ trying
	// to load it as a container.
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			// no file, try to find the installed plugin
			pluginDir := filepath.Join(libexecdir, dirRoot)
			meta, err := loadMetaByName(name, pluginDir)
			if err != nil {
				// Metafile not found, or we cannot read
				// it. There's nothing we can do.
				return manifest, err
			}

			// Replace the original name, which seems to be
			// the name of a plugin, by the path to the
			// installed SIF file for that plugin.
			name = meta.imageName()
		} else {
			// There seems to be a file here, but we cannot
			// read it.
			return manifest, err
		}
	}

	// at this point, either the file is there under the original
	// name or we found one by looking at the metafile.
	fimg, err := sif.LoadContainer(name, true)
	if err != nil {
		return manifest, err
	}

	defer fimg.UnloadContainer()

	r := newSifFileImageReader(&fimg)

	if !isPluginFile(r) {
		return manifest, fmt.Errorf("while opening SIF file: not a valid plugin")
	}

	manifest = getManifest(r)

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
