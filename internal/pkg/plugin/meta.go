// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/plugin/callback"
	"github.com/sylabs/singularity/pkg/image"
	"github.com/sylabs/singularity/pkg/sylog"
)

const (
	// rootDir is the root directory for the plugin
	// installation, typically located within LIBEXECDIR.
	rootDir = buildcfg.PLUGIN_ROOTDIR
	// nameImage is the name of the SIF image of the plugin
	nameManifest = "object.manifest"
	// nameBinary is the name of the plugin object
	nameBinary = "object.so"
)

// Meta is an internal representation of a plugin binary
// and all of its artifacts. This represents the on-disk
// location of the SIF, shared library, config file, etc...
// This struct is written as JSON into the rootDir directory.
type Meta struct {
	// Name is the name of the plugin.
	Name string
	// Enabled reports whether or not the plugin should be loaded.
	Enabled bool
	// Callbacks contains callbacks name registered by the plugin.
	Callbacks []string
}

// loadFromJSON loads a Meta type from an io.Reader containing
// JSON. A plugin Meta object created in this form is read-only.
func loadFromJSON(r io.Reader) (*Meta, error) {
	var m Meta
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		return nil, fmt.Errorf("could not decode meta: %s", err)
	}

	return &m, nil
}

func loadMetaByName(name string) (*Meta, error) {
	m, err := loadMetaByFilename(metaPath(name))
	if err != nil {
		return nil, err
	}

	// make sure we loaded the right thing
	if m.Name != name {
		return nil, fmt.Errorf("unexpected plugin name %q when loading plugin %q", m.Name, name)
	}

	return m, nil
}

func loadMetaByFilename(filename string) (*Meta, error) {
	fh, err := os.Open(filename)
	if err != nil {
		sylog.Debugf("Error opening meta file %q: %s", filename, err)
		return nil, err
	}
	defer fh.Close()

	return loadFromJSON(fh)
}

// metaPath returns the path to the meta file based on the
// the name of the corresponding plugin.
func metaPath(name string) string {
	return filepath.Join(rootDir, pluginIDFromName(name)+".meta")
}

// install installs the plugin represented by m into the plugin installation
// directory. This should normally only be called in InstallFromSIF.
func (m *Meta) install(img *image.Image) error {
	if err := os.MkdirAll(m.path(), 0755); err != nil {
		return err
	}

	if err := m.installBinary(img); err != nil {
		return err
	}
	if err := m.installManifest(img); err != nil {
		return err
	}

	// must be called before installMeta to also
	// get plugin callbacks name
	if err := m.runInstall(); err != nil {
		return err
	}

	if err := m.installMeta(); err != nil {
		return err
	}

	return nil
}

func (m *Meta) installBinary(img *image.Image) error {
	fh, err := os.Create(m.binaryName())
	if err != nil {
		return err
	}
	defer fh.Close()

	r, err := getBinaryReader(img)
	if err != nil {
		return err
	}

	_, err = io.Copy(fh, r)
	return err
}

func (m *Meta) installManifest(img *image.Image) error {
	fh, err := os.Create(m.manifestName())
	if err != nil {
		return err
	}
	defer fh.Close()

	r, err := getManifestReader(img)
	if err != nil {
		return err
	}

	_, err = io.Copy(fh, r)
	return err
}

func (m *Meta) runInstall() error {
	binary := m.binaryName()

	pl, err := LoadObject(binary)
	if err != nil {
		return fmt.Errorf("while loading plugin %s: %s", binary, err)
	}

	if pl.Install != nil {
		if err := pl.Install(m.path()); err != nil {
			return fmt.Errorf("while running plugin Install: %s", err)
		}
	}

	m.Callbacks = callback.Names(pl.Callbacks)

	return nil
}

func (m *Meta) installMeta() error {
	fn := metaPath(m.Name)

	fh, err := os.Create(fn)
	if err != nil {
		return err
	}
	defer fh.Close()

	data, err := json.Marshal(m)
	if err != nil {
		return err
	}

	_, err = fh.Write(data)
	if err != nil {
		return err
	}

	return nil
}

// uninstall removes the plugin it represents from the filesystem.
func (m *Meta) uninstall() error {
	// in this function we cannot fail out on error because
	// we need to clean up as much as possible, so collect
	// all the errors that happen along the way.
	var errs []error

	// remove meta file first and only after we remove
	// plugin directory
	if err := m.uninstallMeta(); err != nil {
		errs = append(errs, err)
	}

	if err := m.removeDir(); err != nil {
		errs = append(errs, err)
	}

	for dir := filepath.Dir(m.Name); dir != "."; dir = filepath.Dir(dir) {
		d := filepath.Join(rootDir, dir)
		sylog.Debugf("Removing directory %q", d)
		if err := os.Remove(d); err != nil {
			// directory is not empty, stop here
			if os.IsExist(err) {
				sylog.Debugf("Directory %q wasn't empty", d)
				break
			}
			errs = append(errs, err)
			break
		}
	}

	switch len(errs) {
	case 0:
		return nil

	case 1:
		return errs[0]

	default:
		// Transform all the errors into a single error. This
		// might be destroying information by grabbing only the
		// textual description of the error. The alternative is
		// to implement an special type that implements Error()
		// in the same way, and offers the option of examining
		// all the errors one by one, but at the moment that's
		// not needed.
		var b strings.Builder
		for i, err := range errs {
			if i > 0 {
				b.WriteString("; ")
			}
			b.WriteString(err.Error())
		}
		return errors.New(b.String())
	}
}

func (m *Meta) removeDir() error {
	if _, err := os.Stat(m.binaryName()); err != nil {
		return err
	}
	if _, err := os.Stat(m.manifestName()); err != nil {
		return err
	}
	return os.RemoveAll(m.path())
}

func (m *Meta) uninstallMeta() error {
	return os.Remove(metaPath(m.Name))
}

func (m *Meta) enable() error {
	m.Enabled = true
	return m.installMeta()
}

func (m *Meta) disable() error {
	m.Enabled = false
	return m.installMeta()
}

//
// Path name helper methods on (m *Meta)
//

func (m *Meta) manifestName() string {
	return filepath.Join(m.path(), nameManifest)
}

func (m *Meta) binaryName() string {
	return filepath.Join(m.path(), nameBinary)
}

func (m *Meta) path() string {
	return filepath.Join(rootDir, pathFromName(m.Name))
}
