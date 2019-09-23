package plugin

import (
	"encoding/json"

	"github.com/sylabs/singularity/internal/pkg/sylog"

	"github.com/sylabs/sif/pkg/sif"
	pluginapi "github.com/sylabs/singularity/pkg/plugin"
)

const (
	// pluginBinaryName is the name of the plugin binary within the
	// SIF file
	pluginBinaryName = "plugin.so"
	// pluginManifestName is the name of the plugin manifest within
	// the SIF file
	pluginManifestName = "plugin.manifest"
)

// sifReader defines helper functions fimg *sif.FileImage.
type sifReader interface {
	Descriptors() int
	IsUsed(name string) bool
	GetDatatype(name string) sif.Datatype
	GetFsType(name string) (sif.Fstype, error)
	GetPartType(name string) (sif.Parttype, error)
	GetData(name string) []byte
}

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
func isPluginFile(fimg sifReader) bool {
	if fimg.Descriptors() < 2 {
		return false
	}

	if !fimg.IsUsed(pluginBinaryName) {
		return false
	}

	if fimg.GetDatatype(pluginBinaryName) != sif.DataPartition {
		return false
	}

	if fstype, err := fimg.GetFsType(pluginBinaryName); err != nil {
		return false
	} else if fstype != sif.FsRaw {
		return false
	}

	if partype, err := fimg.GetPartType(pluginBinaryName); err != nil {
		return false
	} else if partype != sif.PartData {
		return false
	}

	if !fimg.IsUsed(pluginManifestName) {
		return false
	}

	if fimg.GetDatatype(pluginManifestName) != sif.DataGenericJSON {
		return false
	}

	return true
}

// getManifest will extract the Manifest data from the input FileImage.
func getManifest(fimg sifReader) pluginapi.Manifest {
	if fimg.Descriptors() < 2 || !fimg.IsUsed(pluginManifestName) {
		return pluginapi.Manifest{}
	}

	data := fimg.GetData(pluginManifestName)
	if data == nil {
		return pluginapi.Manifest{}
	}

	var manifest pluginapi.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		sylog.Errorf("Could not unmarshal manifest: %v", err)
		return pluginapi.Manifest{}
	}

	return manifest
}

type sifFileImageReader struct {
	fi          *sif.FileImage
	descriptors map[string]int
}

func newSifFileImageReader(fi *sif.FileImage) *sifFileImageReader {
	r := &sifFileImageReader{fi: fi, descriptors: make(map[string]int)}
	for n, desc := range fi.DescrArr {
		if !desc.Used {
			continue
		}
		r.descriptors[fi.DescrArr[n].GetName()] = n
	}
	return r
}

func (r *sifFileImageReader) Descriptors() int {
	return len(r.fi.DescrArr)
}

func (r *sifFileImageReader) IsUsed(name string) bool {
	n := r.descriptors[name]
	return r.fi.DescrArr[n].Used
}

func (r *sifFileImageReader) GetDatatype(name string) sif.Datatype {
	n := r.descriptors[name]
	return r.fi.DescrArr[n].Datatype
}

func (r *sifFileImageReader) GetFsType(name string) (sif.Fstype, error) {
	n := r.descriptors[name]
	return r.fi.DescrArr[n].GetFsType()
}

func (r *sifFileImageReader) GetPartType(name string) (sif.Parttype, error) {
	n := r.descriptors[name]
	return r.fi.DescrArr[n].GetPartType()
}

func (r *sifFileImageReader) GetData(name string) []byte {
	var (
		n     = r.descriptors[name]
		start = r.fi.DescrArr[n].Fileoff
		end   = start + r.fi.DescrArr[n].Filelen
		data  = r.fi.Filedata[start:end]
	)

	return data
}
