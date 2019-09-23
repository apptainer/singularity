package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/pkg/plugin"
)

const (
	// dirRoot is the root directory for the plugin
	// installation, typically located within LIBEXECDIR.
	dirRoot = "singularity/plugin"
	// nameImage is the name of the SIF image of the plugin
	nameImage = "plugin.sif"
	// nameBinary is the name of the plugin object
	nameBinary = "object.so"
	// nameConfig is the name of the plugin's config file
	nameConfig = "config.yaml"
)

// Meta is an internal representation of a plugin binary
// and all of its artifacts. This represents the on-disk
// location of the SIF, shared library, config file, etc...
// This struct is written as JSON into the dirRoot directory.
type Meta struct {
	// Name is the name of the plugin.
	Name string
	// Path is a path, derived from its Name, where the plugin
	// artifacts (config, SIF, .so, etc...) are located.
	Path string
	// Enabled reports whether or not the plugin should be loaded.
	Enabled bool

	fimg   *sif.FileImage // plugin SIF object.
	binary *plugin.Plugin // plugin binary object.
	cfg    *os.File       // plugin YAML config file.

	file *os.File // pointer to Meta file on disk, for Read/Write access.
}

// loadFromJSON loads a Meta type from an io.Reader containing
// JSON. A plugin Meta object created in this form is read-only.
func loadFromJSON(r io.Reader) (*Meta, error) {
	var m Meta
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		return nil, fmt.Errorf("could not decode meta: %s", err)
	}

	var err error
	m.cfg, err = m.config()
	if err != nil {
		return nil, fmt.Errorf("could not get config: %v", err)
	}

	return &m, nil
}

func loadMetaByName(name, plugindir string) (*Meta, error) {
	m, err := loadMetaByFilename(metaPath(plugindir, name))
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
// directory and the name of the corresponding plugin.
func metaPath(dir, name string) string {
	return filepath.Join(dir, pluginIDFromName(name)+".meta")
}

// config returns the plugin configuration file opened as an os.File object.
func (m *Meta) config() (*os.File, error) {
	if !fs.IsFile(m.configName()) {
		return nil, nil
	}

	return os.Open(m.configName())
}

// install installs the plugin represented by m into the destination
// directory. This should normally only be called in InstallFromSIF.
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
	// in this function we cannot fail out on error because
	// we need to clean up as much as possible, so collect
	// all the errors that happen along the way.
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

func (m *Meta) enable() error {
	m.Enabled = true
	return m.installMeta(m.baseDir())
}

func (m *Meta) disable() error {
	m.Enabled = false
	return m.installMeta(m.baseDir())
}

//
// Path name helper methods on (m *Meta)
//

func (m *Meta) imageName() string {
	return filepath.Join(m.Path, nameImage)
}

func (m *Meta) binaryName() string {
	return filepath.Join(m.Path, nameBinary)
}

func (m *Meta) configName() string {
	return filepath.Join(m.Path, nameConfig)
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
