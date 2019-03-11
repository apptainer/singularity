// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package plugin

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"plugin"
	"strings"

	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	pluginapi "github.com/sylabs/singularity/pkg/plugin"
)

const (
	// DirRoot is the root directory for the plugin installation, typically
	// located within LIBEXECDIR.
	DirRoot = "plugin"
	// NameImage is the name of the SIF image of the plugin
	NameImage = "plugin.sif"
	// NameBinary is the name of the plugin object
	NameBinary = "object.so"
	// NameConfig is the name of the plugin's config file
	NameConfig = "config.yaml"
)

// Meta is an internal representation of a plugin binary and all of its
// artifacts. This represents the on-disk location of the SIF, shared library,
// config file, etc... This struct is written as JSON into the DirRoot directory.
type Meta struct {
	// Name is the name of the plugin
	Name string
	// Path is a path, derived from its Name, which the plugins
	// artifacts (config, SIF, .so, etc...) are located
	Path string
	// Enabled reports whether or not the plugin should be loaded
	Enabled bool

	fimg   *sif.FileImage // Plugin SIF object
	binary *plugin.Plugin // Plugin binary object
	cfg    *os.File       // Plugin YAML config file

	file *os.File // Pointer to Meta file on disk, for Read/Write access
}

// LoadFromJSON loads a Meta type from an io.Reader containing JSON. A plugin Meta
// object created in this form is read-only.
func LoadFromJSON(r io.Reader) (*Meta, error) {
	m := &Meta{}

	if err := json.NewDecoder(r).Decode(m); err != nil {
		return nil, fmt.Errorf("while decoding Meta JSON file: %s", err)
	}

	m.cfg, _ = m.Config()

	return m, nil
}

// Config returns the plugin configuration file opened as an os.File object
func (m *Meta) Config() (*os.File, error) {
	if !fs.IsFile(m.configName()) {
		return nil, nil
	}

	return os.Open(m.configName())
}

// InstallFromSIF returns a new meta object which hasn't yet been installed from
// a pointer to an on disk SIF. It will:
//     1. Check that the SIF is a valid plugin
//     2. Open the Manifest to retrieve name and calculate the path
//     3. Copy the SIF into the plugin path
//     4. Extract the binary object into the path
//     5. Generate a default config file in the path
//     6. Write the Meta struct onto disk in DirRoot
func InstallFromSIF(fimg *sif.FileImage, sysconfdir, libexecdir string) (*Meta, error) {
	sylog.Debugf("Installing plugin from SIF to %q", libexecdir)

	if !isPluginFile(fimg) {
		return nil, fmt.Errorf("while opening SIF file: not a valid plugin")
	}

	manifest := getManifest(fimg)

	plugindir := filepath.Join(libexecdir, DirRoot)

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
func Uninstall(name, sysconfdir, libexecdir string) error {
	pluginDir := filepath.Join(libexecdir, DirRoot)
	sylog.Debugf("Uninstalling plugin %q from %q", name, pluginDir)

	meta, err := loadMetaByName(name, pluginDir)
	if err != nil {
		// figure out if this is a not found error?
		return err
	}

	sylog.Debugf("Found plugin %q, meta=%#v", name, meta)
	return meta.uninstall()
}

func loadMetaByName(name, plugindir string) (*Meta, error) {
	fh, err := os.Open(metaPath(plugindir, name))
	if err != nil {
		return nil, err
	}

	defer fh.Close()

	m, err := LoadFromJSON(fh)
	if err != nil {
		return nil, err
	}

	// make sure we loaded the right thing
	if m.Name != name {
		return nil, fmt.Errorf("Unexpected plugin name %q when loading plugin %q", m.Name, name)
	}

	return m, nil
}

// install installs the plugin represented by m into the destination
// directory. This should normally only be called in InstallFromSIF
func (m *Meta) install(dstdir string) error {
	if err := os.MkdirAll(m.Path, 0777); err != nil {
		return err
	}

	if err := m.installImage(); err != nil {
		return err
	}

	if err := m.installBinary(); err != nil {
		return err
	}

	if err := m.installMeta(dstdir); err != nil {
		return err
	}

	return nil
}

func (m *Meta) installImage() error {
	fh, err := os.Create(m.imageName())
	if err != nil {
		return err
	}

	defer fh.Close()

	_, err = fh.Write(m.fimg.Filedata)

	return err
}

func (m *Meta) installBinary() error {
	fh, err := os.Create(m.binaryName())
	if err != nil {
		return err
	}

	defer fh.Close()

	start := m.fimg.DescrArr[0].Fileoff
	end := start + m.fimg.DescrArr[0].Filelen
	_, err = fh.Write(m.fimg.Filedata[start:end])

	return err
}

func (m *Meta) installMeta(dstdir string) error {
	fn := metaPath(dstdir, m.Name)
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
	// in this function we cannot bail out on error because we need
	// to clean up as much as possible, so collect all the errors
	// that happen along the way.

	var errs []error

	if err := m.uninstallImage(); err != nil {
		errs = append(errs, err)
	}

	if err := m.uninstallBinary(); err != nil {
		errs = append(errs, err)
	}

	if err := m.uninstallMeta(); err != nil {
		errs = append(errs, err)
	}

	baseDir := m.baseDir()
	for dir := m.Path; dir != baseDir && dir != "/"; dir = filepath.Dir(dir) {
		sylog.Debugf("Removing directory %q", dir)
		if err := os.Remove(dir); err != nil {
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

func (m *Meta) uninstallImage() error {
	return os.Remove(m.imageName())
}

func (m *Meta) uninstallBinary() error {
	return os.Remove(m.binaryName())
}

func (m *Meta) uninstallMeta() error {
	fn := metaPath(m.baseDir(), m.Name)
	return os.Remove(fn)
}

// metaPath returns the path to the meta file based on the directory and
// the name of the corresponding plugin
func metaPath(dir, name string) string {
	return filepath.Join(dir, pluginIDFromName(name)+".meta")
}

func (m *Meta) baseDir() string {
	// figure out the location where the .meta file should be by
	// removing the name of the plugin from the installation path.
	//
	// the other option is actually walking up m.Path looking for
	// the .meta file, but that's expensive because it would have to
	// perform a whole bunch of stat calls looking for the file.
	return filepath.Clean(strings.TrimSuffix(m.Path, pathFromName(m.Name)))
}

//
// Misc helper functions
//

// pathFromName returns a partial path for the plugin relative to the
// plugin installation directory
func pathFromName(name string) string {
	return filepath.FromSlash(name)
}

// pluginIDFromName returns a unique ID for the plugin given its name
func pluginIDFromName(name string) string {
	sum := sha256.Sum256([]byte(name))
	return fmt.Sprintf("%x", sum)
}

//
// Path name helper methods on (m *Meta)
//

func (m *Meta) imageName() string {
	return filepath.Join(m.Path, NameImage)
}

func (m *Meta) binaryName() string {
	return filepath.Join(m.Path, NameBinary)
}

func (m *Meta) configName() string {
	return filepath.Join(m.Path, NameConfig)
}

//
// Helper functions for fimg *sif.FileImage
//

// isPluginFile checks if the sif.FileImage contains the sections which
// make up a valid plugin. A plugin sif file should have the following
// format:
//
// DESCR[0]: Sifplugin
//   - Datatype: sif.DataPartition
//   - Fstype:   sif.FsRaw
//   - Parttype: sif.PartData
// DESCR[1]: Sifmanifest
//   - Datatype: sif.DataGenericJSON
func isPluginFile(fimg *sif.FileImage) bool {
	if len(fimg.DescrArr) < 2 {
		return false
	}

	if !fimg.DescrArr[0].Used {
		return false
	}

	if fimg.DescrArr[0].Datatype != sif.DataPartition {
		return false
	}

	if fstype, err := fimg.DescrArr[0].GetFsType(); err != nil {
		return false
	} else if fstype != sif.FsRaw {
		return false
	}

	if partype, err := fimg.DescrArr[0].GetPartType(); err != nil {
		return false
	} else if partype != sif.PartData {
		return false
	}

	if !fimg.DescrArr[1].Used {
		return false
	}

	if fimg.DescrArr[1].Datatype != sif.DataGenericJSON {
		return false
	}

	return true
}

// getManifest will extract the Manifest data from the input FileImage
func getManifest(fimg *sif.FileImage) pluginapi.Manifest {
	var (
		manifest pluginapi.Manifest
		start    = fimg.DescrArr[1].Fileoff
		end      = start + fimg.DescrArr[1].Filelen
		data     = fimg.Filedata[start:end]
	)

	if err := json.Unmarshal(data, &manifest); err != nil {
		fmt.Println(err)
	}

	return manifest
}

// List returns all the singularity plugins installed in libexecdir in
// the form of a list of Meta information
func List(sysconfdir, libexecdir string) ([]*Meta, error) {
	pluginDir := filepath.Join(libexecdir, DirRoot)
	pattern := filepath.Join(pluginDir, "*.meta")
	entries, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("Cannot list plugins in directory %q", pluginDir)
	}

	metas := []*Meta{}

	for _, entry := range entries {
		fi, err := os.Stat(entry)
		if err != nil {
			sylog.Debugf("Error stating %s: %s. Skip\n", entry, err)
			continue
		}

		if !fi.Mode().IsRegular() {
			continue
		}

		readMeta := func(name string) *Meta {
			fh, err := os.Open(name)
			if err != nil {
				sylog.Debugf("Error opening %s: %s. Skip\n", name, err)
				return nil
			}
			defer fh.Close()

			meta, err := LoadFromJSON(fh)
			if err != nil {
				sylog.Debugf("Error loading %s: %s. Skip\n", name, err)
				return nil
			}

			return meta
		}

		if meta := readMeta(entry); meta != nil {
			metas = append(metas, meta)
		}
	}

	return metas, nil
}
